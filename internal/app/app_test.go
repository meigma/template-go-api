package app_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cedar-policy/cedar-go/types"
	"github.com/danielgtaylor/huma/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/template-go-api/internal/app"
	"github.com/meigma/template-go-api/internal/authz"
	"github.com/meigma/template-go-api/internal/config"
	"github.com/meigma/template-go-api/internal/observability"
	"github.com/meigma/template-go-api/internal/todo/todotest"
)

// stubAuthenticator authenticates every request as a fixed principal carrying
// the given roles, so the wiring test can drive the authz-enabled server without
// a database-backed api-key store. It satisfies authz.Authenticator.
type stubAuthenticator struct {
	roles []string
}

func (s stubAuthenticator) Authenticate(_ huma.Context) (authz.Principal, error) {
	roleValues := make([]types.Value, 0, len(s.roles))
	for _, role := range s.roles {
		roleValues = append(roleValues, types.String(role))
	}

	return authz.Principal{
		UID: types.NewEntityUID("User", "test-user"),
		Claims: types.NewRecord(types.RecordMap{
			authz.RolesClaim: types.NewSet(roleValues...),
		}),
	}, nil
}

func TestAppWiring(t *testing.T) {
	t.Parallel()

	cfg := config.Load(viper.New())
	logger := observability.NewLogger(io.Discard, slog.LevelError, "json")
	// Inject an in-memory repository so the composition root wires a working
	// server without a database — the postgres path is covered by the
	// container-backed integration suite. Authorization defaults to ON now that
	// the routes are tagged, so inject a stub authenticator that grants the
	// "user" role the todo policy requires; without it the api-key store would
	// need a database.
	application, err := app.New(
		context.Background(), cfg, logger, "test",
		app.WithRepository(todotest.NewRepository()),
		app.WithAuthenticator(stubAuthenticator{roles: []string{"user"}}),
	)
	require.NoError(t, err)

	handler := application.Handler()
	require.NotNil(t, handler)

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, healthReq)
	assert.Equal(t, http.StatusOK, healthRec.Code)

	createReq := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"x"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	assert.Equal(t, http.StatusCreated, createRec.Code)
}

// TestAppWiringDeniesUnauthorized proves authorization is wired and enforced by
// default: a principal carrying no role the todo policy grants is denied (403),
// so the create operation never reaches the handler.
func TestAppWiringDeniesUnauthorized(t *testing.T) {
	t.Parallel()

	cfg := config.Load(viper.New())
	logger := observability.NewLogger(io.Discard, slog.LevelError, "json")
	application, err := app.New(
		context.Background(), cfg, logger, "test",
		app.WithRepository(todotest.NewRepository()),
		app.WithAuthenticator(stubAuthenticator{roles: nil}),
	)
	require.NoError(t, err)

	createReq := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"x"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	application.Handler().ServeHTTP(createRec, createReq)
	assert.Equal(t, http.StatusForbidden, createRec.Code)
}
