package pod

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/util"
	"github.com/harvester/harvester/pkg/webhook/types"
)

var matchingLabels = map[string]string{
	"longhorn.io/component":       "backing-image-data-source",
	"app.kubernetes.io/component": "apiserver",
	"app":                         "rancher",
}

func NewMutator(settingCache v1beta1.SettingCache) types.Mutator {
	return &podMutator{
		setttingCache: settingCache,
	}
}

// podMutator injects http proxy envs to system pods that may access external services.
// Including harvester, rancher, longhorn backing-image-data-source.
type podMutator struct {
	types.DefaultMutator
	setttingCache v1beta1.SettingCache
}

func newResource(ops []admissionregv1.OperationType) types.Resource {
	return types.Resource{
		Name:           string(corev1.ResourcePods),
		Scope:          admissionregv1.NamespacedScope,
		APIGroup:       corev1.SchemeGroupVersion.Group,
		APIVersion:     corev1.SchemeGroupVersion.Version,
		ObjectType:     &corev1.Pod{},
		OperationTypes: ops,
	}
}

func (m *podMutator) Resource() types.Resource {
	return newResource([]admissionregv1.OperationType{
		admissionregv1.Create,
	})
}

func (m *podMutator) Create(request *types.Request, newObj runtime.Object) (types.PatchOps, error) {
	pod := newObj.(*corev1.Pod)

	podLabels := labels.Set(pod.Labels)
	var match bool
	for k, v := range matchingLabels {
		if (labels.Set{k: v}).AsSelector().Matches(podLabels) {
			match = true
			break
		}
	}
	if !match {
		return nil, nil
	}
	proxySetting, err := m.setttingCache.Get("http-proxy")
	if err != nil || proxySetting.Value == "" {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var httpProxyConfig util.HTTPProxyConfig
	if err := json.Unmarshal([]byte(proxySetting.Value), &httpProxyConfig); err != nil {
		return nil, err
	}
	if httpProxyConfig.HTTPProxy == "" && httpProxyConfig.HTTPSProxy == "" && httpProxyConfig.NoProxy == "" {
		return nil, nil
	}

	var proxyEnvs = []corev1.EnvVar{
		{
			Name:  "HTTP_PROXY",
			Value: httpProxyConfig.HTTPProxy,
		},
		{
			Name:  "HTTPS_PROXY",
			Value: httpProxyConfig.HTTPSProxy,
		},
		{
			Name:  "NO_PROXY",
			Value: httpProxyConfig.NoProxy,
		},
	}
	var patchOps types.PatchOps
	for idx, container := range pod.Spec.Containers {
		patchOps = append(patchOps, envPatches(container.Env, proxyEnvs, fmt.Sprintf("/spec/containers/%d/env", idx))...)
	}
	return patchOps, nil
}

func envPatches(target, envVars []corev1.EnvVar, basePath string) types.PatchOps {
	var patchOps types.PatchOps
	first := len(target) == 0
	var value interface{}
	for _, envVar := range envVars {
		value = envVar
		path := basePath
		if first {
			first = false
			value = []corev1.EnvVar{envVar}
		} else {
			path = path + "/-"
		}
		valueStr, err := json.Marshal(value)
		if err != nil {
			logrus.Error(err)
		}
		patchOps = append(patchOps, fmt.Sprintf(`{"op": "add", "path": "%s", "value": %s}`, path, valueStr))
	}
	return patchOps
}
