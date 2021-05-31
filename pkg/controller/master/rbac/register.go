package rbac

import (
	"context"

	"github.com/harvester/harvester/pkg/config"
)

const (
	controllerName  = "harvester-rbac-controller"
	managedLabelKey = "harvesterhci.io/managed"
)

func Register(ctx context.Context, management *config.Management, options config.Options) error {
	if options.RancherEmbedded {
		globalRoles := management.RancherManagementFactory.Management().V3().GlobalRole()
		roleTemplates := management.RancherManagementFactory.Management().V3().RoleTemplate()
		controller := &Handler{
			globalRoles:   globalRoles,
			roleTemplates: roleTemplates,
		}

		globalRoles.OnChange(ctx, controllerName, controller.onGlobalRoleChanged)
		roleTemplates.OnChange(ctx, controllerName, controller.onRoleTemplateChanged)
	}
	return nil
}
