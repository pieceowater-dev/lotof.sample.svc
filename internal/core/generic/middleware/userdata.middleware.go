package middleware

import (
	"context"

	"github.com/pieceowater-dev/lotof.lib.gossiper/v2/tenancy"
)

// The caller/tenant metadata and its interceptor live in gossiper's tenancy
// package (they were copied into every tenant-scoped service); these keep the
// names the rest of the service already uses.

// UserMetadata is who is calling and for which tenant.
type UserMetadata = tenancy.UserMetadata

// GetUserMetadata returns the metadata the interceptor put on the context.
func GetUserMetadata(ctx context.Context) (*tenancy.UserMetadata, bool) {
	return tenancy.UserMetadataFromContext(ctx)
}

// GetNamespaceFromContext extracts the namespace slug injected by the
// metadata middleware. Empty when no metadata is present.
func GetNamespaceFromContext(ctx context.Context) string {
	return tenancy.NamespaceFromContext(ctx)
}

// WithUserMetadata injects metadata for in-process (non-gRPC) code paths.
func WithUserMetadata(ctx context.Context, userID, namespace, deviceMeta string) context.Context {
	return tenancy.WithUserMetadata(ctx, userID, namespace, deviceMeta)
}

// NewMetadataMiddleware builds the interceptor that reads the namespace (and
// optional user id) from incoming gRPC metadata; excludedMethods pass through.
func NewMetadataMiddleware(excludedMethods []string) *tenancy.MetadataInterceptor {
	return tenancy.NewMetadataInterceptor(excludedMethods)
}
