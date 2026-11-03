package cfg

import (
	"github.com/pieceowater-dev/lotof.lib.gossiper/v2/tenancy"
)

// ComputeMigrationTargetVersion derives a deterministic version string from
// the set of AutoMigrate models (field names, types, and gorm tags). Any real
// schema change on any registered model changes the hash, so the migration
// gate in MigrateTenantWithControl/IsTenantReady detects drift on its own --
// nobody has to remember to bump a hand-maintained version string.
func ComputeMigrationTargetVersion(models []any) string {
	return tenancy.ComputeTargetVersion(models)
}

// EffectiveMigrationTargetVersion returns the manual override if one was set
// via TENANT_MIGRATION_TARGET_VERSION (kept only as an emergency escape
// hatch), otherwise the version computed from the current AutoMigrate model
// set.
func (cfg *Config) EffectiveMigrationTargetVersion() string {
	if cfg.TenantMigrationTargetVersion != "" {
		return cfg.TenantMigrationTargetVersion
	}
	return ComputeMigrationTargetVersion(cfg.PostgresModels)
}
