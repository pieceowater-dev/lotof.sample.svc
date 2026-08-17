package cfg

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
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
			AppBundleName:       getEnv("APP_BUNDLE_NAME", "pieceowater.sample"),
			ServiceName:         getEnv("SERVICE_NAME", "lotof.sample.svc"),
			GrpcPort:            getEnv("GRPC_PORT", "50051"),
			HubApplicationAddr:  getEnv("HUB_APPLICATION_ADDR", "localhost:50050"),
			AppBundleSecret:     getEnv("APP_BUNDLE_SECRET", "12345678901234567890123456789012"),
			PostgresDatabaseDSN: normalizePostgresDSN(getEnv("POSTGRES_DB_DSN", "postgres://pieceouser:pieceopassword@localhost:5432/sample?sslmode=disable")),
			PostgresModels:      []any{},
			DebugSQL:            getEnv("DEBUG_SQL", "false") == "true",
			// Empty by default: the effective target version is auto-computed
			// from the AutoMigrate model set (see EffectiveMigrationTargetVersion
			// in tenant_migration_version.go). Set only as an emergency override.
			TenantMigrationTargetVersion: getEnv("TENANT_MIGRATION_TARGET_VERSION", ""),
			Environment:                  getEnv("ENVIRONMENT", "local"),
			OtlpEndpoint:                 getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://tempo:4318"),
			TraceSampleRatio:             getEnvFloat("TRACE_SAMPLE_RATIO", 1.0),
			LogLevel:                     getEnv("LOG_LEVEL", "info"),
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

// getEnv retrieves the value of the environment variable named by the key.
// It returns the value, or the specified default value if the variable is not present.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func normalizePostgresDSN(raw string) string {
	if raw == "" {
		return raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	query := parsed.Query()
	if query.Get("statement_cache_capacity") == "" {
		query.Set("statement_cache_capacity", "0")
	}

	if query.Get("default_query_exec_mode") == "" {
		query.Set("default_query_exec_mode", "simple_protocol")
	}

	parsed.RawQuery = query.Encode()
	return parsed.String()
}
