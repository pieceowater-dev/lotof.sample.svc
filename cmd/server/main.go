package main

import (
	"app/internal/core/cfg"
	pb "app/internal/core/grpc/generated"
	"context"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
)

// Mock Todo Data
var todos = []*pb.Todo{
	{Id: "1", Text: "Personal Todo", Category: pb.TodoCategoryEnum_PERSONAL, Done: false},
	{Id: "2", Text: "Work Todo", Category: pb.TodoCategoryEnum_WORK, Done: true},
}

type TodoServiceServer struct {
	pb.UnimplementedTodoServiceServer
	mu sync.Mutex // For thread-safe access
}

func (s *TodoServiceServer) GetTodos(ctx context.Context, req *pb.GetTodosRequest) (*pb.GetTodosResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Println("GetTodos called")
	return &pb.GetTodosResponse{
		Todos: todos,
	}, nil
}

func (s *TodoServiceServer) CreateTodo(ctx context.Context, req *pb.CreateTodoRequest) (*pb.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("CreateTodo called with text: %s, category: %v", req.Text, req.Category)

	// Create new Todo
	newTodo := &pb.Todo{
		Id:       "new-id", // Replace with actual ID generation logic
		Text:     req.Text,
		Category: req.Category,
		Done:     false,
	}
	todos = append(todos, newTodo)

	return newTodo, nil
}

func main() {
	// Start gRPC Server
	listener, err := net.Listen("tcp", ":"+cfg.Inst().AppPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTodoServiceServer(grpcServer, &TodoServiceServer{})

	log.Printf("gRPC server is running on port %s...\n", cfg.Inst().AppPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
