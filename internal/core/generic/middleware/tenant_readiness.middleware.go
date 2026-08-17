package middleware

import (
	"app/internal/core/cfg"
	"context"
	"time"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	tenantReadyInitialRetryDelay = 200 * time.Millisecond
	tenantReadyMaxRetryDelay     = 5 * time.Second
)

// TenantReadiness gates requests until the tenant schema for the request's
// namespace has been migrated. If not ready, it triggers an async warmup and
// asks the caller to retry.
type TenantReadiness struct {
	cfg             *cfg.Config
	db              gossiper.Database
	excludedMethods map[string]struct{}
}

func NewTenantReadinessMiddleware(cfg *cfg.Config, db gossiper.Database, excludedMethods []string) *TenantReadiness {
	excluded := make(map[string]struct{}, len(excludedMethods))
	for _, method := range excludedMethods {
		excluded[method] = struct{}{}
	}

	return &TenantReadiness{cfg: cfg, db: db, excludedMethods: excluded}
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

		delay := tenantReadyInitialRetryDelay
		for {
			ready, err := m.cfg.IsTenantReady(ctx, m.db, namespace)
			if err == nil && ready {
				break
			}

			m.cfg.TriggerTenantWarmupAsync()

			// The periodic warmup above only rediscovers namespaces already
			// associated with this app (namespace_apps) -- for a namespace's
			// very first connection, that association may not exist yet
			// (it's only written once billing subscription succeeds), so
			// warmup alone would leave this namespace stuck forever. Ask hub
			// directly, scoped to the calling user's own membership.
			if userMetadata, ok := GetUserMetadata(ctx); ok && userMetadata.GetUserId() != "" {
				if ensureErr := m.cfg.EnsureAppTenant(ctx, namespace, userMetadata.GetUserId()); ensureErr == nil {
					if ready, err := m.cfg.IsTenantReady(ctx, m.db, namespace); err == nil && ready {
						break
					}
				}
			}

			select {
			case <-ctx.Done():
				_ = grpc.SetHeader(ctx, metadata.Pairs("retry-after", "1"))
				if err != nil {
					return nil, status.Errorf(codes.Unavailable, "tenant readiness check failed: %v", err)
				}
				return nil, status.Error(codes.Unavailable, "tenant migration in progress, retry later")
			case <-time.After(delay):
			}

			if delay < tenantReadyMaxRetryDelay {
				delay *= 2
				if delay > tenantReadyMaxRetryDelay {
					delay = tenantReadyMaxRetryDelay
				}
			}
		}

		return handler(ctx, req)
	}
}
