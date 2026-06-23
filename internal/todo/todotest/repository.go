// Package todotest provides test-only doubles for the todo domain ports.
//
// Repository here is a real, stateful in-memory store for tests that exercise a
// create-then-read flow end to end (for example the HTTP functional tests). For
// interaction or error-injection assertions, prefer the generated mock in
// internal/todo/mocks instead.
package todotest

import (
	"context"
	"sync"

	"github.com/meigma/template-go-api/internal/todo"
)

// Repository is an in-memory todo.Repository that is safe for concurrent use.
type Repository struct {
	mu    sync.RWMutex
	todos map[string]todo.Todo
}

// NewRepository constructs an empty Repository.
func NewRepository() *Repository {
	return &Repository{
		mu:    sync.RWMutex{},
		todos: make(map[string]todo.Todo),
	}
}

// Save inserts or replaces the todo.
func (r *Repository) Save(_ context.Context, t todo.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.todos[t.ID] = t

	return nil
}

// FindByID returns the stored todo or todo.ErrNotFound.
func (r *Repository) FindByID(_ context.Context, id string) (todo.Todo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stored, ok := r.todos[id]
	if !ok {
		return todo.Todo{}, todo.ErrNotFound
	}

	return stored, nil
}

// List returns all stored todos in unspecified order.
func (r *Repository) List(_ context.Context) ([]todo.Todo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	todos := make([]todo.Todo, 0, len(r.todos))
	for _, t := range r.todos {
		todos = append(todos, t)
	}

	return todos, nil
}
