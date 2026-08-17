package cfg

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
)

var (
	warmupMu      sync.Mutex
	warmupRunning bool

	tenantMigrationStateTableMu    sync.Mutex
	tenantMigrationStateTableReady bool
)

const (
	tenantMigrationRetryBaseDelay = 5 * time.Second
	tenantMigrationRetryMaxDelay  = 5 * time.Minute
)

func (cfg *Config) TriggerTenantWarmupAsync() {
	warmupMu.Lock()
	if warmupRunning {
		warmupMu.Unlock()
		return
	}
	warmupRunning = true
	warmupMu.Unlock()

	go func() {
		defer func() {
			warmupMu.Lock()
			warmupRunning = false
			warmupMu.Unlock()
		}()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[TriggerTenantWarmupAsync] recovered from panic: %v", r)
			}
		}()
		cfg.ConfigTenants()
	}()
}

// ensureTenantMigrationStateTable used to run its CREATE TABLE/ALTER TABLE
// bootstrap on every single call to IsTenantReady -- meaning every unary gRPC
// request, since IsTenantReady is called once per RPC by
// tenantReadinessMiddleware. DDL statements take real Postgres catalog
// locks/parsing even as a no-op "IF NOT EXISTS", and this table's schema
// genuinely only needs establishing once per process lifetime.
// tenantMigrationStateTableReady is only set on success (not cached via
// sync.Once) so a transient DB outage during startup still self-heals on the
// next call instead of wedging the service for its whole lifetime.
func (cfg *Config) ensureTenantMigrationStateTable(ctx context.Context, database gossiper.Database) error {
	tenantMigrationStateTableMu.Lock()
	ready := tenantMigrationStateTableReady
	tenantMigrationStateTableMu.Unlock()
	if ready {
		return nil
	}

	if err := cfg.doEnsureTenantMigrationStateTable(ctx, database); err != nil {
		return err
	}

	tenantMigrationStateTableMu.Lock()
	tenantMigrationStateTableReady = true
	tenantMigrationStateTableMu.Unlock()
	return nil
}

func (cfg *Config) doEnsureTenantMigrationStateTable(ctx context.Context, database gossiper.Database) error {
	query := `
CREATE TABLE IF NOT EXISTS public.tenant_migration_state (
	service_name TEXT NOT NULL,
	tenant_namespace TEXT NOT NULL,
	target_version TEXT NOT NULL,
	applied_version TEXT,
	status TEXT NOT NULL,
	failure_count INT NOT NULL DEFAULT 0,
	next_retry_at TIMESTAMPTZ,
	started_at TIMESTAMPTZ,
	finished_at TIMESTAMPTZ,
	error TEXT,
	PRIMARY KEY (service_name, tenant_namespace)
)`

	if err := database.GetDB().WithContext(ctx).Exec(query).Error; err != nil {
		return err
	}

	alterFailureCount := `ALTER TABLE public.tenant_migration_state ADD COLUMN IF NOT EXISTS failure_count INT NOT NULL DEFAULT 0`
	if err := database.GetDB().WithContext(ctx).Exec(alterFailureCount).Error; err != nil {
		return err
	}

	alterNextRetry := `ALTER TABLE public.tenant_migration_state ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ`
	if err := database.GetDB().WithContext(ctx).Exec(alterNextRetry).Error; err != nil {
		return err
	}

	return nil
}

func (cfg *Config) tryTenantLock(ctx context.Context, database gossiper.Database, namespace string) (bool, error) {
	var locked bool
	err := database.GetDB().WithContext(ctx).
		Raw("SELECT pg_try_advisory_lock(hashtext(?), hashtext(?))", cfg.ServiceName, namespace).
		Row().Scan(&locked)
	if err != nil {
		return false, err
	}
	return locked, nil
}

func (cfg *Config) unlockTenant(ctx context.Context, database gossiper.Database, namespace string) {
	if err := database.GetDB().WithContext(ctx).
		Exec("SELECT pg_advisory_unlock(hashtext(?), hashtext(?))", cfg.ServiceName, namespace).Error; err != nil {
		log.Printf("[TenantMigration] failed to release advisory lock for namespace '%s': %v", namespace, err)
	}
}

func (cfg *Config) upsertTenantMigrationState(ctx context.Context, database gossiper.Database, namespace, targetVersion, appliedVersion, status string, failureCount int, nextRetryAt, startedAt, finishedAt *time.Time, errText *string) error {
	query := `
INSERT INTO public.tenant_migration_state
(service_name, tenant_namespace, target_version, applied_version, status, failure_count, next_retry_at, started_at, finished_at, error)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (service_name, tenant_namespace)
DO UPDATE SET
	target_version = EXCLUDED.target_version,
	applied_version = EXCLUDED.applied_version,
	status = EXCLUDED.status,
	failure_count = EXCLUDED.failure_count,
	next_retry_at = EXCLUDED.next_retry_at,
	started_at = EXCLUDED.started_at,
	finished_at = EXCLUDED.finished_at,
	error = EXCLUDED.error`

	return database.GetDB().WithContext(ctx).Exec(
		query,
		cfg.ServiceName,
		namespace,
		targetVersion,
		appliedVersion,
		status,
		failureCount,
		nextRetryAt,
		startedAt,
		finishedAt,
		errText,
	).Error
}

