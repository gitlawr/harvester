package rbac

import (
	"context"

	"github.com/rancher/rancher/pkg/auth/providers/common"
	rconfig "github.com/rancher/rancher/pkg/types/config"

	"github.com/harvester/harvester/pkg/config"
)

const (
	controllerName                        = "harvester-rolebinding-controller"
	managedLabelKey                       = "harvesterhci.io/managed"
	userPrincipalIDAnnotationKey          = "harvesterhci.io/userPrincipalId"
	userPrincipalDisplayNameAnnotationKey = "harvesterhci.io/userPrincipalDisplayName"
)

func Register(ctx context.Context, management *config.Management, options config.Options) error {
	if options.RancherEmbedded {
		rscaled, err := rconfig.NewScaledContext(*management.RestConfig, nil)
		if err != nil {
			return err
		}
		userManager, err := common.NewUserManagerNoBindings(rscaled)
		if err != nil {
			return err
		}
		roleBindings := management.RbacFactory.Rbac().V1().RoleBinding()
		controller := &roleBindingHandler{
			userManager:  userManager,
			roleBindings: roleBindings,
		}
		roleBindings.OnChange(ctx, controllerName, controller.onChanged)
	}
	return nil
}
