package ctrl

import (
	pb "app/internal/core/grpc/generated"
	"app/internal/pkg/todo/svc"
	"context"
)

type TodoController struct {
	todoService *svc.TodoService
	pb.UnimplementedTodoServiceServer
}

func NewTodoController(service *svc.TodoService) *TodoController {
	return &TodoController{todoService: service}
}

func (c *TodoController) GetTodos(ctx context.Context, req *pb.GetTodosRequest) (*pb.GetTodosResponse, error) {
	// Delegate to the service
	todos, err := c.todoService.GetTodos()
	if err != nil {
		return nil, err
	}

	return &pb.GetTodosResponse{
		Todos: todos,
	}, nil
}
