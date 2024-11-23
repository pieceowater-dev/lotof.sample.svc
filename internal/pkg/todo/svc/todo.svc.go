package svc

import (
	pb "app/internal/core/grpc/generated"
	"log"
)

type TodoService struct {
}

func NewTodoService() *TodoService {
	return &TodoService{}
}

// Mock Todo Data
var todos = []*pb.Todo{
	{Id: "1", Text: "Personal Todo", Category: pb.TodoCategoryEnum_PERSONAL, Done: false},
	{Id: "2", Text: "Work Todo", Category: pb.TodoCategoryEnum_WORK, Done: true},
}

func (s *TodoService) GetTodos() ([]*pb.Todo, error) {
	log.Println("Fetching todos...")
	return todos, nil
}
