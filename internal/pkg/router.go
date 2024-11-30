package pkg

import (
	pb "app/internal/core/grpc/generated"
	"app/internal/pkg/todo"
	"github.com/gin-gonic/gin"
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

// InitGRPC initializes gRPC routes
func (r *Router) InitGRPC(grpcServer *grpc.Server) {
	// Register gRPC services
	pb.RegisterTodoServiceServer(grpcServer, r.todoModule.Controller)
}

// InitREST initializes REST routes using Gin
func (r *Router) InitREST(router *gin.Engine) {
	//api := router.Group("/api")
	{
		// Register GIN routes
		//api.GET("/todos", r.todoModule.Controller.ListREST)
	}
}
