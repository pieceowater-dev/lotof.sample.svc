package todo

import (
	"app/internal/core/cfg"
	"app/internal/pkg/todo/ctrl"
	"app/internal/pkg/todo/svc"
	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"log"
)

type Module struct {
	Controller *ctrl.TodoController
}

func New() *Module {
	// Create database instance
	database, err := gossiper.NewDB(
		gossiper.PostgresDB,
		cfg.Inst().PostgresDatabaseDSN,
		false,
	)
	if err != nil {
		log.Fatalf("Failed to create database instance: %v", err)
	}

	// Initialize and return the module
	return &Module{
		Controller: ctrl.NewTodoController(
			svc.NewTodoService(database),
		),
	}
}
