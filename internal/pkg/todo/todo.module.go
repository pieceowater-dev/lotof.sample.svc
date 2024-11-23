package todo

import (
	"app/internal/core/cfg"
	"app/internal/core/db"
	"app/internal/pkg/todo/ctrl"
	"app/internal/pkg/todo/svc"
	"log"
)

type Module struct {
	Controller *ctrl.TodoController
}

func New() *Module {
	// Create database instance
	database, err := db.New(
		cfg.Inst().PostgresDatabaseDSN,
		false,
	).Create(db.PostgresDB)
	if err != nil {
		log.Fatalf("Failed to create database instance: %v", err)
	}

	// Seed example data
	//data := []ent.Todo{
	//	{Text: "Learn Interfaces", Done: false},
	//	{Text: "Implement Factory", Done: true},
	//}
	//if err := dbInstance.SeedData(data); err != nil {
	//	log.Fatalf("Failed to seed data: %v", err)
	//}

	// Initialize and return the module
	return &Module{
		Controller: ctrl.NewTodoController(
			svc.NewTodoService(database),
		),
	}
}
