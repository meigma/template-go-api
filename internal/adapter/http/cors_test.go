package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/template-go-api/internal/observability"
)

// TestCORSPreflightAllowsConfiguredOrigin verifies a preflight against a
// configured origin returns the allow-origin and Vary headers.
func TestCORSPreflightAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()

	srv := corsTestServer(t, []string{"https://app.example"})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodOptions, srv.URL+"/todos", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "https://app.example", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header.Get("Vary"), "Origin")
}

// TestCORSDisabledByDefault verifies that with no configured origins the server
// emits no CORS headers at all (the safe template default).
func TestCORSDisabledByDefault(t *testing.T) {
	t.Parallel()

	srv := corsTestServer(t, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/healthz", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://app.example")

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

func corsTestServer(t *testing.T, origins []string) *httptest.Server {
	t.Helper()

	discard := observability.NewLogger(io.Discard, slog.LevelError, "json")
	handler := NewRouter(RouterDeps{
		Logger:             discard,
		Metrics:            observability.NewMetrics(),
		Version:            "test",
		RequestTimeout:     testRequestTimeout,
		CORSAllowedOrigins: origins,
		Readiness:          nil,
		Register:           nil,
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return srv
}
