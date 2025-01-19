package main

import (
	"app/internal/core/cfg"
	"app/internal/pkg"
	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load application configuration
	appCfg := cfg.Inst()

	// Create a new gRPC server
	grpcServ := grpc.NewServer()

	// Create a new application router
	appRouter := pkg.NewRouter(grpcServ)

	// Register reflection service on gRPC server
	reflection.Register(grpcServ)

	// Create a new server manager
	serverManager := gossiper.NewServerManager()

	// Add the gRPC server to the server manager
	serverManager.AddServer(gossiper.NewGRPCServ(appCfg.GrpcPort, grpcServ, appRouter.InitializeGRPCRoutes))

	// Start all servers managed by the server manager
	serverManager.StartAll()

	// Ensure all servers are stopped when the main function exits
	defer serverManager.StopAll()
}
