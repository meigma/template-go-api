package todo

import "context"

// Repository is the outbound port the todo service depends on for persistence.
// It is declared here, by its consumer, and implemented by adapters (for example,
// an in-memory or SQL store).
type Repository interface {
	// Save persists the todo, inserting or replacing any existing entry.
	Save(ctx context.Context, todo Todo) error
	// FindByID returns the todo with the given id, or ErrNotFound if absent.
	FindByID(ctx context.Context, id string) (Todo, error)
	// List returns all stored todos.
	List(ctx context.Context) ([]Todo, error)
}
