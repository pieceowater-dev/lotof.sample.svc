package pg

import (
	"app/internal/pkg/todo/ent"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

type Postgres struct {
	DB *gorm.DB
}

// NewPostgres initializes the Postgres instance with a configurable logger
func NewPostgres(dsn string, enableLogs bool) *Postgres {
	var newLogger logger.Interface
	if enableLogs {
		newLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	} else {
		newLogger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get DB instance: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	//if err := db.AutoMigrate(&ent.Todo{}); err != nil {
	//	log.Fatalf("Failed to migrate database: %v", err)
	//}

	return &Postgres{DB: db}
}

// GetDB returns the GORM database instance
func (p *Postgres) GetDB() *gorm.DB {
	return p.DB
}

// WithTransaction executes a function within a transaction
func (p *Postgres) WithTransaction(fn func(tx *gorm.DB) error) error {
	tx := p.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// SeedData populates the database with dynamic initial data
func (p *Postgres) SeedData(data any) error {
	todos, ok := data.([]ent.Todo)
	if !ok {
		return fmt.Errorf("invalid data type, expected []ent.Todo")
	}
	for _, todo := range todos {
		p.DB.FirstOrCreate(&todo, ent.Todo{Text: todo.Text})
	}
	return nil
}
