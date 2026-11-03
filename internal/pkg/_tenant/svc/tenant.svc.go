package svc

import (
	"app/internal/core/cfg"
	"app/internal/core/grpc/generated/generic/tenants"
	"context"
	"log/slog"

	"github.com/pieceowater-dev/lotof.lib.gossiper/v2/observability"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
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

	mgr := cfg.Inst().Tenants()
	err := mgr.MigrateTenant(ctx, s.database, &tenant, func(ctx context.Context, namespace string) error {
		logger.Info("Switching to schema")
		if err := mgr.MigrateSchema(ctx, s.database, namespace, s.autoMigrateEntities, logger); err != nil {
			return err
		}
		return nil
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
	ready, appliedVersion, targetVersion, err := cfg.Inst().Tenants().Status(ctx, s.database, req.Namespace)
	if err != nil {
		return nil, err
	}
	return &tenants.GetTenantStatusResponse{
		SchemaReady:    ready,
		AppliedVersion: appliedVersion,
		TargetVersion:  targetVersion,
	}, nil
}
