package downloadurl

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/config"
)

const (
	controllerName = "downloadurl-controller"
)

func Register(ctx context.Context, management *config.Management, options config.Options) error {
	logrus.Debugln("registered " + controllerName)
	downloadURLController := management.HarvesterFactory.Harvesterhci().V1beta1().DownloadURL()
	handler := &handler{
		httpClient: http.Client{
			Timeout: 15 * time.Second,
		},
	}
	downloadURLController.OnChange(ctx, controllerName, handler.OnChange)
	return nil
}

// handler tries to download from specified URL on DownloadURL changes.
type handler struct {
	httpClient http.Client
}

func (h *handler) OnChange(_ string, downloadURL *v1beta1.DownloadURL) (*v1beta1.DownloadURL, error) {
	if downloadURL == nil || downloadURL.DeletionTimestamp != nil {
		return nil, nil
	}

	logrus.Debugf("download from URL: %s", downloadURL.URL)
	resp, err := h.httpClient.Get(downloadURL.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// then operates the response body

	return downloadURL, nil
}
