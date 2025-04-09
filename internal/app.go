package internal

import (
	"app/internal/core/cfg"
	"app/internal/pkg"
	"context"
	"log"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Application interface {
	Start()
	Stop()
}

type App struct {
	cfg     *cfg.Config
	ctx     context.Context
	servers *gossiper.ServerManager
	db      gossiper.Database // Add database field
}

func NewApp() *App {
	// Initialize database
	database, err := gossiper.NewDB(
		gossiper.PostgresDB,
		cfg.Inst().PostgresDatabaseDSN,
		false,
		[]any{},
	)
	if err != nil {
		log.Fatalf("Failed to create database instance: %v", err)
	}

	return &App{
		// Initialize context
		// This context can be used to manage the lifecycle of the application
		// and pass it to various components as needed
		ctx: context.Background(),
		// Load configuration
		// This configuration can be used to set up the application
		cfg: cfg.Inst(),
		// Initialize server manager
		// This server manager can be used to manage multiple servers
		// and their lifecycle
		// It can also be used to add new servers dynamically
		// and manage their lifecycle
		servers: gossiper.NewServerManager(),
		db:      database, // Assign database
	}
}

func (a *App) Start() {
	grpcServ := grpc.NewServer()
	appRouter := pkg.NewRouter(grpcServ, a.db) // Pass database to router
	reflection.Register(grpcServ)

	a.servers.AddServer(gossiper.NewGRPCServ(a.cfg.GrpcPort, grpcServ, appRouter.InitializeGRPCRoutes))

	a.servers.StartAll()
	defer a.servers.StopAll()
}

func (a *App) Stop() {
	a.servers.StopAll()
}
