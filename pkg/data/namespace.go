package data

import (
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/harvester/harvester/pkg/config"
)

const (
	publicNamespace            = "harvester-public"
	userNamespaceAnnotationKey = "harvesterhci.io/user-namespace"
)

func addNamespaces(mgmtCtx *config.Management) error {
	namespaces := mgmtCtx.CoreFactory.Core().V1().Namespace()
	roleBindings := mgmtCtx.RbacFactory.Rbac().V1().RoleBinding()

	// Create harvester-public namespace
	if _, err := namespaces.Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: publicNamespace},
	}); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	// All authenticated users are readable in the public namespace
	if _, err := roleBindings.Create(&rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rolebinding-harvester-public",
			Namespace: publicNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "harvester-read-only",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.GroupKind,
				Name:     "system:authenticated",
			},
		},
	}); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// Mark the default namespace as a user namespace
	defaultNamespace, err := namespaces.Get("default", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	if defaultNamespace.Annotations == nil {
		defaultNamespace.Annotations = make(map[string]string)
	}
	if defaultNamespace.Annotations[userNamespaceAnnotationKey] != "true" {
		defaultNamespace.Annotations[userNamespaceAnnotationKey] = "true"
		_, err = namespaces.Update(defaultNamespace)
		return err
	}

	return nil
}
