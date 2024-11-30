package svc

import (
	"app/internal/core/grpc/generated"
	"app/internal/pkg/todo/ent"
	"fmt"
	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"log"
)

type TodoService struct {
	db gossiper.Database
}

func NewTodoService(db gossiper.Database) *TodoService {
	return &TodoService{db: db}
}

// GetTodos fetches all todos from the database
func (s *TodoService) GetTodos() ([]*generated.Todo, error) {
	log.Println("Fetching todos from database...")

	// Fetch todos using the database interface
	var items []ent.Todo
	if err := s.db.GetDB().Find(&items).Error; err != nil {
		return nil, err
	}

	// Convert database records to gRPC-compatible responses
	var todos []*generated.Todo
	for _, item := range items {
		todos = append(todos, &generated.Todo{
			Id:   fmt.Sprintf("%d", item.ID),
			Text: item.Text,
			Done: item.Done,
		})
	}

	return todos, nil
}
