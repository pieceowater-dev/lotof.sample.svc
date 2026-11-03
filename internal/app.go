package internal

import (
	"context"
	"log"
	"log/slog"
	"time"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"github.com/pieceowater-dev/lotof.lib.gossiper/v2/observability"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"

	"app/internal/core/cfg"
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
	grpcServer, grpcHealth := newGRPCServer(a.cfg, a.db, a.logger, a.tracer)
	appRouter := pkg.NewRouter(grpcServer, a.db)

	a.servers.AddServer(gossiper.NewGRPCServ(a.cfg.GrpcPort, grpcServer, appRouter.InitializeGRPCRoutes))
	grpcHealth.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	// "" above is plain liveness; the named "readiness" service follows the
	// database, for readiness probes (see gossiper.WatchReadiness).
	go gossiper.WatchReadiness(a.ctx, grpcHealth, 10*time.Second, gossiper.PingDatabase(a.db))
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
