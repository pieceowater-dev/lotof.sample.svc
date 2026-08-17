package pkg

import (
	"google.golang.org/grpc"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"

	"app/internal/core/generic/interfaces"
	tenantspb "app/internal/core/grpc/generated/generic/tenants"
	pb "app/internal/core/grpc/generated/lotof.sample.svc/domainItem"
	tenant "app/internal/pkg/_tenant"
	"app/internal/pkg/domainItem"
)

type Router struct {
	modules map[string]interfaces.IModule // Map of module names to their instances.
	db      gossiper.Database             // Database instance for the router.
	server  *grpc.Server
}

// NewRouter creates a new Router instance and initializes every module.
func NewRouter(server *grpc.Server, db gossiper.Database) *Router {
	domainItemModule := domainItem.New(db)
	tenantModule := tenant.New()

	return &Router{
		server: server,
		db:     db,
		modules: map[string]interfaces.IModule{
			domainItemModule.Name(): domainItemModule,
			tenantModule.Name():     tenantModule,
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
	pb.RegisterSampleDomainItemServiceServer(grpcServer, r.modules["DomainItem"].(*domainItem.Module).API)
	tenantspb.RegisterAppTenantsServiceServer(grpcServer, r.modules["tenant"].(*tenant.Module).API)
}

// GetModules returns the map of modules.
func (r *Router) GetModules() map[string]interfaces.IModule {
	return r.modules
}
