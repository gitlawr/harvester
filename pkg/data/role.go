package data

import (
	k8scnicncfio "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io"
	"github.com/rancher/rancher/pkg/apis/management.cattle.io"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/kubernetes/pkg/apis/core"
	virtv1 "kubevirt.io/client-go/api/v1"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/config"
)

func addRoles(mgmtCtx *config.Management) error {

	apply, err := apply.NewForConfig(mgmtCtx.RestConfig)
	if err != nil {
		return err
	}
	apply = apply.WithDynamicLookup().WithSetID("harvester-roles")
	builder := newRoleBuilder()

	// roles in cluster scope(work with ClusterRoleBindings)
	builder.addRole("Administrator", "admin").
		addRule().apiGroups("*").resources("*").verbs("*").
		addRule().apiGroups().nonResourceURLs("*").verbs("*")
	builder.addRole("Standard User", "user").
		addRule().apiGroups(harvesterhci.GroupName).resources("preferences").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("settings").verbs("get", "list", "watch")
	builder.addRole("Configure Authentication", "authn-manage").
		addRule().apiGroups(management.GroupName).resources("authconfigs").verbs("get", "list", "watch", "update")
	builder.addRole("Manage Roles", "role-manage").
		addRule().apiGroups(v1.GroupName).resources("clusterroles").verbs("*")
	builder.addRole("Manage Users", "user-manage").
		addRule().apiGroups(management.GroupName).resources("users", "globalrolebindings").verbs("*").
		addRule().apiGroups(management.GroupName).resources("globalroles").verbs("get", "list", "watch")
	builder.addRole("Manage Settings", "setting-manage").
		addRule().apiGroups(harvesterhci.GroupName).resources("settings").verbs("get", "list", "watch", "update")

	builtInGlobalRoles := builder.buildGlobalRoles()
	if err := apply.ApplyObjects(builtInGlobalRoles...); err != nil {
		return err
	}

	builder = newRoleBuilder()

	// roles in namespace scope(work with RoleBindings)
	builder.addRoleTemplate("Project Owner", "project-owner", "project", false, false, false).
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachines").verbs("*").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstances").verbs("*").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstancemigrations").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("*").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch").
		setRoleTemplateNames("project-owner")
	builder.addRoleTemplate("Project Member", "project-member", "project", false, false, false).
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachines").verbs("*").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstances").verbs("*").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstancemigrations").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("*").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch").
		setRoleTemplateNames("project-member")
	builder.addRoleTemplate("Read-only", "read-only", "project", false, false, false).
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachines").verbs("get", "list", "watch").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstances").verbs("get", "list", "watch").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstancemigrations").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("get", "list", "watch").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch").
		setRoleTemplateNames("read-only")

	builder.addRoleTemplate("Manage Volumes", "volume-manage", "project", false, false, false).
		addRule().apiGroups(core.GroupName).resources("persistentvolumes").verbs("get", "list", "watch").
		addRule().apiGroups(core.GroupName).resources("persistentvolumeclaims").verbs("*").
		addRule().apiGroups(storagev1.GroupName).resources("storageclasses").verbs("get", "list", "watch").
		addRule().apiGroups(cdi.GroupName).resources("datavolumes").verbs("*")
	builder.addRoleTemplate("View Volumes", "volume-view", "project", false, false, false).
		addRule().apiGroups(core.GroupName).resources("persistentvolumes").verbs("get", "list", "watch").
		addRule().apiGroups(core.GroupName).resources("persistentvolumeclaims").verbs("get", "list", "watch").
		addRule().apiGroups(storagev1.GroupName).resources("storageclasses").verbs("get", "list", "watch").
		addRule().apiGroups(cdi.GroupName).resources("datavolumes").verbs("get", "list", "watch")
	builder.addRoleTemplate("Manage Virtual Machines", "virtual-machine-manage", "project", false, false, false).
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachines").verbs("*").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstances").verbs("*").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstancemigrations").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("*").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch")
	builder.addRoleTemplate("View Virtual Machines", "virtual-machine-view", "project", false, false, false).
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachines").verbs("get", "list", "watch").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstances").verbs("get", "list", "watch").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstancemigrations").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("get", "list", "watch").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch")
	builder.addRoleTemplate("Manage Project Members", "projectroletemplatebindings-manage", "project", false, false, false).
		addRule().apiGroups(management.GroupName).resources("projectroletemplatebindings").verbs("*")
	builder.addRoleTemplate("View Project Members", "projectroletemplatebindings-view", "project", false, false, false).
		addRule().apiGroups(management.GroupName).resources("projectroletemplatebindings").verbs("get", "list", "watch")

	builtInRoleTemplates := builder.buildRoleTemplates()
	if err := apply.ApplyObjects(builtInRoleTemplates...); err != nil {
		return err
	}

	logrus.Infoln("applied built-in roles.")
	return nil
}
