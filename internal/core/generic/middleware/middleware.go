package middleware

import "google.golang.org/grpc"

type GrpcMiddleware interface {
	MiddlewareMethod() grpc.UnaryServerInterceptor
}
