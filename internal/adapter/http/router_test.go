package http

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	discard := observability.NewLogger(io.Discard, slog.LevelError, "json")
	service := todo.NewService(memory.NewTodoRepository(), discard)
	handler := NewRouter(RouterDeps{
		Service:        service,
		Logger:         discard,
		Metrics:        observability.NewMetrics(),
		Version:        "test",
		RequestTimeout: testRequestTimeout,
		Readiness:      nil,
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

	var created TodoDTO
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
		Todos []TodoDTO `json:"todos"`
	}
	require.NoError(t, json.Unmarshal([]byte(resp.body), &listed))
	assert.Len(t, listed.Todos, 1)

	resp = doRequest(t, srv, http.MethodPost, "/todos/"+created.ID+"/complete", "")
	require.Equal(t, http.StatusOK, resp.status)

	var completed TodoDTO
	require.NoError(t, json.Unmarshal([]byte(resp.body), &completed))
	assert.Equal(t, "completed", completed.Status)
	assert.NotNil(t, completed.CompletedAt)

	resp = doRequest(t, srv, http.MethodPost, "/todos/missing/complete", "")
	require.Equal(t, http.StatusNotFound, resp.status)
}

func TestInfraRoutes(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)

	for _, path := range []string{"/healthz", "/readyz"} {
		resp := doRequest(t, srv, http.MethodGet, path, "")
		assert.Equal(t, http.StatusOK, resp.status, path)
	}

	doRequest(t, srv, http.MethodGet, "/todos", "")

	resp := doRequest(t, srv, http.MethodGet, "/metrics", "")
	require.Equal(t, http.StatusOK, resp.status)
	assert.Contains(t, resp.body, "http_requests_total")
	assert.Contains(t, resp.body, "go_goroutines")
}
