package setting

import (
	"context"

	"github.com/harvester/harvester/pkg/config"
)

const (
	controllerName = "harvester-setting-controller"
)

func Register(ctx context.Context, management *config.Management, options config.Options) error {
	settings := management.HarvesterFactory.Harvesterhci().V1beta1().Setting()
	secrets := management.CoreFactory.Core().V1().Secret()
	deployments := management.AppsFactory.Apps().V1().Deployment()
	controller := &Handler{
		namespace:       options.Namespace,
		settings:        settings,
		secrets:         secrets,
		secretCache:     secrets.Cache(),
		deployments:     deployments,
		deploymentCache: deployments.Cache(),
	}

	settings.OnChange(ctx, controllerName, controller.LogLevelOnChanged)
	settings.OnChange(ctx, controllerName, controller.HTTPProxyOnChanged)
	return nil
}
