package main

import (
	"app/internal/core/cfg"
	"app/internal/pkg"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	appCfg := cfg.Inst()

	// Start gRPC server
	listener, err := net.Listen("tcp", ":"+appCfg.AppPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()

	// Initialize and register routes
	router := pkg.NewRouter()
	router.Init(grpcServer)

	log.Printf("gRPC server is running on port %s...\n", appCfg.AppPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
