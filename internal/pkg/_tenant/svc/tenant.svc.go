package svc

import (
	"app/internal/core/cfg"
	"app/internal/core/grpc/generated/generic/tenants"
	"app/internal/core/observability"
	"context"
	"fmt"
	"log/slog"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"gorm.io/gorm"
)

type TenantServer struct {
	database gossiper.Database
	tenants.UnimplementedAppTenantsServiceServer
	autoMigrateEntities []any
}

func NewTenantServer(database gossiper.Database, autoMigrateEntities []any) *TenantServer {
	return &TenantServer{
		database:            database,
		autoMigrateEntities: autoMigrateEntities,
	}
}

func (s *TenantServer) NewTenant(ctx context.Context, req *tenants.AddNamespaceTenantRequest) (*tenants.AddNamespaceTenantResponse, error) {
	logger := observability.LoggerFromContext(ctx, slog.Default()).With(
		slog.String("operation", "new_tenant"),
		slog.String("namespace", req.Namespace))

	tenant := gossiper.EncryptedTenant{
		Namespace:   req.Namespace,
		Credentials: req.Credentials,
	}

	err := cfg.Inst().MigrateTenantWithControl(ctx, s.database, &tenant, func(ctx context.Context, namespace string) error {
		logger.Info("Switching to schema")

		logger.Info("Running AutoMigrate", slog.Int("entity_count", len(s.autoMigrateEntities)))
		return s.database.WithSchema(ctx, namespace, func(tx *gorm.DB) error {
			for i, entity := range s.autoMigrateEntities {
				entityType := fmt.Sprintf("%T", entity)
				logger.Debug("Migrating entity",
					slog.Int("progress", i+1),
					slog.Int("total", len(s.autoMigrateEntities)),
					slog.String("entity_type", entityType))

				if err := tx.AutoMigrate(entity); err != nil {
					logger.Error("Failed to auto-migrate entity",
						slog.String("entity_type", entityType),
						slog.Any("error", err))
					return fmt.Errorf("auto-migrate failed for %s: %w", entityType, err)
				}
			}
			return nil
		})
	})
	if err != nil {
		logger.Error("Tenant migration failed", slog.String("error", err.Error()))
		return nil, err
	}

	logger.Info("Successfully initialized namespace")
	return &tenants.AddNamespaceTenantResponse{Success: true}, nil
}

// GetTenantStatus reports this tenant's live migration/schema readiness --
// used by an admin console's on-demand namespace health check. Deliberately
// excluded from tenantReadinessMiddleware (see app.go's excludedMethods):
// its whole purpose is to report readiness including when it's false, so
// gating it on readiness would make it hang/retry instead of ever reporting
// a broken tenant.
func (s *TenantServer) GetTenantStatus(ctx context.Context, req *tenants.GetTenantStatusRequest) (*tenants.GetTenantStatusResponse, error) {
	status, appliedVersion, err := cfg.Inst().GetTenantMigrationInfo(ctx, s.database, req.Namespace)
	if err != nil {
		return nil, err
	}
	targetVersion := cfg.Inst().EffectiveMigrationTargetVersion()

	schemaReady := status == "done" && appliedVersion != "" && appliedVersion == targetVersion
	if schemaReady {
		// IsTenantReady-equivalent bookkeeping alone only proves the tenant
		// *was* migrated successfully at some point -- a cheap live read
		// closes the gap to "is it actually queryable right now."
		if liveErr := s.database.WithSchemaReadOnly(ctx, req.Namespace, func(tx *gorm.DB) error {
			return tx.Exec("SELECT 1").Error
		}); liveErr != nil {
			schemaReady = false
		}
	}

	return &tenants.GetTenantStatusResponse{
		SchemaReady:    schemaReady,
		AppliedVersion: appliedVersion,
		TargetVersion:  targetVersion,
	}, nil
}
