package memory

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/template-go-api/internal/todo"
)

const concurrentSaves = 50

func makeTodo(t *testing.T, id string) todo.Todo {
	t.Helper()

	created, err := todo.NewTodo(id, "title-"+id, time.Now())
	require.NoError(t, err)

	return created
}

func TestRepositorySaveAndFind(t *testing.T) {
	t.Parallel()

	repo := NewTodoRepository()
	ctx := context.Background()
	want := makeTodo(t, "a")

	require.NoError(t, repo.Save(ctx, want))

	got, err := repo.FindByID(ctx, "a")
	require.NoError(t, err)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.Title, got.Title)
}

func TestRepositoryFindMissing(t *testing.T) {
	t.Parallel()

	repo := NewTodoRepository()

	_, err := repo.FindByID(context.Background(), "missing")
	require.ErrorIs(t, err, todo.ErrNotFound)
}

func TestRepositoryList(t *testing.T) {
	t.Parallel()

	repo := NewTodoRepository()
	ctx := context.Background()
	require.NoError(t, repo.Save(ctx, makeTodo(t, "a")))
	require.NoError(t, repo.Save(ctx, makeTodo(t, "b")))

	got, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestRepositoryConcurrentSave(t *testing.T) {
	t.Parallel()

	repo := NewTodoRepository()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := range concurrentSaves {
		wg.Go(func() {
			// Build the todo inline (no require) since this runs off the test goroutine.
			created, _ := todo.NewTodo(strconv.Itoa(i), "concurrent", time.Now())
			_ = repo.Save(ctx, created)
		})
	}
	wg.Wait()

	got, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, got, concurrentSaves)
}
