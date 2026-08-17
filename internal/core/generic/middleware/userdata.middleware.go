package middleware

import (
	pb "app/internal/core/grpc/generated/generic/utils"
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const UserMetadataKey contextKey = "userMetadata"

func GetUserMetadata(ctx context.Context) (*pb.UserMetadata, bool) {
	userMetadata, ok := ctx.Value(UserMetadataKey).(*pb.UserMetadata)
	return userMetadata, ok
}

// GetNamespaceFromContext extracts the namespace slug injected by the
// metadata middleware. Empty when no metadata is present.
func GetNamespaceFromContext(ctx context.Context) string {
	userMetadata, ok := GetUserMetadata(ctx)
	if !ok || userMetadata == nil {
		return ""
	}
	return userMetadata.Namespace
}

// WithUserMetadata injects metadata for in-process (non-gRPC) code paths.
// This is useful when calling services/controllers directly without going
// through the gRPC layer.
func WithUserMetadata(ctx context.Context, userID, namespace, deviceMeta string) context.Context {
	return context.WithValue(ctx, UserMetadataKey, &pb.UserMetadata{
		UserId:     userID,
		Namespace:  namespace,
		DeviceMeta: deviceMeta,
	})
}

type GrpcMetadata struct {
	excludedMethods map[string]struct{}
}

func NewMetadataMiddleware(excludedMethods []string) *GrpcMetadata {
	methodSet := make(map[string]struct{}, len(excludedMethods))
	for _, method := range excludedMethods {
		methodSet[method] = struct{}{}
	}

	return &GrpcMetadata{excludedMethods: methodSet}
}

// MiddlewareMethod extracts the namespace (and optional user id) from
// incoming gRPC metadata and injects it into the context. The namespace is
// mandatory -- it is required to switch the tenant schema.
func (md *GrpcMetadata) MiddlewareMethod() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, excluded := md.excludedMethods[info.FullMethod]; excluded {
			return handler(ctx, req)
		}

		userMetadata := metadataFromContext(ctx)
		// CRITICAL: namespace must be present for multi-tenant schema isolation.
		if userMetadata.Namespace == "" {
			return nil, status.Error(codes.PermissionDenied, "namespace is required")
		}

		ctx = context.WithValue(ctx, UserMetadataKey, userMetadata)
		return handler(ctx, req)
	}
}

func metadataFromContext(ctx context.Context) *pb.UserMetadata {
	result := &pb.UserMetadata{}

	incoming, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return result
	}

	// Be tolerant to both key variants to ease cross-service integration
	if v := incoming.Get("userid"); len(v) > 0 {
		result.UserId = v[0]
	} else if v := incoming.Get("user-id"); len(v) > 0 {
		result.UserId = v[0]
	}

	if v := incoming.Get("namespace"); len(v) > 0 {
		result.Namespace = v[0]
	} else if v := incoming.Get("namespace-slug"); len(v) > 0 {
		result.Namespace = v[0]
	}

	if v := incoming.Get("device_meta"); len(v) > 0 {
		result.DeviceMeta = v[0]
	}

	return result
}
