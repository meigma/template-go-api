package http

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// TestReadyzReflectsChecks verifies the ReadinessCheck seam: a failing check
// yields 503, and a passing check yields 200.
func TestReadyzReflectsChecks(t *testing.T) {
	t.Parallel()

	discard := observability.NewLogger(io.Discard, slog.LevelError, "json")

	tests := []struct {
		name  string
		check ReadinessCheck
		want  int
	}{
		{name: "ready", check: func(context.Context) error { return nil }, want: http.StatusOK},
		{
			name:  "unavailable",
			check: func(context.Context) error { return errors.New("down") },
			want:  http.StatusServiceUnavailable,
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
				Readiness:      []ReadinessCheck{tt.check},
				Register:       nil,
			})

			srv := httptest.NewServer(handler)
			t.Cleanup(srv.Close)

			assert.Equal(t, tt.want, get(t, srv, "/readyz").status)
		})
	}
}

type testResponse struct {
	status int
	body   string
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

	return testResponse{status: resp.StatusCode, body: string(data)}
}
