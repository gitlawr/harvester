package pvc

import (
	"sort"
	"strings"

	"github.com/rancher/vm/pkg/generated/controllers/kubevirt.io/v1alpha3"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	virtv1alpha3 "kubevirt.io/client-go/api/v1alpha3"
)

const (
	controllerAgentName  = "vm-vmi-controller"
	mountedByVmiLabelKey = "vm.cattle.io/mounted-by-vmi"
)

// Handler records mounted-by-vmi infos in pvc labels
type Handler struct {
	pvcs     v1.PersistentVolumeClaimController
	pvcCache v1.PersistentVolumeClaimCache
	vmiCache v1alpha3.VirtualMachineInstanceCache
}

func (h *Handler) OnVmiChanged(key string, vmi *virtv1alpha3.VirtualMachineInstance) (*virtv1alpha3.VirtualMachineInstance, error) {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		return vmi, nil
	}
	return vmi, h.syncAllPvcLabels(vmi.Namespace)
}

func (h *Handler) OnVmiRemove(key string, vmi *virtv1alpha3.VirtualMachineInstance) (*virtv1alpha3.VirtualMachineInstance, error) {
	return vmi, h.syncAllPvcLabels(vmi.Namespace)
}

func (h *Handler) syncAllPvcLabels(ns string) error {
	pvcs, err := h.pvcCache.List(ns, labels.Everything())
	if err != nil {
		return err
	}
	vmis, err := h.vmiCache.List(ns, labels.Everything())
	if err != nil {
		return err
	}
	for _, pvc := range pvcs {
		if err := h.syncPvcMountedBy(pvc, vmis); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) syncPvcMountedBy(pvc *corev1.PersistentVolumeClaim, vmis []*virtv1alpha3.VirtualMachineInstance) error {
	newMountedBy := getMountedBy(pvc.Name, vmis)
	if getMountedByFromLabel(pvc.Labels) == newMountedBy {
		return nil
	}
	toUpdate := pvc.DeepCopy()
	if toUpdate.Labels == nil {
		toUpdate.Labels = make(map[string]string)
	}
	toUpdate.Labels[mountedByVmiLabelKey] = newMountedBy
	_, err := h.pvcs.Update(toUpdate)
	return err
}

func getMountedByFromLabel(labels map[string]string) string {
	if labels == nil {
		return ""
	}
	return labels[mountedByVmiLabelKey]
}

func getMountedBy(pvcName string, vmis []*virtv1alpha3.VirtualMachineInstance) string {
	var ids []string
	for _, vmi := range vmis {
		if vmi.DeletionTimestamp != nil {
			continue
		}
		for _, volume := range vmi.Spec.Volumes {
			if volume.VolumeSource.PersistentVolumeClaim != nil &&
				volume.VolumeSource.PersistentVolumeClaim.ClaimName == pvcName {
				ids = append(ids, vmi.Name)
			}
		}

	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}
