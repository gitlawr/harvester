package pvc

import (
	"context"
	"github.com/rancher/vm/pkg/config"
)

func Register(ctx context.Context, management *config.Management) error {
	pvcs := management.CoreFactory.Core().V1().PersistentVolumeClaim()
	vmis := management.VirtFactory.Kubevirt().V1alpha3().VirtualMachineInstance()
	controller := &Handler{
		pvcs:     pvcs,
		pvcCache: pvcs.Cache(),
		vmiCache: vmis.Cache(),
	}

	vmis.OnChange(ctx, controllerAgentName, controller.OnVmiChanged)
	vmis.OnRemove(ctx, controllerAgentName, controller.OnVmiRemove)
	return nil
}
