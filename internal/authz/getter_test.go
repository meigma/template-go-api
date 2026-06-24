package authz

import (
	"context"
	"errors"
	"testing"

	"github.com/cedar-policy/cedar-go/types"
	"github.com/stretchr/testify/assert"
)

// recordingResolver owns one entity type and counts how many times Resolve is
// called, so the getter's caching can be observed.
type recordingResolver struct {
	typ    string
	uid    types.EntityUID
	calls  int
	err    error
	sinkOf context.Context
}

func (r *recordingResolver) Types() []string { return []string{r.typ} }

func (r *recordingResolver) Resolve(uid types.EntityUID) (types.Entity, bool) {
	r.calls++
	if r.err != nil {
		RecordLoadError(r.sinkOf, r.err)

		return types.Entity{}, false
	}
	if uid != r.uid {
		return types.Entity{}, false
	}

	return types.Entity{UID: uid}, true
}

func TestGetterRoutesAndCaches(t *testing.T) {
	t.Parallel()

	todoUID := types.NewEntityUID("Todo", "1")
	resolver := &recordingResolver{typ: "Todo", uid: todoUID}
	g := newGetter(context.Background(), Anonymous(), []Contribution{{
		Resolver: func(ctx context.Context, _ Principal) EntityResolver {
			resolver.sinkOf = ctx

			return resolver
		},
	}})

	entity, ok := g.Get(todoUID)
	assert.True(t, ok)
	assert.Equal(t, todoUID, entity.UID)

	// A second lookup of the same uid is served from the cache.
	_, _ = g.Get(todoUID)
	assert.Equal(t, 1, resolver.calls, "the entity should be loaded once and cached")
	assert.NoError(t, g.Err())
}

func TestGetterCachesMisses(t *testing.T) {
	t.Parallel()

	resolver := &recordingResolver{typ: "Todo", uid: types.NewEntityUID("Todo", "1")}
	g := newGetter(context.Background(), Anonymous(), []Contribution{{
		Resolver: func(ctx context.Context, _ Principal) EntityResolver {
			resolver.sinkOf = ctx

			return resolver
		},
	}})

	missing := types.NewEntityUID("Todo", "404")
	_, ok := g.Get(missing)
	assert.False(t, ok)
	_, _ = g.Get(missing)
	assert.Equal(t, 1, resolver.calls, "a miss is cached so it is not retried")
}

func TestGetterUnownedTypeIsMiss(t *testing.T) {
	t.Parallel()

	g := newGetter(context.Background(), Anonymous(), nil)
	_, ok := g.Get(types.NewEntityUID("Unknown", "x"))
	assert.False(t, ok)
	assert.NoError(t, g.Err(), "an unowned type is a miss, not an error")
}

func TestGetterCapturesFirstLoadError(t *testing.T) {
	t.Parallel()

	first := errors.New("first failure")
	resolver := &recordingResolver{typ: "Todo", err: first}
	g := newGetter(context.Background(), Anonymous(), []Contribution{{
		Resolver: func(ctx context.Context, _ Principal) EntityResolver {
			resolver.sinkOf = ctx

			return resolver
		},
	}})

	_, ok := g.Get(types.NewEntityUID("Todo", "1"))
	assert.False(t, ok)
	assert.ErrorIs(t, g.Err(), first, "the resolver's load error must be captured for fail-closed handling")
}
