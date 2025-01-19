package pkg

import (
	"app/internal/core/generic/interfaces"
	pb "app/internal/core/grpc/generated/lotof.sample.proto/lotof.sample.svc/domainItem"
	"app/internal/pkg/domainItem"
	"google.golang.org/grpc"
)

type Router struct {
	modules map[string]interfaces.IModule // Map of module names to their instances.

	server *grpc.Server
}

// NewRouter creates a new Router instance and initializes the DomainItem module.
func NewRouter(server *grpc.Server) *Router {
	domainItemModule := domainItem.New()

	return &Router{
		server: server,
		modules: map[string]interfaces.IModule{
			domainItemModule.Name(): domainItemModule,
		},
	}
}

// InitializeRouter initializes the router and its gRPC routes.
func (r *Router) InitializeRouter() (any, error) {
	r.InitializeGRPCRoutes(r.server)
	return nil, nil
}

// InitializeGRPCRoutes registers the gRPC routes for the modules.
func (r *Router) InitializeGRPCRoutes(grpcServer *grpc.Server) {
	pb.RegisterDomainItemServiceServer(grpcServer, r.modules["DomainItem"].(*domainItem.Module).API)
}

// GetModules returns the map of modules.
func (r *Router) GetModules() map[string]interfaces.IModule {
	return r.modules
}
