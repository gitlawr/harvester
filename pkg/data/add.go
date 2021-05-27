package data

import (
	"context"

	"github.com/harvester/harvester/pkg/config"
)

// Init adds built-in resources
func Init(ctx context.Context, mgmtCtx *config.Management, namespace string) error {
	if err := addCRDs(ctx, mgmtCtx.RestConfig); err != nil {
		return err
	}
	if err := addRoles(mgmtCtx); err != nil {
		return err
	}
	if err := addDefaultAdmin(mgmtCtx, namespace); err != nil {
		return err
	}
	if err := addNamespaces(mgmtCtx); err != nil {
		return err
	}
	if err := addTemplates(mgmtCtx, publicNamespace); err != nil {
		return err
	}

	return nil
}
