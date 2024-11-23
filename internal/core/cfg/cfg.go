package cfg

import (
	"app/internal/pkg/todo/ent"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"sync"
)

type Config struct {
	AppPort             string
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
			AppPort:             getEnv("APP_PORT", "50051"),
			PostgresDatabaseDSN: getEnv("POSTGRES_DB_DSN", "postgres://pieceouser:pieceopassword@localhost:5432/sample?sslmode=disable"),
			PostgresModels: []any{
				&ent.Todo{},
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
