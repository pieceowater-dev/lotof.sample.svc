package cfg

import (
	someEnt "app/internal/pkg/domainItem/ent"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"sync"
)

// Config holds the configuration settings for the application.
type Config struct {
	GrpcPort            string // Port for gRPC server
	RestPort            string // Port for REST server
	PostgresDatabaseDSN string // Data Source Name for PostgreSQL database
	PostgresModels      []any  // List of models for database migration
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
