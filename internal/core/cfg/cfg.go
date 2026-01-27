package cfg

import (
	someEnt "app/internal/pkg/domainItem/ent"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

// Config holds the configuration settings for the application.
type Config struct {
	GrpcPort string // Port for gRPC server
	RestPort string // Port for REST server

	PostgresDatabaseDSN string // Data Source Name for PostgreSQL database
	PostgresModels      []any  // List of models for database migration

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
		err := godotenv.Load()
		if err != nil {
			fmt.Println("No .env file found, loading from OS environment variables.")
		}

		instance = &Config{
			GrpcPort:            getEnv("GRPC_PORT", "50051"),
			RestPort:            getEnv("REST_PORT", "3000"),
			PostgresDatabaseDSN: getEnv("POSTGRES_DB_DSN", "postgres://pieceouser:pieceopassword@localhost:5432/sample?sslmode=disable"),
			PostgresModels: []any{
				// models to migration here:
				// &ent.MyModel{},
				&someEnt.Something{},
			},
			Environment:      getEnv("ENVIRONMENT", "local"),
			OtlpEndpoint:     getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://tempo:4318"),
			TraceSampleRatio: getEnvFloat("TRACE_SAMPLE_RATIO", 1.0),
			LogLevel:         getEnv("LOG_LEVEL", "info"),
		}
	})
	return instance
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
