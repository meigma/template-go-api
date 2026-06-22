package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/meigma/template-go-api/internal/todo"
)

// tagTodos groups the todo operations in the OpenAPI document.
const tagTodos = "Todos"

// todoHandlers adapts the todo service to Huma operation handlers.
type todoHandlers struct {
	service *todo.Service
}

// register wires the todo operations onto api. API routes go through Huma, so
// they are validated and appear in the OpenAPI specification.
func (h *todoHandlers) register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-todo",
		Method:        http.MethodPost,
		Path:          "/todos",
		Summary:       "Create a todo",
		Tags:          []string{tagTodos},
		DefaultStatus: http.StatusCreated,
	}, h.create)

	huma.Register(api, huma.Operation{
		OperationID: "list-todos",
		Method:      http.MethodGet,
		Path:        "/todos",
		Summary:     "List todos",
		Tags:        []string{tagTodos},
	}, h.list)

	huma.Register(api, huma.Operation{
		OperationID: "get-todo",
		Method:      http.MethodGet,
		Path:        "/todos/{id}",
		Summary:     "Get a todo by id",
		Tags:        []string{tagTodos},
		Errors:      []int{http.StatusNotFound},
	}, h.get)

	huma.Register(api, huma.Operation{
		OperationID: "complete-todo",
		Method:      http.MethodPost,
		Path:        "/todos/{id}/complete",
		Summary:     "Mark a todo as completed",
		Tags:        []string{tagTodos},
		Errors:      []int{http.StatusNotFound},
	}, h.complete)
}

func (h *todoHandlers) create(ctx context.Context, input *CreateTodoInput) (*TodoOutput, error) {
	created, err := h.service.Create(ctx, input.Body.Title)
	if err != nil {
		return nil, toHumaError(err)
	}

	return &TodoOutput{Body: toTodoDTO(created)}, nil
}

func (h *todoHandlers) get(ctx context.Context, input *GetTodoInput) (*TodoOutput, error) {
	found, err := h.service.Get(ctx, input.ID)
	if err != nil {
		return nil, toHumaError(err)
	}

	return &TodoOutput{Body: toTodoDTO(found)}, nil
}

func (h *todoHandlers) list(ctx context.Context, _ *struct{}) (*ListTodosOutput, error) {
	todos, err := h.service.List(ctx)
	if err != nil {
		return nil, toHumaError(err)
	}

	out := &ListTodosOutput{}
	out.Body.Todos = toTodoDTOs(todos)

	return out, nil
}

func (h *todoHandlers) complete(ctx context.Context, input *CompleteTodoInput) (*TodoOutput, error) {
	completed, err := h.service.Complete(ctx, input.ID)
	if err != nil {
		return nil, toHumaError(err)
	}

	return &TodoOutput{Body: toTodoDTO(completed)}, nil
}
