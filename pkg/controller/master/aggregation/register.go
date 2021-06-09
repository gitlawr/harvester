package aggregation

import (
	"context"
	"crypto/tls"
	"github.com/harvester/harvester/pkg/config"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
)

func Register(ctx context.Context, management *config.Management, options config.Options) error {
	//router := mux.NewRouter()
	//router.PathPrefix("/v1/harvester").Handler(&aggregationHandler{})
	//
	//aggregation.Watch(ctx,
	//	management.CoreFactory.Core().V1().Secret(),
	//	options.Namespace,
	//	"harvester-aggregation",
	//	router)

	return nil
}

type aggregationHandler struct {
}

func (h *aggregationHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	director := func(r *http.Request) {
		r.URL.Scheme = "https"
		r.URL.Host = "localhost:8443"
		//r.URL.Path = strings.Replace(req.URL.Path, "/v1/harvester/", "/", 1)
	}
	httpProxy := &httputil.ReverseProxy{
		Director: director,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	httpProxy.ServeHTTP(rw, req)
}

type handler struct{}

func (h handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	logrus.Infoln("get helloworld")
	for k, v := range req.Header {
		logrus.Infoln("Header %s is %v", k, v)
	}
	rw.Write([]byte("helloworld"))
}
