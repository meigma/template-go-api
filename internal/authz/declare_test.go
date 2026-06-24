package authz

import (
	"testing"

	"github.com/cedar-policy/cedar-go/types"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireRecordsActionAndSecurity(t *testing.T) {
	t.Parallel()

	action := types.NewEntityUID("Action", "todo:read")
	meta := Require(action, "todoID")

	decl, ok := meta[metadataKey].(*declaration)
	require.True(t, ok, "metadata must carry a declaration")
	assert.Equal(t, kindRequire, decl.kind)
	assert.Equal(t, action, decl.action)
	assert.Equal(t, "todoID", decl.idParam)

	security, ok := meta["security"].([]map[string][]string)
	require.True(t, ok, "Require must populate the OpenAPI security requirement")
	assert.Equal(t, SecurityRequirement(), security)
}

func TestRequireWithoutIDParam(t *testing.T) {
	t.Parallel()

	decl, ok := Require(types.NewEntityUID("Action", "todo:list"))[metadataKey].(*declaration)
	require.True(t, ok)
	assert.Empty(t, decl.idParam, "a collection operation binds no instance id")
}

func TestPublicRecordsOptOut(t *testing.T) {
	t.Parallel()

	meta := Public()
	decl, ok := meta[metadataKey].(*declaration)
	require.True(t, ok)
	assert.Equal(t, kindPublic, decl.kind)
	assert.NotContains(t, meta, "security", "a public operation declares no security requirement")
}

func TestDeclarationFrom(t *testing.T) {
	t.Parallel()

	t.Run("present", func(t *testing.T) {
		t.Parallel()

		op := &huma.Operation{Metadata: Public()}
		decl, ok := declarationFrom(op)
		require.True(t, ok)
		assert.Equal(t, kindPublic, decl.kind)
	})

	t.Run("undeclared operation", func(t *testing.T) {
		t.Parallel()

		_, ok := declarationFrom(&huma.Operation{})
		assert.False(t, ok, "an operation with no declaration must be reported as undeclared")
	})

	t.Run("nil operation", func(t *testing.T) {
		t.Parallel()

		_, ok := declarationFrom(nil)
		assert.False(t, ok)
	})
}

func TestResourceTypeFromAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action types.EntityUID
		want   types.EntityType
	}{
		{
			name:   "resource verb maps to pascal-cased type",
			action: types.NewEntityUID("Action", "todo:read"),
			want:   "Todo",
		},
		{
			name:   "multi-segment verb keeps only the resource",
			action: types.NewEntityUID("Action", "todo:list:all"),
			want:   "Todo",
		},
		{
			name:   "action without a resource prefix yields a zero type",
			action: types.NewEntityUID("Action", "ping"),
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, resourceTypeFromAction(tc.action).Type)
		})
	}
}
