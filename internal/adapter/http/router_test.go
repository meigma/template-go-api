package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/template-go-api/internal/adapter/http/problem"
	"github.com/meigma/template-go-api/internal/observability"
)

const testRequestTimeout = 5 * time.Second

// TestInfraRoutesWithoutResources exercises the generic transport with no resource
// registrations (Register is nil): the infrastructure routes must still serve.
func TestInfraRoutesWithoutResources(t *testing.T) {
	t.Parallel()

	discard := observability.NewLogger(io.Discard, slog.LevelError, "json")
	handler := NewRouter(RouterDeps{
		Logger:         discard,
		Metrics:        observability.NewMetrics(),
		Version:        "test",
		RequestTimeout: testRequestTimeout,
		Readiness:      nil,
		Register:       nil,
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	for _, path := range []string{"/healthz", "/readyz"} {
		resp := get(t, srv, path)
		assert.Equal(t, http.StatusOK, resp.status, path)
	}

	get(t, srv, "/healthz")

	metrics := get(t, srv, "/metrics")
	require.Equal(t, http.StatusOK, metrics.status)
	assert.Contains(t, metrics.body, "http_requests_total")
	assert.Contains(t, metrics.body, "go_goroutines")
}

// TestReadyzReflectsChecks verifies the named readiness seam: every check runs
// (no short-circuit) and each result is reported by name, with the overall
// status reflecting whether any check failed.
func TestReadyzReflectsChecks(t *testing.T) {
	t.Parallel()

	discard := observability.NewLogger(io.Discard, slog.LevelError, "json")

	ok := func(context.Context) error { return nil }
	down := func(context.Context) error { return errors.New("down") }

	tests := []struct {
		name       string
		checks     []ReadinessCheck
		wantStatus int
		wantBody   map[string]string
	}{
		{
			name:       "no checks is ready",
			checks:     nil,
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{},
		},
		{
			name:       "all pass",
			checks:     []ReadinessCheck{{Name: "store", Check: ok}},
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"store": "ok"},
		},
		{
			name:       "any failure is unavailable",
			checks:     []ReadinessCheck{{Name: "store", Check: ok}, {Name: "cache", Check: down}},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   map[string]string{"store": "ok", "cache": "unavailable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := NewRouter(RouterDeps{
				Logger:         discard,
				Metrics:        observability.NewMetrics(),
				Version:        "test",
				RequestTimeout: testRequestTimeout,
				Readiness:      tt.checks,
				Register:       nil,
			})

			srv := httptest.NewServer(handler)
			t.Cleanup(srv.Close)

			resp := get(t, srv, "/readyz")
			assert.Equal(t, tt.wantStatus, resp.status)

			var body readyzResponse
			require.NoError(t, json.Unmarshal([]byte(resp.body), &body))
			assert.Equal(t, tt.wantBody, body.Checks)
		})
	}
}

// TestNotFoundReturnsProblemJSON verifies the chi NotFound fallback emits RFC 9457
// problem+json instead of chi's text/plain default.
func TestNotFoundReturnsProblemJSON(t *testing.T) {
	t.Parallel()

	discard := observability.NewLogger(io.Discard, slog.LevelError, "json")
	handler := NewRouter(RouterDeps{
		Logger:         discard,
		Metrics:        observability.NewMetrics(),
		Version:        "test",
		RequestTimeout: testRequestTimeout,
		Readiness:      nil,
		Register:       nil,
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	resp := get(t, srv, "/does-not-exist")
	assert.Equal(t, http.StatusNotFound, resp.status)
	assert.Equal(t, problem.ContentType, resp.contentType)
	assert.Contains(t, resp.body, `"status":404`)
}

type testResponse struct {
	status      int
	body        string
	contentType string
}

func get(t *testing.T, srv *httptest.Server, path string) testResponse {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+path, nil)
	require.NoError(t, err)

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
