package setting

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	harvesterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/util"
)

func (h *Handler) HTTPProxyOnChanged(key string, setting *harvesterv1.Setting) (*harvesterv1.Setting, error) {
	if setting == nil || setting.DeletionTimestamp != nil || setting.Name != "http-proxy" {
		return nil, nil
	}

	if setting.Value == "" && setting.Annotations[util.AnnotationHash] == "" {
		return nil, nil
	}

	hash := sha256.New224()
	hash.Write([]byte(setting.Value))
	currentHash := fmt.Sprintf("%x", hash.Sum(nil))
	if currentHash == setting.Annotations[util.AnnotationHash] {
		return nil, nil
	}

	if err := h.syncHTTPProxy(setting); err != nil {
		return nil, err
	}

	toUpdate := setting.DeepCopy()
	if toUpdate.Annotations == nil {
		toUpdate.Annotations = make(map[string]string)
	}
	toUpdate.Annotations[util.AnnotationHash] = currentHash
	return h.settings.Update(toUpdate)
}

func (h *Handler) syncHTTPProxy(setting *harvesterv1.Setting) error {
	// Add envs to the backup secret used by Longhorn backups
	if err := h.UpdateBackupSecret(setting); err != nil {
		return err
	}

	//redeploy system services. The proxy envs will be injected by the mutation webhook.
	if err := h.redeployDeployment(util.CattleSystemNamespaceName, "rancher"); err != nil {
		return err
	}
	return h.redeployDeployment(h.namespace, "harvester")
}

func (h *Handler) UpdateBackupSecret(setting *harvesterv1.Setting) error {
	var httpProxyConfig util.HTTPProxyConfig
	if err := json.Unmarshal([]byte(setting.Value), &httpProxyConfig); err != nil {
		return err
	}
	secret, err := h.secretCache.Get(util.LonghornSystemNamespaceName, util.BackupTargetSecretName)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	toUpdate := secret.DeepCopy()
	if toUpdate.Data == nil {
		toUpdate.Data = make(map[string][]byte)
	}
	toUpdate.Data["HTTP_PROXY"] = []byte(httpProxyConfig.HTTPProxy)
	toUpdate.Data["HTTPS_PROXY"] = []byte(httpProxyConfig.HTTPSProxy)
	toUpdate.Data["NO_PROXY"] = []byte(httpProxyConfig.NoProxy)
	_, err = h.secrets.Update(toUpdate)
	return err
}

func (h *Handler) redeployDeployment(namespace, name string) error {
	deployment, err := h.deploymentCache.Get(namespace, name)
	if err != nil {
		return err
	}
	toUpdate := deployment.DeepCopy()
	if deployment.Spec.Template.Annotations == nil {
		toUpdate.Spec.Template.Annotations = make(map[string]string)
	}
	toUpdate.Spec.Template.Annotations[util.AnnotationTimestamp] = time.Now().Format(time.RFC3339)

	_, err = h.deployments.Update(toUpdate)
	return err
}
