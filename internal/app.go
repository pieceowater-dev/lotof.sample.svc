package internal

import (
	"context"
	"log"
	"log/slog"
	"strings"
	"time"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"app/internal/core/cfg"
	"app/internal/core/observability"
	"app/internal/pkg"
)

type Application interface {
	Start()
	Stop()
}

type App struct {
	cfg      *cfg.Config
	ctx      context.Context
	servers  *gossiper.ServerManager
	db       gossiper.Database
	logger   *slog.Logger
	tracer   trace.Tracer
	shutdown func(context.Context) error
}

func NewApp() *App {
	baseCtx := context.Background()

	obsLogger, tracer, shutdown, err := observability.Init(baseCtx, observability.Config{
		ServiceName:  "lotof.hub.msvc.users",
		Environment:  cfg.Inst().Environment,
		OtlpEndpoint: cfg.Inst().OtlpEndpoint,
		SampleRatio:  cfg.Inst().TraceSampleRatio,
		LogLevel:     parseLevel(cfg.Inst().LogLevel),
	})
	if err != nil {
		log.Printf("observability init failed: %v", err)
		obsLogger = slog.Default()
		tracer = trace.NewNoopTracerProvider().Tracer("noop")
		shutdown = func(context.Context) error { return nil }
	}

	// Initialize database
	database, err := gossiper.NewDB(
		gossiper.PostgresDB,
		cfg.Inst().PostgresDatabaseDSN,
		false,
		[]any{
			// entities here
		},
	)
	if err != nil {
		obsLogger.Error("failed to create database instance", slog.String("error", err.Error()))
	}

	return &App{
		// Initialize context
		// This context can be used to manage the lifecycle of the application
		// and pass it to various components as needed
		ctx: baseCtx,
		// Load configuration
		// This configuration can be used to set up the application
		cfg: cfg.Inst(),
		// Initialize server manager
		// This server manager can be used to manage multiple servers
		// and their lifecycle
		// It can also be used to add new servers dynamically
		// and manage their lifecycle
		servers:  gossiper.NewServerManager(),
		db:       database, // Assign database
		logger:   obsLogger,
		tracer:   tracer,
		shutdown: shutdown,
	}
}

func (a *App) Start() {
	grpcServ := grpc.NewServer(
		grpc.UnaryInterceptor(observability.GRPCServerInterceptor(a.logger, a.tracer)),
	)
	appRouter := pkg.NewRouter(grpcServ, a.db) // Pass database to router
	reflection.Register(grpcServ)

	a.servers.AddServer(gossiper.NewGRPCServ(a.cfg.GrpcPort, grpcServ, appRouter.InitializeGRPCRoutes))

	a.servers.StartAll()
	defer a.servers.StopAll()
}

func (a *App) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if a.shutdown != nil {
		_ = a.shutdown(ctx)
	}
	a.servers.StopAll()
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
