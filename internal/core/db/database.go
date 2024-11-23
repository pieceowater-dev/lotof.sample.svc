package db

import (
	"app/internal/core/db/pg"
	"fmt"
	"gorm.io/gorm"
)

// Database defines the common methods for database operations
type Database interface {
	GetDB() *gorm.DB
	WithTransaction(func(tx *gorm.DB) error) error
	SeedData(data interface{}) error
}

// DatabaseType defines the type of databases supported
type DatabaseType int

const (
	PostgresDB DatabaseType = iota
	// Future database types can be added here, e.g., MySQLDB, Mongo, Redis
)

// DatabaseFactory is a factory for creating database instances
type DatabaseFactory struct {
	DSN        string
	EnableLogs bool
}

// New initializes a new DatabaseFactory
func New(dsn string, enableLogs bool) *DatabaseFactory {
	return &DatabaseFactory{
		DSN:        dsn,
		EnableLogs: enableLogs,
	}
}

// Create creates a database instance based on the given type
func (f *DatabaseFactory) Create(dbType DatabaseType) (Database, error) {
	switch dbType {
	case PostgresDB:
		return pg.NewPostgres(f.DSN, f.EnableLogs), nil
	// Add more cases for other database types
	default:
		return nil, fmt.Errorf("unsupported database type: %v", dbType)
	}
}
