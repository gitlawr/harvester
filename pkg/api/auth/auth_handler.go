package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	dashboardauthapi "github.com/kubernetes/dashboard/src/app/backend/auth/api"
	"github.com/rancher/rancher/pkg/auth/requests"
	rancherconfig "github.com/rancher/rancher/pkg/types/config"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/harvester/harvester/pkg/auth"
	"github.com/harvester/harvester/pkg/config"
	"github.com/harvester/harvester/pkg/util"
)

const (
	jwtServiceAccountClaimSubject = "sub" // https://github.com/kubernetes/kubernetes/blob/3783e03dc9df61604c470aa21f198a888e3ec692/pkg/serviceaccount/claims.go#L64
)

func NewMiddleware(ctx context.Context, scaled *config.Scaled, rancherRestConfig *rest.Config, AddRancherAuthenticator bool, pathPrefix string, ignorePrefix ...string) (*Middleware, error) {
	middleware := &Middleware{
		tokenManager: scaled.TokenManager,
		pathPrefix:   pathPrefix,
		ignorePrefix: ignorePrefix,
	}

	if !AddRancherAuthenticator {
		return middleware, nil
	}

	emptyClusterID := func(*http.Request) string {
		return ""
	}
	sc, err := rancherconfig.NewScaledContext(*rancherRestConfig, nil)
	if err != nil {
		return nil, err
	}
	// initialize to add tokenKeyIndexer
	requests.NewAuthenticator(ctx, emptyClusterID, sc)

	if err := sc.Start(ctx); err != nil {
		return nil, err
	}
	middleware.rancherAuthenticator = requests.NewAuthenticator(ctx, emptyClusterID, sc)

	return middleware, nil
}

type Middleware struct {
	tokenManager         dashboardauthapi.TokenManager
	rancherAuthenticator requests.Authenticator

	pathPrefix   string
	ignorePrefix []string
}

func (h *Middleware) ToAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if auth.IsRancherAuthMode() {
				h.rancherAuth(rw, r, next)
			} else {
				h.auth(rw, r, next)
			}
		})
	}
}

func (h *Middleware) rancherAuth(rw http.ResponseWriter, r *http.Request, next http.Handler) {
	if !h.matches(r.URL.Path) {
		next.ServeHTTP(rw, r)
		return
	}
	ok, u, groups, err := h.rancherAuthenticator.Authenticate(r)
	if err != nil {
		util.ResponseError(rw, http.StatusUnauthorized, err)
		return
	}
	info := &user.DefaultInfo{
		Name:   u,
		UID:    u,
		Groups: groups,
	}
	if !ok {
		info = &user.DefaultInfo{
			Name: "system:unauthenticated",
			UID:  "system:unauthenticated",
			Groups: []string{
				"system:unauthenticated",
			},
		}
	}

	ctx := request.WithUser(r.Context(), info)
	r = r.WithContext(ctx)
	next.ServeHTTP(rw, r)
}

func (h *Middleware) auth(rw http.ResponseWriter, r *http.Request, next http.Handler) {
	jweToken, err := extractJWETokenFromRequest(r)
	if err != nil {
		util.ResponseError(rw, http.StatusUnauthorized, err)
		return
	}

	userInfo, err := h.getUserInfoFromToken(jweToken)
	if err != nil {
		util.ResponseError(rw, http.StatusUnauthorized, err)
		return
	}

	ctx := request.WithUser(r.Context(), userInfo)
	r = r.WithContext(ctx)
	next.ServeHTTP(rw, r)
}

func (h *Middleware) getUserInfoFromToken(jweToken string) (userInfo user.Info, err error) {
	//handle panic from calling kubernetes dashboard tokenManager.Decrypt
	defer func() {
		if recoveryMessage := recover(); recoveryMessage != nil {
			err = fmt.Errorf("%v", recoveryMessage)
		}
	}()

	var authInfo *api.AuthInfo
	authInfo, err = h.tokenManager.Decrypt(jweToken)
	if err != nil {
		return
	}

	return impersonateAuthInfoToUserInfo(authInfo), nil
}

func (h Middleware) matches(path string) bool {
	if strings.HasPrefix(path, h.pathPrefix) {
		for _, prefix := range h.ignorePrefix {
			if strings.HasPrefix(path, prefix) {
				return false
			}
		}
		return true
	}
	return false
}
