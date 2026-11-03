package internal

import (
	"log/slog"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"github.com/pieceowater-dev/lotof.lib.gossiper/v2/observability"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"

	"app/internal/core/cfg"
	"app/internal/core/generic/middleware"
)

// newGRPCServer builds the inbound gRPC server: service-to-service auth, the
// tenant metadata/readiness gates and the health service. Provisioning
// (NewTenant), health checks and readiness reports must bypass the tenant
// gates. The returned health server starts NOT_SERVING; the caller flips it
// to SERVING once Start has finished initializing.
func newGRPCServer(c *cfg.Config, db gossiper.Database, logger *slog.Logger, tracer trace.Tracer) (*grpc.Server, *health.Server) {
	excludedMethods := []string{
		"/tenants.AppTenantsService/NewTenant",
		// GetTenantStatus reports readiness (including "not ready") -- gating
		// it behind tenantReadinessMiddleware would make it hang/retry
		// instead of ever answering for a genuinely broken tenant.
		"/tenants.AppTenantsService/GetTenantStatus",
		"/grpc.health.v1.Health/Check",
	}

	metadataMiddleware := middleware.NewMetadataMiddleware(excludedMethods)
	tenantReadinessMiddleware := middleware.NewTenantReadinessMiddleware(c, db, excludedMethods)

	return gossiper.NewGRPCServer(
		// Service-to-service auth: only callers holding SERVICE_AUTH_SECRET (this
		// domain's gateway) get past this point; a no-op until the secret is set
		// (development only), and report-only while SERVICE_AUTH_MODE=report.
		gossiper.ServiceAuthServerInterceptorWithMode(c.ServiceAuthSecret, c.ServiceAuthMode,
			"/grpc.health.v1.Health/", "/grpc.reflection."),
		metadataMiddleware.MiddlewareMethod(),
		tenantReadinessMiddleware.MiddlewareMethod(),
		observability.GRPCServerInterceptor(logger, tracer),
	)
}
