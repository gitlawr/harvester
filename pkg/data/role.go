package data

import (
	k8scnicncfio "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/sirupsen/logrus"
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
	builder.addClusterRole("harvester-edit", "admin", "edit").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("*").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch")

	builder.addClusterRole("harvester-view", "view").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("get", "list", "watch").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("get", "list", "watch")

	builtInClusterRoles := builder.buildClusterRoles()
	if err := apply.ApplyObjects(builtInClusterRoles...); err != nil {
		return err
	}

	builder = newRoleBuilder()

	builder.addRole("Manage Harvester Settings", "harvester-setting-manage").
		addRule().apiGroups(harvesterhci.GroupName).resources("settings").verbs("get", "list", "watch", "update")
	builder.addRole("Manage Networks", "harvester-network-manage").
		addRule().apiGroups(k8scnicncfio.GroupName).resources("network-attachment-definitions").verbs("*").
		addRule().apiGroups("network.harvesterhci.io").resources("clusternetworks").verbs("*").
		addRule().apiGroups("network.harvesterhci.io").resources("nodenetworks").verbs("*")

	builtInGlobalRoles := builder.buildGlobalRoles()
	if err := apply.ApplyObjects(builtInGlobalRoles...); err != nil {
		return err
	}

	builder = newRoleBuilder()

	builder.addRoleTemplate("Manage Volumes", "virtual-machine-volume-manage", "project", false, false, false).
		addRule().apiGroups(core.GroupName).resources("persistentvolumes").verbs("get", "list", "watch").
		addRule().apiGroups(core.GroupName).resources("persistentvolumeclaims").verbs("*").
		addRule().apiGroups(storagev1.GroupName).resources("storageclasses").verbs("get", "list", "watch").
		addRule().apiGroups(cdi.GroupName).resources("datavolumes").verbs("*")
	builder.addRoleTemplate("View Volumes", "virtual-machine-volume-view", "project", false, false, false).
		addRule().apiGroups(core.GroupName).resources("persistentvolumes").verbs("get", "list", "watch").
		addRule().apiGroups(core.GroupName).resources("persistentvolumeclaims").verbs("get", "list", "watch").
		addRule().apiGroups(storagev1.GroupName).resources("storageclasses").verbs("get", "list", "watch").
		addRule().apiGroups(cdi.GroupName).resources("datavolumes").verbs("get", "list", "watch")
	builder.addRoleTemplate("Manage Virtual Machine Images", "virtual-machine-image-manage", "project", false, false, false).
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("*")
	builder.addRoleTemplate("View Virtual Machine Images", "virtual-machine-image-view", "project", false, false, false).
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("get", "list", "watch")
	builder.addRoleTemplate("Manage SSH Keys", "keypair-manage", "project", false, false, false).
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("*")
	builder.addRoleTemplate("View SSH Keys", "keypair-view", "project", false, false, false).
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("get", "list", "watch")
	builder.addRoleTemplate("Manage Virtual Machine Templates", "virtual-machine-template-manage", "project", false, false, false).
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("*").
		addRule().apiGroups(core.GroupName).resources("configmaps").verbs("*")
	builder.addRoleTemplate("View Virtual Machine Templates", "virtual-machine-template-view", "project", false, false, false).
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("get", "list", "watch").
		addRule().apiGroups(core.GroupName).resources("configmaps").verbs("get", "list", "watch")
	builder.addRoleTemplate("Manage Virtual Machines", "virtual-machine-manage", "project", false, false, false).
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachines").verbs("*").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstances").verbs("*").
		addRule().apiGroups(virtv1.GroupName).resources("virtualmachineinstancemigrations").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackups").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinebackupcontents").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinerestores").verbs("*").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachineimages").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplates").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("virtualmachinetemplateversions").verbs("get", "list", "watch").
		addRule().apiGroups(harvesterhci.GroupName).resources("keypairs").verbs("get", "list", "watch").
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

	builtInRoleTemplates := builder.buildRoleTemplates()
	if err := apply.ApplyObjects(builtInRoleTemplates...); err != nil {
		return err
	}

	logrus.Infoln("applied built-in roles.")
	return nil
}
