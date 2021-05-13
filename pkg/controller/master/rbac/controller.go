package rbac

import (
	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	ctlv3 "github.com/rancher/rancher/pkg/generated/controllers/management.cattle.io/v3"
	"github.com/rancher/wrangler/pkg/slice"
)

var (
	globalRoles   = []string{"admin", "user", "roles-manage", "authn-manage", "users-manage"}
	roleTemplates = []string{"project-member", "project-owner", "read-only", "projectroletemplatebindings-view", "projectroletemplatebindings-manage", "create-ns"}
)

// Handler adds the "harvesterhci.io/managed" label to built-in Rancher globalRoles and roleTemplates that Harvester uses.
type Handler struct {
	globalRoles   ctlv3.GlobalRoleClient
	roleTemplates ctlv3.RoleTemplateClient
}

func (h Handler) onGlobalRoleChanged(_ string, globalRole *v3.GlobalRole) (*v3.GlobalRole, error) {
	if globalRole == nil || globalRole.DeletionTimestamp != nil {
		return nil, nil
	}
	if slice.ContainsString(globalRoles, globalRole.Name) && globalRole.Labels[managedLabelKey] != "true" {
		toUpdate := globalRole.DeepCopy()
		if toUpdate.Labels == nil {
			toUpdate.Labels = make(map[string]string)
		}
		toUpdate.Labels[managedLabelKey] = "true"
		return h.globalRoles.Update(toUpdate)
	}

	return nil, nil
}

func (h Handler) onRoleTemplateChanged(_ string, roleTemplate *v3.RoleTemplate) (*v3.RoleTemplate, error) {
	if roleTemplate == nil || roleTemplate.DeletionTimestamp != nil {
		return nil, nil
	}
	if slice.ContainsString(roleTemplates, roleTemplate.Name) && roleTemplate.Labels[managedLabelKey] != "true" {
		toUpdate := roleTemplate.DeepCopy()
		if toUpdate.Labels == nil {
			toUpdate.Labels = make(map[string]string)
		}
		toUpdate.Labels[managedLabelKey] = "true"
		return h.roleTemplates.Update(toUpdate)
	}

	return nil, nil
}
