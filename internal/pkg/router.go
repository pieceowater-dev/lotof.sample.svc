package pkg

import (
	pb "app/internal/core/grpc/generated"
	"app/internal/pkg/todo"
	"google.golang.org/grpc"
)

type Router struct {
	todoModule *todo.Module
}

func NewRouter() *Router {
	return &Router{
		todoModule: todo.New(),
	}
}

func (r *Router) Init(grpcServer *grpc.Server) {
	// Register gRPC services
	pb.RegisterTodoServiceServer(grpcServer, r.todoModule.Controller)
}
