package data

import (
	"fmt"

	k8scnicncfio "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io"
	"github.com/rancher/rancher/pkg/apis/management.cattle.io"
	"github.com/rancher/wrangler/pkg/apply"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/config"
)

var (
	aggregationClusterRoles = []string{"admin", "edit", "view"}
)

func addClusterRoles(mgmtCtx *config.Management) error {
	if err := patchAggregationClusterRoleLabels(mgmtCtx); err != nil {
		return err
	}

	apply, err := apply.NewForConfig(mgmtCtx.RestConfig)
	if err != nil {
		return err
	}
	apply = apply.WithDynamicLookup().WithSetID("harvester-cluster-roles")
	builder := clusterRoleBuilder{}

	// roles in cluster scope(work with ClusterRoleBindings)
	builder.addRole("Administrator", "admin", roleContextCluster).
		addRule().apiGroups("*").resources("*").verbs("*").
		addRule().apiGroups().nonResourceURLs("*").verbs("*")
	builder.addRole("Standard User", "user", roleContextCluster).
		addRule().apiGroups(harvesterhci.GroupName).resources("preferences").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("settings").verbs("get", "list", "watch")
	builder.addRole("Configure Authentication", "authn-manage", roleContextCluster).
		addRule().apiGroups(management.GroupName).resources("authconfigs").verbs("get", "list", "watch", "update")
	builder.addRole("Manage Namespaces", "namespace-manage", roleContextCluster).
		addRule().apiGroups("").resources("namespaces").verbs("*")
	builder.addRole("Manage Roles", "role-manage", roleContextCluster).
		addRule().apiGroups(v1.GroupName).resources("clusterroles").verbs("*")
	builder.addRole("Manage Users", "user-manage", roleContextCluster).
		addRule().apiGroups(management.GroupName).resources("users").verbs("*").
		addRule().apiGroups(v1.GroupName).resources("clusterroles").verbs("get", "list", "watch").
		addRule().apiGroups(v1.GroupName).resources("clusterrolebindings").verbs("*")
	builder.addRole("Manage Settings", "setting-manage", roleContextCluster).
		addRule().apiGroups(harvesterhci.GroupName).resources("settings").verbs("get", "list", "watch", "update")

	// roles in namespace scope(work with RoleBindings)
	builder.addRole("Namespace Owner", "namespace-owner", roleContextNamespace).
		setAggregationClusterRoles("admin")
	builder.addRole("Namespace Member", "namespace-member", roleContextNamespace).
		setAggregationClusterRoles("edit")
	builder.addRole("Read-only", "read-only", roleContextNamespace).
		setAggregationClusterRoles("view")
	builder.addRole("Manage Volumes", "volume-manage", roleContextNamespace).
		addRule().apiGroups("").resources("persistentvolumes").verbs("get", "list", "watch").
		addRule().apiGroups("storage.k8s.io").resources("storageclasses").verbs("get", "list", "watch").
		addRule().apiGroups("").resources("persistentvolumeclaims").verbs("*").
		addRule().apiGroups(cdi.GroupName).resources("datavolumes").verbs("*")
	builder.addRole("View Volumes", "volume-view", roleContextNamespace).
		addRule().apiGroups("").resources("persistentvolumes").verbs("get", "list", "watch").
		addRule().apiGroups("storage.k8s.io").resources("storageclasses").verbs("get", "list", "watch").
		addRule().apiGroups("").resources("persistentvolumeclaims").verbs("get", "list", "watch").
		addRule().apiGroups(cdi.GroupName).resources("datavolumes").verbs("get", "list", "watch")
	builder.addRole("Manage Virtual Machines", "virtual-machine-manage", roleContextNamespace).
		addRule().apiGroups("kubevirt.io").resources("virtualmachines").verbs("*").
		addRule().apiGroups("kubevirt.io").resources("virtualmachineinstances").verbs("*").
		addRule().apiGroups("kubevirt.io").resources("virtualmachineinstancemigrations").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("*").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch").
		setAggregateToClusterRoles("admin", "edit")
	builder.addRole("View Virtual Machines", "virtual-machine-view", roleContextNamespace).
		addRule().apiGroups("kubevirt.io").resources("virtualmachines").verbs("get", "list", "watch").
		addRule().apiGroups("kubevirt.io").resources("virtualmachineinstances").verbs("get", "list", "watch").
		addRule().apiGroups("kubevirt.io").resources("virtualmachineinstancemigrations").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("get", "list", "watch").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch").
		setAggregateToClusterRoles("view")
	builder.addRole("Manage Namespace Members", "namespace-member-manage", roleContextNamespace).
		addRule().apiGroups("rbac.authorization.k8s.io").resources("rolebindings").verbs("*")
	builder.addRole("View Namespace Members", "namespace-member-view", roleContextNamespace).
		addRule().apiGroups("rbac.authorization.k8s.io").resources("rolebindings").verbs("get", "list", "watch")

	builtInClusterRoles := builder.build()
	if err := apply.ApplyObjects(builtInClusterRoles...); err != nil {
		return err
	}
	return nil
}

// patchAggregationClusterRoleLabels patches labels to the default cluster roles so that they can be used as
// aggregation roles by harvester built-in roles.
func patchAggregationClusterRoleLabels(mgmtCtx *config.Management) error {
	clusterRoles := mgmtCtx.RbacFactory.Rbac().V1().ClusterRole()
	for _, name := range aggregationClusterRoles {
		data := fmt.Sprintf(`{"metadata":{"labels":{"harvesterhci.io/name":"%s"}}}}`, name)
		if _, err := clusterRoles.Patch(name, types.StrategicMergePatchType, []byte(data)); err != nil {
			return err
		}
	}
	return nil
}