func (cfg *Config) getTenantFailureState(ctx context.Context, database gossiper.Database, namespace string) (int, *time.Time, error) {
	var failureCount int
	var nextRetryAt sql.NullTime

	err := database.GetDB().WithContext(ctx).
		Raw(`SELECT failure_count, next_retry_at FROM public.tenant_migration_state WHERE service_name = ? AND tenant_namespace = ?`, cfg.ServiceName, namespace).
		Row().Scan(&failureCount, &nextRetryAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil, nil
		}
		return 0, nil, err
	}

	if nextRetryAt.Valid {
		t := nextRetryAt.Time
		return failureCount, &t, nil
	}

	return failureCount, nil, nil
}

func tenantRetryDelay(failureCount int) time.Duration {
	if failureCount <= 0 {
		return tenantMigrationRetryBaseDelay
	}

	delay := tenantMigrationRetryBaseDelay
	for i := 1; i < failureCount; i++ {
		delay *= 2
		if delay >= tenantMigrationRetryMaxDelay {
			return tenantMigrationRetryMaxDelay
		}
	}

	if delay > tenantMigrationRetryMaxDelay {
		return tenantMigrationRetryMaxDelay
	}

	return delay
}

func (cfg *Config) MigrateTenantWithControl(ctx context.Context, database gossiper.Database, tenant *gossiper.EncryptedTenant, migrateFn func(context.Context, string) error) error {
	if tenant == nil || tenant.Namespace == "" {
		return fmt.Errorf("tenant namespace is required")
	}

	if err := cfg.ensureTenantMigrationStateTable(ctx, database); err != nil {
		return err
	}

	failureCount, nextRetryAt, err := cfg.getTenantFailureState(ctx, database, tenant.Namespace)
	if err != nil {
		return err
	}
	if nextRetryAt != nil && time.Now().UTC().Before(*nextRetryAt) {
		return fmt.Errorf("tenant migration backoff active for namespace '%s' until %s", tenant.Namespace, nextRetryAt.UTC().Format(time.RFC3339))
	}

	locked, err := cfg.tryTenantLock(ctx, database, tenant.Namespace)
	if err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("tenant migration is already running for namespace '%s'", tenant.Namespace)
	}
	defer cfg.unlockTenant(ctx, database, tenant.Namespace)

	now := time.Now().UTC()
	if err := cfg.upsertTenantMigrationState(ctx, database, tenant.Namespace, cfg.EffectiveMigrationTargetVersion(), "", "running", failureCount, nil, &now, nil, nil); err != nil {
		return err
	}

	if !cfg.SyncTenant(database, tenant) {
		errMsg := "sync tenant failed"
		finished := time.Now().UTC()
		nextRetry := finished.Add(tenantRetryDelay(failureCount + 1))
		_ = cfg.upsertTenantMigrationState(ctx, database, tenant.Namespace, cfg.EffectiveMigrationTargetVersion(), "", "failed", failureCount+1, &nextRetry, &now, &finished, &errMsg)
		return fmt.Errorf("sync tenant failed for namespace '%s'", tenant.Namespace)
	}

	if migrateFn != nil {
		if err := migrateFn(ctx, tenant.Namespace); err != nil {
			errText := err.Error()
			finished := time.Now().UTC()
			nextRetry := finished.Add(tenantRetryDelay(failureCount + 1))
			_ = cfg.upsertTenantMigrationState(ctx, database, tenant.Namespace, cfg.EffectiveMigrationTargetVersion(), "", "failed", failureCount+1, &nextRetry, &now, &finished, &errText)
			return err
		}
	}

	finished := time.Now().UTC()
	if err := cfg.upsertTenantMigrationState(ctx, database, tenant.Namespace, cfg.EffectiveMigrationTargetVersion(), cfg.EffectiveMigrationTargetVersion(), "done", 0, nil, &now, &finished, nil); err != nil {
		return err
	}

	return nil
}

func (cfg *Config) IsTenantReady(ctx context.Context, database gossiper.Database, namespace string) (bool, error) {
	if namespace == "" {
		return true, nil
	}

	if err := cfg.ensureTenantMigrationStateTable(ctx, database); err != nil {
		return false, err
	}

	var statusVal string
	var appliedVersion sql.NullString
	err := database.GetDB().WithContext(ctx).
		Raw(`SELECT status, applied_version FROM public.tenant_migration_state WHERE service_name = ? AND tenant_namespace = ?`, cfg.ServiceName, namespace).
		Row().Scan(&statusVal, &appliedVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return statusVal == "done" && appliedVersion.Valid && appliedVersion.String == cfg.EffectiveMigrationTargetVersion(), nil
}

// GetTenantMigrationInfo returns this tenant's last-recorded migration
// status and applied version -- for diagnostics (e.g. an admin console's
// namespace health check) where "never migrated" vs "migrated but now
// stale" vs "migration failed" needs to stay distinguishable, unlike
// IsTenantReady's single collapsed boolean. Empty strings mean no migration
// record exists yet for this tenant on this service.
func (cfg *Config) GetTenantMigrationInfo(ctx context.Context, database gossiper.Database, namespace string) (status string, appliedVersion string, err error) {
	if namespace == "" {
		return "", "", nil
	}

	if err := cfg.ensureTenantMigrationStateTable(ctx, database); err != nil {
		return "", "", err
	}

	var statusVal string
	var applied sql.NullString
	err = database.GetDB().WithContext(ctx).
		Raw(`SELECT status, applied_version FROM public.tenant_migration_state WHERE service_name = ? AND tenant_namespace = ?`, cfg.ServiceName, namespace).
		Row().Scan(&statusVal, &applied)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil
		}
		return "", "", err
	}

	return statusVal, applied.String, nil
}
