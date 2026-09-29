package internal

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"app/internal/core/cfg"
	"app/internal/core/generic/middleware"
	"app/internal/core/observability"
	"app/internal/pkg"
	domainItemEnt "app/internal/pkg/domainItem/ent"
)

type Application interface {
	Start()
	Stop()
}

type App struct {
	cfg          *cfg.Config
	ctx          context.Context
	servers      *gossiper.ServerManager
	db           gossiper.Database
	logger       *slog.Logger
	tracer       trace.Tracer
	shutdown     func(context.Context) error
	warmupCancel context.CancelFunc
}

func NewApp() *App {
	baseCtx := context.Background()

	obsLogger, tracer, shutdown, err := observability.Init(baseCtx, observability.Config{
		ServiceName:  cfg.Inst().ServiceName,
		Environment:  cfg.Inst().Environment,
		OtlpEndpoint: cfg.Inst().OtlpEndpoint,
		SampleRatio:  cfg.Inst().TraceSampleRatio,
		LogLevel:     parseLevel(cfg.Inst().LogLevel),
	})
	if err != nil {
		log.Printf("observability init failed: %v", err)
		obsLogger = slog.Default()
		tracer = noop.NewTracerProvider().Tracer("noop")
		shutdown = func(context.Context) error { return nil }
	}

	// Every entity that needs its own per-namespace table goes here -- this
	// list drives both AutoMigrate and the migration-version hash (see
	// tenant_migration_version.go), so a schema change on any of these
	// entities is enough to trigger a re-migration for every tenant.
	models := []any{
		&domainItemEnt.DomainItem{},
	}
	cfg.Inst().PostgresModels = models

	database, err := gossiper.NewDB(
		gossiper.PostgresDB,
		cfg.Inst().PostgresDatabaseDSN,
		cfg.Inst().DebugSQL,
		nil,
	)
	if err != nil {
		obsLogger.Error("failed to create database instance", slog.String("error", err.Error()))
	}

	return &App{
		ctx:      baseCtx,
		cfg:      cfg.Inst(),
		servers:  gossiper.NewServerManager(),
		db:       database,
		logger:   obsLogger,
		tracer:   tracer,
		shutdown: shutdown,
	}
}

func (a *App) Start() {
	excludedMethods := []string{
		"/tenants.AppTenantsService/NewTenant",
		// GetTenantStatus reports readiness (including "not ready") -- gating
		// it behind tenantReadinessMiddleware would make it hang/retry
		// instead of ever answering for a genuinely broken tenant.
		"/tenants.AppTenantsService/GetTenantStatus",
		"/grpc.health.v1.Health/Check",
	}

	metadataMiddleware := middleware.NewMetadataMiddleware(excludedMethods)
	tenantReadinessMiddleware := middleware.NewTenantReadinessMiddleware(a.cfg, a.db, excludedMethods)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			gossiper.RecoveryUnaryServerInterceptor(),
			metadataMiddleware.MiddlewareMethod(),
			tenantReadinessMiddleware.MiddlewareMethod(),
			observability.GRPCServerInterceptor(a.logger, a.tracer),
		),
	)
	grpcHealth := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, grpcHealth)
	grpcHealth.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)

	appRouter := pkg.NewRouter(grpcServer, a.db)
	reflection.Register(grpcServer)

	a.servers.AddServer(gossiper.NewGRPCServ(a.cfg.GrpcPort, grpcServer, appRouter.InitializeGRPCRoutes))
	grpcHealth.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	a.startTenantWarmupLoop()
	a.cfg.TriggerTenantWarmupAsync()

	a.servers.StartAll()
	defer a.servers.StopAll()
}

func (a *App) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if a.warmupCancel != nil {
		a.warmupCancel()
	}

	if a.shutdown != nil {
		_ = a.shutdown(ctx)
	}
	a.servers.StopAll()
}

// startTenantWarmupLoop periodically re-discovers namespaces registered for
// this app and re-provisions/re-migrates any that changed -- a namespace
// added to this app after startup, or a schema change deployed while a
// tenant's process was already up, still gets picked up without a restart.
func (a *App) startTenantWarmupLoop() {
	interval := 10 * time.Minute
	if raw := os.Getenv("TENANT_MIGRATION_POLL_INTERVAL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			interval = parsed
		}
	}

	loopCtx, cancel := context.WithCancel(context.Background())
	a.warmupCancel = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				a.cfg.TriggerTenantWarmupAsync()
			case <-loopCtx.Done():
				return
			}
		}
	}()
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
