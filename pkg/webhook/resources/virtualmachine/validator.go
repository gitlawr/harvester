package virtualmachine

import (
	"encoding/json"
	"errors"
	"fmt"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/client-go/api/v1"

	"github.com/harvester/harvester/pkg/util"
	werror "github.com/harvester/harvester/pkg/webhook/error"
	"github.com/harvester/harvester/pkg/webhook/types"
)

func NewValidator() types.Validator {
	return &vmValidator{}
}

type vmValidator struct {
	types.DefaultValidator
}

func (v *vmValidator) Resource() types.Resource {
	return types.Resource{
		Name:       "virtualmachines",
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   kubevirtv1.SchemeGroupVersion.Group,
		APIVersion: kubevirtv1.SchemeGroupVersion.Version,
		ObjectType: &kubevirtv1.VirtualMachine{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}

func (v *vmValidator) Create(request *types.Request, newObj runtime.Object) error {
	vm := newObj.(*kubevirtv1.VirtualMachine)

	if err := v.checkVolumeClaimTemplatesAnnotation(vm); err != nil {
		message := fmt.Sprintf("the volumeClaimTemplates annotaion is invalid: %v", err)
		return werror.NewInvalidError(message, "metadata.annotations")
	}
	return nil
}

// Update checks if a UPDATE operation is allowed. If no error is returned, the operation is allowed.
func (v *vmValidator) Update(request *types.Request, oldObj runtime.Object, newObj runtime.Object) error {
	vm := newObj.(*kubevirtv1.VirtualMachine)

	if err := v.checkVolumeClaimTemplatesAnnotation(vm); err != nil {
		message := fmt.Sprintf("the volumeClaimTemplates annotaion is invalid: %v", err)
		return werror.NewInvalidError(message, "metadata.annotations")
	}
	return nil
}

func (v *vmValidator) checkVolumeClaimTemplatesAnnotation(vm *kubevirtv1.VirtualMachine) error {
	if vm == nil {
		return nil
	}
	volumeClaimTemplates, ok := vm.Annotations[util.AnnotationVolumeClaimTemplates]
	if !ok || volumeClaimTemplates == "" {
		return nil
	}
	var pvcs []*corev1.PersistentVolumeClaim
	if err := json.Unmarshal([]byte(volumeClaimTemplates), &pvcs); err != nil {
		return err
	}
	for _, pvc := range pvcs {
		if pvc.Name == "" {
			return errors.New("PVC name is required")
		}
	}
	return nil
}
