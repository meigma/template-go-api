package todo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	saved   map[string]Todo
	listErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{saved: map[string]Todo{}, listErr: nil}
}

func (f *fakeRepo) Save(_ context.Context, t Todo) error {
	f.saved[t.ID] = t

	return nil
}

func (f *fakeRepo) FindByID(_ context.Context, id string) (Todo, error) {
	stored, ok := f.saved[id]
	if !ok {
		return Todo{}, ErrNotFound
	}

	return stored, nil
}

func (f *fakeRepo) List(_ context.Context) ([]Todo, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}

	todos := make([]Todo, 0, len(f.saved))
	for _, t := range f.saved {
		todos = append(todos, t)
	}

	return todos, nil
}

func newTestService(repo Repository) *Service {
	fixed := time.Date(2026, time.June, 22, 13, 0, 0, 0, time.UTC)

	return NewService(repo, nil,
		WithClock(func() time.Time { return fixed }),
		WithIDGenerator(func() string { return "fixed-id" }),
	)
}

func TestServiceCreate(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := newTestService(repo)

	got, err := svc.Create(context.Background(), "buy milk")
	require.NoError(t, err)
	assert.Equal(t, "fixed-id", got.ID)
	assert.Equal(t, StatusOpen, got.Status)
	assert.Contains(t, repo.saved, "fixed-id")
}

func TestServiceCreateInvalid(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := newTestService(repo)

	_, err := svc.Create(context.Background(), "   ")
	require.ErrorIs(t, err, ErrInvalidTitle)
	assert.Empty(t, repo.saved)
}

func TestServiceGetNotFound(t *testing.T) {
	t.Parallel()

	svc := newTestService(newFakeRepo())

	_, err := svc.Get(context.Background(), "missing")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceComplete(t *testing.T) {
	t.Parallel()

	svc := newTestService(newFakeRepo())

	created, err := svc.Create(context.Background(), "buy milk")
	require.NoError(t, err)

	completed, err := svc.Complete(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, completed.Status)
	require.NotNil(t, completed.CompletedAt)
}

func TestServiceCompleteNotFound(t *testing.T) {
	t.Parallel()

	svc := newTestService(newFakeRepo())

	_, err := svc.Complete(context.Background(), "missing")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceListPropagatesError(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	repo.listErr = errors.New("boom")
	svc := newTestService(repo)

	_, err := svc.List(context.Background())
	require.Error(t, err)
}
