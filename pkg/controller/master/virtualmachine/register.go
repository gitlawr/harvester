package virtualmachine

import (
	"context"

	"github.com/harvester/harvester/pkg/config"
)

const (
	vmControllerCreatePvcsFromAnnotationControllerName = "VMController.CreatePVCsFromAnnotation"
	vmControllerSetOwnerOfPvcsControllerName           = "VMController.SetOwnerOfPVCs"
	vmControllerUnsetOwnerOfPvcsControllerName         = "VMController.UnsetOwnerOfPVCs"
	vmiControllerUnsetOwnerOfPvcsControllerName        = "VMIController.UnsetOwnerOfPVCs"
	vmControllerSetDefaultManagementNetworkMac         = "VMController.SetDefaultManagementNetworkMacAddress"
)

func Register(ctx context.Context, management *config.Management, options config.Options) error {
	var pvcClient = management.CoreFactory.Core().V1().PersistentVolumeClaim()
	var pvcCache = pvcClient.Cache()

	// registers the vm controller
	var vmCtrl = &VMController{
		pvcClient: pvcClient,
		pvcCache:  pvcCache,
	}
	var virtualMachineClient = management.VirtFactory.Kubevirt().V1().VirtualMachine()
	virtualMachineClient.OnChange(ctx, vmControllerCreatePvcsFromAnnotationControllerName, vmCtrl.createPvcFromAnnotation)
	virtualMachineClient.OnChange(ctx, vmControllerSetOwnerOfPvcsControllerName, vmCtrl.SetOwnerOfPvcs)
	virtualMachineClient.OnRemove(ctx, vmControllerUnsetOwnerOfPvcsControllerName, vmCtrl.UnsetOwnerOfPvcs)

	// registers the vmi controller
	var virtualMachineCache = virtualMachineClient.Cache()
	var vmiCtrl = &VMIController{
		virtualMachineCache: virtualMachineCache,
		pvcClient:           pvcClient,
		pvcCache:            pvcCache,
	}
	var virtualMachineInstanceClient = management.VirtFactory.Kubevirt().V1().VirtualMachineInstance()
	virtualMachineInstanceClient.OnRemove(ctx, vmiControllerUnsetOwnerOfPvcsControllerName, vmiCtrl.UnsetOwnerOfPVCs)

	// register the vm network controller upon the VMI changes
	var (
		vmClient  = management.VirtFactory.Kubevirt().V1().VirtualMachine()
		vmCache   = management.VirtFactory.Kubevirt().V1().VirtualMachine().Cache()
		vmiClient = management.VirtFactory.Kubevirt().V1().VirtualMachineInstance()
	)
	var vmNetworkCtl = &VMNetworkController{
		vmClient:  vmClient,
		vmCache:   vmCache,
		vmiClient: vmiClient,
	}
	virtualMachineInstanceClient.OnChange(ctx, vmControllerSetDefaultManagementNetworkMac, vmNetworkCtl.SetDefaultNetworkMacAddress)

	return nil
}
