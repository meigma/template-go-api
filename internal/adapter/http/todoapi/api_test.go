package todoapi_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapterhttp "github.com/meigma/template-go-api/internal/adapter/http"
	"github.com/meigma/template-go-api/internal/adapter/http/todoapi"
	"github.com/meigma/template-go-api/internal/adapter/memory"
	"github.com/meigma/template-go-api/internal/observability"
	"github.com/meigma/template-go-api/internal/todo"
)

const testRequestTimeout = 5 * time.Second

type testResponse struct {
	status      int
	body        string
	contentType string
}

// newTestServer wires the real generic router to the real todo adapter and the
// real in-memory store, exercising the full request path end to end.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	discard := observability.NewLogger(io.Discard, slog.LevelError, "json")
	service := todo.NewService(memory.NewTodoRepository(), discard)
	handler := adapterhttp.NewRouter(adapterhttp.RouterDeps{
		Logger:         discard,
		Metrics:        observability.NewMetrics(),
		Version:        "test",
		RequestTimeout: testRequestTimeout,
		Readiness:      nil,
		Register:       func(api huma.API) { todoapi.Register(api, service) },
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return srv
}

func doRequest(t *testing.T, srv *httptest.Server, method, path, body string) testResponse {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, srv.URL+path, reader)
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return testResponse{
		status:      resp.StatusCode,
		body:        string(data),
		contentType: resp.Header.Get("Content-Type"),
	}
}

func TestTodoAPIFunctional(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)

	resp := doRequest(t, srv, http.MethodPost, "/todos", `{"title":"buy milk"}`)
	require.Equal(t, http.StatusCreated, resp.status)

	var created todoapi.TodoDTO
	require.NoError(t, json.Unmarshal([]byte(resp.body), &created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "buy milk", created.Title)
	assert.Equal(t, "open", created.Status)

	resp = doRequest(t, srv, http.MethodGet, "/todos/"+created.ID, "")
	require.Equal(t, http.StatusOK, resp.status)

	resp = doRequest(t, srv, http.MethodGet, "/todos/does-not-exist", "")
	require.Equal(t, http.StatusNotFound, resp.status)
	assert.Contains(t, resp.contentType, "application/problem+json")

	resp = doRequest(t, srv, http.MethodPost, "/todos", `{"title":""}`)
	require.Equal(t, http.StatusUnprocessableEntity, resp.status)
	assert.Contains(t, resp.contentType, "application/problem+json")

	resp = doRequest(t, srv, http.MethodGet, "/todos", "")
	require.Equal(t, http.StatusOK, resp.status)

	var listed struct {
		Todos []todoapi.TodoDTO `json:"todos"`
	}
	require.NoError(t, json.Unmarshal([]byte(resp.body), &listed))
	assert.Len(t, listed.Todos, 1)

	resp = doRequest(t, srv, http.MethodPost, "/todos/"+created.ID+"/complete", "")
	require.Equal(t, http.StatusOK, resp.status)

	var completed todoapi.TodoDTO
	require.NoError(t, json.Unmarshal([]byte(resp.body), &completed))
	assert.Equal(t, "completed", completed.Status)
	assert.NotNil(t, completed.CompletedAt)

	resp = doRequest(t, srv, http.MethodPost, "/todos/missing/complete", "")
	require.Equal(t, http.StatusNotFound, resp.status)

	// A valid path with an unregistered method returns RFC 9457 problem+json.
	resp = doRequest(t, srv, http.MethodDelete, "/todos", "")
	require.Equal(t, http.StatusMethodNotAllowed, resp.status)
	assert.Contains(t, resp.contentType, "application/problem+json")
}
