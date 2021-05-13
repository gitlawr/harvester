package rbac

import (
	"github.com/rancher/rancher/pkg/user"
	ctlv1 "github.com/rancher/wrangler/pkg/generated/controllers/rbac/v1"
	v1 "k8s.io/api/rbac/v1"
)

// roleBindingHandler ensures the user and updates the RoleBinding subjects.
// This is for creating RoleBindings with principals from auth providers when
// the Harvester user does not exist.
type roleBindingHandler struct {
	roleBindings ctlv1.RoleBindingClient
	userManager  user.Manager
}

func (h roleBindingHandler) onChanged(_ string, binding *v1.RoleBinding) (*v1.RoleBinding, error) {
	if binding == nil || binding.DeletionTimestamp != nil {
		return nil, nil
	}
	if binding.Labels[managedLabelKey] != "true" ||
		binding.Annotations[userPrincipalIDAnnotationKey] == "" ||
		binding.Annotations[userPrincipalDisplayNameAnnotationKey] == "" {
		return nil, nil
	}
	user, err := h.userManager.EnsureUser(binding.Annotations[userPrincipalIDAnnotationKey], binding.Annotations[userPrincipalDisplayNameAnnotationKey])
	if err != nil {
		return nil, err
	}
	toUpdate := binding.DeepCopy()
	delete(toUpdate.Annotations, userPrincipalIDAnnotationKey)
	delete(toUpdate.Annotations, userPrincipalDisplayNameAnnotationKey)
	toUpdate.Subjects = append(toUpdate.Subjects, v1.Subject{
		APIGroup: v1.GroupName,
		Kind:     v1.UserKind,
		Name:     user.Name,
	})
	return h.roleBindings.Update(toUpdate)
}
