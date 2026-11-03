package cfg

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/pieceowater-dev/lotof.lib.gossiper/v2/appkit"

	"github.com/joho/godotenv"
	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
)

// Config holds the configuration settings for the application.
type Config struct {
	AppBundleName      string // App bundle id registered in Hub's namespace_apps (e.g. "pieceowater.sample")
	ServiceName        string // Used as the observability service name
	GrpcPort           string // Port for gRPC server
	HubApplicationAddr string // Hub gateway gRPC address, used for tenant discovery/provisioning
	AppBundleSecret    string // Secret key (32 bytes) to decrypt tenant db credentials

	PostgresDatabaseDSN          string // Data Source Name for PostgreSQL database
	PostgresModels               []any  // List of models for database migration (populated by app.go)
	Namespaces                   []string
	DebugSQL                     bool
	TenantMigrationTargetVersion string // Emergency override only; leave empty (see EffectiveMigrationTargetVersion)

	Environment      string
	OtlpEndpoint     string
	TraceSampleRatio float64
	LogLevel         string
	// ServiceAuthSecret is the shared service-to-service token (SERVICE_AUTH_SECRET);
	// empty disables the check in development only.
	ServiceAuthSecret string
	// ServiceAuthMode is "enforce" (default) or "report" (log calls without a
	// valid token but let them through) -- see gossiper.ServiceAuthServerInterceptorWithMode.
	ServiceAuthMode string
}

var (
	once     sync.Once
	instance *Config
)

// Inst returns a singleton instance of Config, loading environment variables if necessary.
func Inst() *Config {
	once.Do(func() {
		loadDotEnv()

		instance = &Config{
			AppBundleName:       appkit.Env("APP_BUNDLE_NAME", "pieceowater.sample"),
			ServiceName:         appkit.Env("SERVICE_NAME", "lotof.sample.svc"),
			GrpcPort:            appkit.Env("GRPC_PORT", "50051"),
			HubApplicationAddr:  appkit.Env("HUB_APPLICATION_ADDR", "localhost:50050"),
			AppBundleSecret:     gossiper.SecretEnv("APP_BUNDLE_SECRET"),
			PostgresDatabaseDSN: appkit.NormalizePostgresDSN(gossiper.SecretEnv("POSTGRES_DB_DSN")),
			PostgresModels:      []any{},
			DebugSQL:            appkit.Env("DEBUG_SQL", "false") == "true",
			// Empty by default: the effective target version is auto-computed
			// from the AutoMigrate model set (see EffectiveMigrationTargetVersion
			// in tenant_migration_version.go). Set only as an emergency override.
			TenantMigrationTargetVersion: appkit.Env("TENANT_MIGRATION_TARGET_VERSION", ""),
			Environment:                  appkit.Env("ENVIRONMENT", "local"),
			OtlpEndpoint:                 appkit.Env("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			TraceSampleRatio:             appkit.EnvFloat("TRACE_SAMPLE_RATIO", 1.0),
			LogLevel:                     appkit.Env("LOG_LEVEL", "info"),
			ServiceAuthSecret:            appkit.Env("SERVICE_AUTH_SECRET", ""),
			ServiceAuthMode:              appkit.Env("SERVICE_AUTH_MODE", gossiper.ServiceAuthEnforce),
		}
	})
	return instance
}

func loadDotEnv() {
	if err := godotenv.Load(); err == nil {
		return
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("No .env file found, loading from OS environment variables.")
		return
	}

	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../"))
	dotEnvPath := filepath.Join(projectRoot, ".env")

	if err := godotenv.Load(dotEnvPath); err != nil {
		fmt.Println("No .env file found, loading from OS environment variables.")
	}
}
