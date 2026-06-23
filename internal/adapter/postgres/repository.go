package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/meigma/template-go-api/internal/adapter/postgres/sqlc"
	"github.com/meigma/template-go-api/internal/todo"
)

// TodoRepository is a PostgreSQL todo.Repository backed by a pgx connection pool
// and sqlc-generated queries.
type TodoRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewTodoRepository constructs a TodoRepository over the given pool.
func NewTodoRepository(pool *pgxpool.Pool) *TodoRepository {
	return &TodoRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Save inserts or replaces the todo via an upsert.
func (r *TodoRepository) Save(ctx context.Context, t todo.Todo) error {
	params, err := toUpsertParams(t)
	if err != nil {
		return err
	}
	if err := r.queries.UpsertTodo(ctx, params); err != nil {
		return fmt.Errorf("upsert todo %s: %w", t.ID, err)
	}

	return nil
}

// FindByID returns the stored todo or todo.ErrNotFound when absent.
func (r *TodoRepository) FindByID(ctx context.Context, id string) (todo.Todo, error) {
	uid, err := uuidParse(id)
	if err != nil {
		return todo.Todo{}, err
	}

	row, err := r.queries.GetTodo(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return todo.Todo{}, todo.ErrNotFound
		}

		return todo.Todo{}, fmt.Errorf("get todo %s: %w", id, err)
	}

	return fromRow(row)
}

// List returns all stored todos ordered by creation time.
func (r *TodoRepository) List(ctx context.Context) ([]todo.Todo, error) {
	rows, err := r.queries.ListTodos(ctx)
	if err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}

	todos := make([]todo.Todo, 0, len(rows))
	for _, row := range rows {
		t, err := fromRow(row)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}

	return todos, nil
}

// Ping verifies the pool can reach the database, for use as a readiness check.
func (r *TodoRepository) Ping(ctx context.Context) error {
	if err := r.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	return nil
}
