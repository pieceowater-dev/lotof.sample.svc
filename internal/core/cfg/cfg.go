package cfg

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"sync"
)

type Config struct {
	GrpcPort            string
	RestPort            string
	PostgresDatabaseDSN string
	PostgresModels      []any
}

var (
	once     sync.Once
	instance *Config
)

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
			PostgresModels:      []any{
				// models to migration here:
				// &ent.MyModel{},
			},
		}
	})
	return instance
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
