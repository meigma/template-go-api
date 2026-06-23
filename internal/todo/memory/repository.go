// Package memory provides an in-memory implementation of the todo outbound port.
// It stores todos in a mutex-guarded map and is the zero-infrastructure reference
// adapter; swap it for a real datastore by implementing todo.Repository.
package memory

import (
	"context"
	"sync"

	"github.com/meigma/template-go-api/internal/todo"
)

// TodoRepository is an in-memory todo.Repository that is safe for concurrent use.
type TodoRepository struct {
	mu    sync.RWMutex
	todos map[string]todo.Todo
}

// NewTodoRepository constructs an empty TodoRepository.
func NewTodoRepository() *TodoRepository {
	return &TodoRepository{
		mu:    sync.RWMutex{},
		todos: make(map[string]todo.Todo),
	}
}

// Save inserts or replaces the todo.
func (r *TodoRepository) Save(_ context.Context, t todo.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.todos[t.ID] = t

	return nil
}

// FindByID returns the stored todo or todo.ErrNotFound.
func (r *TodoRepository) FindByID(_ context.Context, id string) (todo.Todo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stored, ok := r.todos[id]
	if !ok {
		return todo.Todo{}, todo.ErrNotFound
	}

	return stored, nil
}

// List returns all stored todos in unspecified order.
func (r *TodoRepository) List(_ context.Context) ([]todo.Todo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	todos := make([]todo.Todo, 0, len(r.todos))
	for _, t := range r.todos {
		todos = append(todos, t)
	}

	return todos, nil
}
