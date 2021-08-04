package virtualmachine

import (
	"encoding/json"
	"fmt"
	"strings"

	v1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/slice"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	kv1 "kubevirt.io/client-go/api/v1"

	"github.com/harvester/harvester/pkg/indexeres"
	"github.com/harvester/harvester/pkg/ref"
	"github.com/harvester/harvester/pkg/util"
)

type VMController struct {
	pvcClient v1.PersistentVolumeClaimClient
	pvcCache  v1.PersistentVolumeClaimCache
}

// createPvcFromAnnotation creates PVCs defined in the volumeClaimTemplates annotation if they don't exist.
func (h *VMController) createPvcFromAnnotation(_ string, vm *kv1.VirtualMachine) (*kv1.VirtualMachine, error) {
	if vm == nil || vm.DeletionTimestamp != nil {
		return nil, nil
	}
	volumeClaimTemplates, ok := vm.Annotations[util.AnnotationVolumeClaimTemplates]
	if !ok || volumeClaimTemplates == "" {
		return nil, nil
	}
	var pvcs []*corev1.PersistentVolumeClaim
	if err := json.Unmarshal([]byte(volumeClaimTemplates), &pvcs); err != nil {
		return nil, err
	}
	for _, pvc := range pvcs {
		pvc.Namespace = vm.Namespace
		if _, err := h.pvcCache.Get(vm.Namespace, pvc.Name); apierrors.IsNotFound(err) {
			if _, err := h.pvcClient.Create(pvc); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// SetOwnerOfPvcs records the target VirtualMachine as the owner of the PVCs in annotation.
func (h *VMController) SetOwnerOfPvcs(_ string, vm *kv1.VirtualMachine) (*kv1.VirtualMachine, error) {
	if vm == nil || vm.DeletionTimestamp != nil || vm.Spec.Template == nil {
		return vm, nil
	}

	pvcNames := getPvcNames(&vm.Spec.Template.Spec)

	vmReferenceKey := ref.Construct(vm.Namespace, vm.Name)
	// get all the volumes attached to this virtual machine
	attachedPvcs, err := h.pvcCache.GetByIndex(indexeres.PvcByVmIndex, vmReferenceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get attached PVCs by VM index: %w", err)
	}

	vmGVK := kv1.VirtualMachineGroupVersionKind
	vmAPIVersion, vmKind := vmGVK.ToAPIVersionAndKind()
	vmGK := vmGVK.GroupKind()

	for _, attachedPvc := range attachedPvcs {
		// check if it's still attached
		if pvcNames.Has(attachedPvc.Name) {
			continue
		}

		toUpdate := attachedPvc.DeepCopy()

		// if this volume is no longer attached
		// remove volume's annotation
		owners, err := ref.GetSchemaOwnersFromAnnotation(attachedPvc)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema owners from annotation: %w", err)
		}

		isAttached := owners.Remove(vmGK, vm)
		if isAttached {
			if err := owners.Bind(toUpdate); err != nil {
				return nil, fmt.Errorf("failed to apply schema owners to annotation: %w", err)
			}
		}

		// remove volume's ownerReferences
		isOwned := false
		ownerReferences := make([]metav1.OwnerReference, 0, len(attachedPvc.OwnerReferences))
		for _, reference := range attachedPvc.OwnerReferences {
			if reference.APIVersion == vmAPIVersion && reference.Kind == vmKind && reference.Name == vm.Name {
				isOwned = true
				continue
			}
			ownerReferences = append(ownerReferences, reference)
		}
		if isOwned {
			if len(ownerReferences) == 0 {
				ownerReferences = nil
			}
			toUpdate.OwnerReferences = ownerReferences
		}

		// update volume
		if isAttached || isOwned {
			if _, err = h.pvcClient.Update(toUpdate); err != nil {
				return nil, fmt.Errorf("failed to clean schema owners for PVC(%s/%s): %w",
					attachedPvc.Namespace, attachedPvc.Name, err)
			}
		}
	}

	var pvcNamespace = vm.Namespace
	for _, pvcName := range pvcNames.List() {
		var pvc, err = h.pvcCache.Get(pvcNamespace, pvcName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// ignores not-found error, A VM can reference a non-existent PVC.
				continue
			}
			return vm, fmt.Errorf("failed to get PVC(%s/%s): %w", pvcNamespace, pvcName, err)
		}

		err = setOwnerlessPvcReference(h.pvcClient, pvc, vm)
		if err != nil {
			return vm, fmt.Errorf("failed to grant VitrualMachine(%s/%s) as PVC(%s/%s)'s owner: %w",
				vm.Namespace, vm.Name, pvcNamespace, pvcName, err)
		}
	}

	return vm, nil
}

// UnsetOwnerOfPvcs erases the target VirtualMachine from the owner of the PVCs in annotation.
func (h *VMController) UnsetOwnerOfPvcs(_ string, vm *kv1.VirtualMachine) (*kv1.VirtualMachine, error) {
	if vm == nil || vm.DeletionTimestamp == nil || vm.Spec.Template == nil {
		return vm, nil
	}

	var (
		pvcNamespace = vm.Namespace
		pvcNames     = getPvcNames(&vm.Spec.Template.Spec)
		removedPvcs  = getRemovedPvcs(vm)
	)
	for _, pvcName := range pvcNames.List() {
		var pvc, err = h.pvcCache.Get(pvcNamespace, pvcName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// NB(thxCode): ignores error, since this can't be fixed by an immediate requeue,
				// and also doesn't block the whole logic if the PVC has already been deleted.
				continue
			}
			return vm, fmt.Errorf("failed to get PVC(%s/%s): %w", pvcNamespace, pvcName, err)
		}

		if err := unsetBoundedPvcReference(h.pvcClient, pvc, vm); err != nil {
			return vm, fmt.Errorf("failed to revoke VitrualMachine(%s/%s) as PVC(%s/%s)'s owner: %w",
				vm.Namespace, vm.Name, pvcNamespace, pvcName, err)
		}

		if slice.ContainsString(removedPvcs, pvcName) {
			numberOfOwner, err := numberOfBoundedPvcReference(pvc)
			if err != nil {
				return vm, fmt.Errorf("failed to count number of owners for PVC(%s/%s): %w", pvcNamespace, pvcName, err)
			}
			// We are the solely owner here, so try to delete the PVC as requested.
			if numberOfOwner == 1 {
				if err := h.pvcClient.Delete(pvcNamespace, pvcName, &metav1.DeleteOptions{}); err != nil {
					return vm, err
				}
			}
		}
	}

	return vm, nil
}

// getRemovedPvcs returns removed PVCs.
func getRemovedPvcs(vm *kv1.VirtualMachine) []string {
	return strings.Split(vm.Annotations[util.RemovedPvcsAnnotationKey], ",")
}

// getPvcNames returns a name set of the PVCs used by the VMI.
func getPvcNames(vmiSpecPtr *kv1.VirtualMachineInstanceSpec) sets.String {
	var pvcNames = sets.String{}

	for _, volume := range vmiSpecPtr.Volumes {
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName != "" {
			pvcNames.Insert(volume.PersistentVolumeClaim.ClaimName)
		}
	}

	return pvcNames
}
