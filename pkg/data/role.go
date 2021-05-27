package data

import (
	k8scnicncfio "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io"
	"github.com/rancher/rancher/pkg/apis/management.cattle.io"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/config"
)

var (
	aggregationClusterRoles = []string{"admin", "edit", "view"}
)

func addRoles(mgmtCtx *config.Management) error {

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
	globalRoles := mgmtCtx.RancherManagementFactory.Management().V3().GlobalRole()
	for _, role := range builtInGlobalRoles {
		if _, err := globalRoles.Create(&role); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	builder = newRoleBuilder()

	// roles in namespace scope(work with RoleBindings)
	builder.addRoleTemplate("Project Owner", "project-owner", "project", false, false, false).
		setRoleTemplateNames("project-owner", "virtual-machine-manage")
	builder.addRoleTemplate("Project Member", "project-member", "project", false, false, false).
		setRoleTemplateNames("project-member", "virtual-machine-manage")
	builder.addRoleTemplate("Read-only", "read-only", "project", false, false, false).
		setRoleTemplateNames("read-only")

	builder.addRoleTemplate("Manage Volumes", "volume-manage", "project", false, false, false).
		addRule().apiGroups("").resources("persistentvolumes").verbs("get", "list", "watch").
		addRule().apiGroups("storage.k8s.io").resources("storageclasses").verbs("get", "list", "watch").
		addRule().apiGroups("").resources("persistentvolumeclaims").verbs("*").
		addRule().apiGroups(cdi.GroupName).resources("datavolumes").verbs("*")
	builder.addRoleTemplate("View Volumes", "volume-view", "project", false, false, false).
		addRule().apiGroups("").resources("persistentvolumes").verbs("get", "list", "watch").
		addRule().apiGroups("storage.k8s.io").resources("storageclasses").verbs("get", "list", "watch").
		addRule().apiGroups("").resources("persistentvolumeclaims").verbs("get", "list", "watch").
		addRule().apiGroups(cdi.GroupName).resources("datavolumes").verbs("get", "list", "watch")
	builder.addRoleTemplate("Manage Virtual Machines", "virtual-machine-manage", "project", false, false, false).
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
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch")
	builder.addRoleTemplate("View Virtual Machines", "virtual-machine-view", "project", false, false, false).
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
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch")
	builder.addRoleTemplate("Manage Project Members", "projectroletemplatebindings-manage", "project", false, false, false).
		setRoleTemplateNames("projectroletemplatebindings-manage")
	builder.addRoleTemplate("View Project Members", "projectroletemplatebindings-view", "project", false, false, false).
		setRoleTemplateNames("projectroletemplatebindings-view")

	builtInRoleTemplates := builder.buildRoleTemplates()
	roleTemplates := mgmtCtx.RancherManagementFactory.Management().V3().RoleTemplate()
	for _, roleTemplate := range builtInRoleTemplates {
		if _, err := roleTemplates.Create(&roleTemplate); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	logrus.Infoln("applied built-in roles.")
	return nil
}
