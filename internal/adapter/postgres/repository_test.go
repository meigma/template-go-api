//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/template-go-api/internal/todo"
)

// pgTimePrecision is Postgres timestamptz's resolution. Go [time.Time] carries
// nanoseconds, so todos are truncated to this precision before saving to get an
// exact round-trip comparison.
const pgTimePrecision = time.Microsecond

// makeTodo builds a valid open todo with a fresh uuid, truncated to the
// precision Postgres preserves so reads compare equal to writes.
func makeTodo(t *testing.T, title string) todo.Todo {
	t.Helper()

	created, err := todo.NewTodo(uuid.NewString(), title, time.Now().UTC().Truncate(pgTimePrecision))
	require.NoError(t, err)

	return created
}

// assertTodoEqual compares two todos field by field. Times are compared with
// [time.Time.Equal] so a round trip through Postgres (which returns timestamptz
// in a different [time.Location] than the UTC value written) still matches on
// the underlying instant rather than the wall-clock representation.
func assertTodoEqual(t *testing.T, want, got todo.Todo) {
	t.Helper()

	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.Title, got.Title)
	assert.Equal(t, want.Status, got.Status)
	assert.True(t, want.CreatedAt.Equal(got.CreatedAt),
		"created_at: want %s, got %s", want.CreatedAt, got.CreatedAt)

	if want.CompletedAt == nil {
		assert.Nil(t, got.CompletedAt)

		return
	}

	require.NotNil(t, got.CompletedAt)
	assert.True(t, want.CompletedAt.Equal(*got.CompletedAt),
		"completed_at: want %s, got %s", *want.CompletedAt, *got.CompletedAt)
}

// TestRepository exercises the PostgreSQL adapter against the same behavioral
// contract the in-memory adapter satisfies, plus the upsert and time-replacement
// semantics specific to the SQL store. It shares one migrated container and
// restores the clean snapshot between subtests for isolation, so the subtests
// run sequentially rather than in parallel.
func TestRepository(t *testing.T) {
	ctx := context.Background()
	fix := setupPostgres(ctx, t)

	t.Run("SaveAndFind", func(t *testing.T) {
		repo := fix.Reset(ctx, t)

		want := makeTodo(t, "buy milk")
		require.NoError(t, repo.Save(ctx, want))

		got, err := repo.FindByID(ctx, want.ID)
		require.NoError(t, err)
		assertTodoEqual(t, want, got)
	})

	t.Run("FindMissingIsNotFound", func(t *testing.T) {
		repo := fix.Reset(ctx, t)

		_, err := repo.FindByID(ctx, uuid.NewString())
		require.ErrorIs(t, err, todo.ErrNotFound)
	})

	t.Run("FindInvalidIDIsNotFound", func(t *testing.T) {
		repo := fix.Reset(ctx, t)

		// A syntactically invalid id can never match a stored uuid, so the
		// adapter maps it to ErrNotFound like the in-memory peer.
		_, err := repo.FindByID(ctx, "not-a-uuid")
		require.ErrorIs(t, err, todo.ErrNotFound)
	})

	t.Run("List", func(t *testing.T) {
		repo := fix.Reset(ctx, t)

		require.NoError(t, repo.Save(ctx, makeTodo(t, "first")))
		require.NoError(t, repo.Save(ctx, makeTodo(t, "second")))

		got, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("ListEmpty", func(t *testing.T) {
		repo := fix.Reset(ctx, t)

		got, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("ListOrdersByCreatedAt", func(t *testing.T) {
		repo := fix.Reset(ctx, t)

		older := makeTodo(t, "older")
		newer := makeTodo(t, "newer")
		newer.CreatedAt = older.CreatedAt.Add(time.Hour)

		// Save out of order to prove the query, not insertion order, sorts.
		require.NoError(t, repo.Save(ctx, newer))
		require.NoError(t, repo.Save(ctx, older))

		got, err := repo.List(ctx)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, older.ID, got[0].ID)
		assert.Equal(t, newer.ID, got[1].ID)
	})

	t.Run("SaveUpsertReplaces", func(t *testing.T) {
		repo := fix.Reset(ctx, t)

		original := makeTodo(t, "draft")
		require.NoError(t, repo.Save(ctx, original))

		// Re-save under the same id: Save is a full insert-or-replace, so every
		// mutable field (including created_at) is overwritten, matching the
		// in-memory adapter exactly.
		completedAt := original.CreatedAt.Add(2 * time.Hour)
		updated := todo.Todo{
			ID:          original.ID,
			Title:       "final",
			Status:      todo.StatusCompleted,
			CreatedAt:   original.CreatedAt.Add(time.Hour),
			CompletedAt: &completedAt,
		}
		require.NoError(t, repo.Save(ctx, updated))

		got, err := repo.FindByID(ctx, original.ID)
		require.NoError(t, err)
		assertTodoEqual(t, updated, got)

		// The upsert replaces rather than inserts, so there is still one row.
		all, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Len(t, all, 1)
	})

	t.Run("SaveAndFindCompleted", func(t *testing.T) {
		repo := fix.Reset(ctx, t)

		open := makeTodo(t, "finish report")
		completed := open.Complete(time.Now().UTC().Truncate(pgTimePrecision))
		require.NoError(t, repo.Save(ctx, completed))

		got, err := repo.FindByID(ctx, completed.ID)
		require.NoError(t, err)
		assertTodoEqual(t, completed, got)
	})

	t.Run("IsolationAfterReset", func(t *testing.T) {
		repo := fix.Reset(ctx, t)

		// Earlier subtests wrote rows; Reset must have restored the empty
		// snapshot so this test starts clean.
		got, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
