package httpapi

import "time"

// TodoDTO is the transport representation of a todo.
type TodoDTO struct {
	ID          string     `json:"id"                    doc:"Unique identifier of the todo."`
	Title       string     `json:"title"                 doc:"Human-readable title of the todo."`
	Status      string     `json:"status"                doc:"Lifecycle state of the todo."         enum:"open,completed"`
	CreatedAt   time.Time  `json:"createdAt"             doc:"When the todo was created."`
	CompletedAt *time.Time `json:"completedAt,omitempty" doc:"When the todo was completed, if set."`
}

// CreateTodoInput is the request body for creating a todo.
type CreateTodoInput struct {
	Body struct {
		Title string `json:"title" minLength:"1" maxLength:"200" doc:"Human-readable title of the todo."`
	}
}

// GetTodoInput identifies a todo by its path id.
type GetTodoInput struct {
	ID string `path:"id" maxLength:"64" doc:"Unique identifier of the todo."`
}

// CompleteTodoInput identifies a todo to complete by its path id.
type CompleteTodoInput struct {
	ID string `path:"id" maxLength:"64" doc:"Unique identifier of the todo."`
}

// TodoOutput wraps a single todo response body.
type TodoOutput struct {
	Body TodoDTO
}

// ListTodosOutput wraps the todo collection response body.
type ListTodosOutput struct {
	Body struct {
		Todos []TodoDTO `json:"todos" doc:"All stored todos."`
	}
}
