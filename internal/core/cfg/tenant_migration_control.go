package cfg

import (
	"context"
	"fmt"
	"sync"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"github.com/pieceowater-dev/lotof.lib.gossiper/v2/tenancy"
)

// The per-tenant migration bookkeeping itself lives in gossiper's tenancy
// package -- it was copied into every service and drifted into four
// variants (audit F1). What stays here is the service-specific part: the
// schema sync, the model set the version is computed from, and the warmup
// that rediscovers this app's namespaces.

var (
	controlOnce sync.Once
	control     *tenancy.Control

	warmupOnce sync.Once
	warmup     *tenancy.Warmup
)

func (cfg *Config) tenantControl(database gossiper.Database) *tenancy.Control {
	controlOnce.Do(func() {
		control = tenancy.NewControl(database, cfg.ServiceName, cfg.EffectiveMigrationTargetVersion)
	})
	return control
}

// TriggerTenantWarmupAsync re-discovers this app's namespaces in the
// background: one scan at a time, and not again within the cooldown.
func (cfg *Config) TriggerTenantWarmupAsync() {
	warmupOnce.Do(func() {
		warmup = tenancy.NewWarmup(cfg.ConfigTenants)
	})
	warmup.TriggerAsync()
}

// IsTenantReady reports whether this tenant's schema is migrated to the
// version the current model set implies.
func (cfg *Config) IsTenantReady(ctx context.Context, database gossiper.Database, namespace string) (bool, error) {
	return cfg.tenantControl(database).IsReady(ctx, namespace)
}

// GetTenantMigrationInfo returns the tenant's last recorded status and
// applied version -- empty strings when this service has no record of the
// tenant at all, which is a different answer from "migration in progress".
func (cfg *Config) GetTenantMigrationInfo(ctx context.Context, database gossiper.Database, namespace string) (status string, appliedVersion string, err error) {
	return cfg.tenantControl(database).Info(ctx, namespace)
}

// MigrateTenantWithControl syncs the tenant's schema and runs migrateFn
// under an advisory lock, recording the outcome so other pods (and the next
// request) can see it.
func (cfg *Config) MigrateTenantWithControl(ctx context.Context, database gossiper.Database, tenant *gossiper.EncryptedTenant, migrateFn func(context.Context, string) error) error {
	if tenant == nil || tenant.Namespace == "" {
		return fmt.Errorf("tenant namespace is required")
	}

	return cfg.tenantControl(database).Migrate(ctx, tenant.Namespace,
		func(context.Context) error {
			if !cfg.SyncTenant(database, tenant) {
				return fmt.Errorf("sync tenant failed for namespace %q", tenant.Namespace)
			}
			return nil
		},
		func(ctx context.Context) error {
			if migrateFn == nil {
				return nil
			}
			return migrateFn(ctx, tenant.Namespace)
		},
	)
}
