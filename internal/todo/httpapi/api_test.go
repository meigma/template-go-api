package httpapi_test

import (
	"bytes"
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
	"github.com/meigma/template-go-api/internal/observability"
	"github.com/meigma/template-go-api/internal/todo"
	"github.com/meigma/template-go-api/internal/todo/httpapi"
	"github.com/meigma/template-go-api/internal/todo/todotest"
)

const testRequestTimeout = 5 * time.Second

type testResponse struct {
	status      int
	body        string
	contentType string
}

// newTestServer wires the real generic router to the real todo adapter and a
// stateful in-memory test repository, exercising the full request path end to end.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	discard := observability.NewLogger(io.Discard, slog.LevelError, "json")
	service := todo.NewService(todotest.NewRepository(), discard)
	handler := adapterhttp.NewRouter(adapterhttp.RouterDeps{
		Logger:         discard,
		Metrics:        observability.NewMetrics(),
		Version:        "test",
		RequestTimeout: testRequestTimeout,
		Readiness:      nil,
		Register:       func(api huma.API) { httpapi.Register(api, service) },
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

	var created httpapi.TodoDTO
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
		Todos []httpapi.TodoDTO `json:"todos"`
	}
	require.NoError(t, json.Unmarshal([]byte(resp.body), &listed))
	assert.Len(t, listed.Todos, 1)

	resp = doRequest(t, srv, http.MethodPost, "/todos/"+created.ID+"/complete", "")
	require.Equal(t, http.StatusOK, resp.status)

	var completed httpapi.TodoDTO
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

// TestServiceLogCarriesRequestID proves the request-scoped logger reaches the
// domain: the service's "todo created" line must carry the same request_id as
// the access-log line. The handler is driven synchronously (no live server) so
// the shared buffer is written and read on one goroutine, avoiding a data race.
func TestServiceLogCarriesRequestID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := observability.NewLogger(&buf, slog.LevelInfo, "json")
	service := todo.NewService(todotest.NewRepository(), logger)
	handler := adapterhttp.NewRouter(adapterhttp.RouterDeps{
		Logger:         logger,
		Metrics:        observability.NewMetrics(),
		Version:        "test",
		RequestTimeout: testRequestTimeout,
		Readiness:      nil,
		Register:       func(api huma.API) { httpapi.Register(api, service) },
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"buy milk"}`))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	created := findLogEntry(t, buf.String(), "todo created")
	access := findLogEntry(t, buf.String(), "http request")
	require.NotEmpty(t, created["request_id"])
	assert.Equal(t, access["request_id"], created["request_id"])
}

// findLogEntry returns the first JSON log line in logs whose msg equals want.
func findLogEntry(t *testing.T, logs, want string) map[string]any {
	t.Helper()

	for line := range strings.SplitSeq(strings.TrimSpace(logs), "\n") {
		if line == "" {
			continue
		}

		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry["msg"] == want {
			return entry
		}
	}

	t.Fatalf("no log entry with msg %q in:\n%s", want, logs)

	return nil
}
