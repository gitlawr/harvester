package setting

import (
	v1 "github.com/rancher/wrangler/pkg/generated/controllers/apps/v1"
	corev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	harvesterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
)

type Handler struct {
	namespace       string
	settings        v1beta1.SettingClient
	secrets         corev1.SecretClient
	secretCache     corev1.SecretCache
	deployments     v1.DeploymentClient
	deploymentCache v1.DeploymentCache
}

// LogLevelOnChanged updates the log level on setting changes
func (h *Handler) LogLevelOnChanged(key string, setting *harvesterv1.Setting) (*harvesterv1.Setting, error) {
	if setting == nil || setting.DeletionTimestamp != nil || setting.Name != "log-level" || setting.Value == "" {
		return setting, nil
	}

	level, err := logrus.ParseLevel(setting.Value)
	if err != nil {
		return setting, err
	}

	logrus.Infof("set log level to %s", level)
	logrus.SetLevel(level)
	return setting, nil
}
