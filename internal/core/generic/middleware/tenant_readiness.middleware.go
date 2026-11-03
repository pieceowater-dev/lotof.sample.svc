package middleware

import (
	"context"
	"time"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"app/internal/core/cfg"
)

const (
	tenantReadyInitialRetryDelay = 200 * time.Millisecond
	tenantReadyMaxRetryDelay     = time.Second
	// tenantReadyMaxWait bounds how long ONE request waits for a tenant that
	// is not ready. The loop used to run until the caller's deadline (60s by
	// default), so a request naming a namespace this service will never have
	// -- public endpoints take the namespace straight from a request header,
	// so anyone can name one -- held a handler for a minute and re-triggered
	// a full tenant warmup on every iteration.
	tenantReadyMaxWait = 3 * time.Second
)

// tenantReadinessConfig is the part of *cfg.Config this middleware uses,
// named so the wait logic can be tested without a database.
type tenantReadinessConfig interface {
	IsTenantReady(ctx context.Context, database gossiper.Database, namespace string) (bool, error)
	GetTenantMigrationInfo(ctx context.Context, database gossiper.Database, namespace string) (status string, appliedVersion string, err error)
	TriggerTenantWarmupAsync()
	EnsureAppTenant(ctx context.Context, namespaceSlug string, userID string) error
}

// TenantReadiness gates requests until the tenant schema for the request's
// namespace has been migrated. If it is not ready within tenantReadyMaxWait,
// the caller is asked to retry instead of being held open.
type TenantReadiness struct {
	cfg             tenantReadinessConfig
	db              gossiper.Database
	excludedMethods map[string]struct{}
	maxWait         time.Duration
}

func NewTenantReadinessMiddleware(cfg *cfg.Config, db gossiper.Database, excludedMethods []string) *TenantReadiness {
	return newTenantReadinessMiddleware(cfg, db, excludedMethods, tenantReadyMaxWait)
}

func newTenantReadinessMiddleware(config tenantReadinessConfig, db gossiper.Database, excludedMethods []string, maxWait time.Duration) *TenantReadiness {
	excluded := make(map[string]struct{}, len(excludedMethods))
	for _, method := range excludedMethods {
		excluded[method] = struct{}{}
	}

	return &TenantReadiness{cfg: config, db: db, excludedMethods: excluded, maxWait: maxWait}
}

func (m *TenantReadiness) MiddlewareMethod() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, excluded := m.excludedMethods[info.FullMethod]; excluded {
			return handler(ctx, req)
		}

		namespace := GetNamespaceFromContext(ctx)
		if namespace == "" {
			return handler(ctx, req)
		}

		giveUpAt := time.Now().Add(m.maxWait)
		delay := tenantReadyInitialRetryDelay
		warmupTriggered := false

		for {
			ready, err := m.cfg.IsTenantReady(ctx, m.db, namespace)
			if err == nil && ready {
				return handler(ctx, req)
			}

			// Once per request, not once per iteration: the warmup re-syncs
			// every namespace of this app, and a caller naming an unknown one
			// used to keep it running back to back.
			if !warmupTriggered {
				warmupTriggered = true
				m.cfg.TriggerTenantWarmupAsync()

				// The warmup only rediscovers namespaces already associated
				// with this app (namespace_apps). On a namespace's very first
				// connection that association may not exist yet -- it is
				// written once the billing subscription succeeds -- so warmup
				// alone would leave the namespace stuck. Ask hub directly,
				// scoped to the calling user's own membership.
				if userMetadata, ok := GetUserMetadata(ctx); ok && userMetadata.GetUserId() != "" {
					if ensureErr := m.cfg.EnsureAppTenant(ctx, namespace, userMetadata.GetUserId()); ensureErr == nil {
						if ready, readyErr := m.cfg.IsTenantReady(ctx, m.db, namespace); readyErr == nil && ready {
							return handler(ctx, req)
						}
					}
				}
			}

			// No migration record at all means this service has never
			// provisioned the namespace, and waiting cannot change that --
			// only the warmup triggered above or a call from a member of the
			// namespace can. Say so immediately with a code callers do not
			// retry, so a made-up namespace costs one short call instead of
			// the wait budget times the read-retry count.
			if migrationStatus, _, infoErr := m.cfg.GetTenantMigrationInfo(ctx, m.db, namespace); infoErr == nil && migrationStatus == "" {
				return nil, status.Errorf(codes.FailedPrecondition,
					"namespace %q is not provisioned for this service", namespace)
			}

			remaining := time.Until(giveUpAt)
			if remaining <= 0 {
				return nil, tenantNotReadyError(ctx, err)
			}
			if delay > remaining {
				delay = remaining
			}

			select {
			case <-ctx.Done():
				return nil, tenantNotReadyError(ctx, err)
			case <-time.After(delay):
			}

			if delay < tenantReadyMaxRetryDelay {
				delay *= 2
				if delay > tenantReadyMaxRetryDelay {
					delay = tenantReadyMaxRetryDelay
				}
			}
		}
	}
}

func tenantNotReadyError(ctx context.Context, lastErr error) error {
	_ = grpc.SetHeader(ctx, metadata.Pairs("retry-after", "1"))
	if lastErr != nil {
		return status.Errorf(codes.Unavailable, "tenant readiness check failed: %v", lastErr)
	}
	return status.Error(codes.Unavailable, "tenant migration in progress, retry later")
}
