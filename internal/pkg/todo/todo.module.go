package todo

import (
	"app/internal/pkg/todo/ctrl"
	"app/internal/pkg/todo/svc"
)

type Module struct {
	Controller *ctrl.TodoController
}

func New() *Module {
	controller := ctrl.NewTodoController(svc.NewTodoService())
	return &Module{
		Controller: controller,
	}
}
