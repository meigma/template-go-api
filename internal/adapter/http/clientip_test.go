package http

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/template-go-api/internal/observability"
)

// TestClientIPResolution verifies the trust model: by default a forged proxy
// header is ignored and the access log records the TCP peer, but a configured
// trusted header is honored. The handler is driven synchronously so the shared
// log buffer is written and read on one goroutine.
func TestClientIPResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		trustedHeader string
		wantClientIP  string
	}{
		{name: "ignores forged header by default", trustedHeader: "", wantClientIP: "192.0.2.1"},
		{name: "honors trusted header", trustedHeader: "X-Real-IP", wantClientIP: "203.0.113.7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := observability.NewLogger(&buf, slog.LevelInfo, "json")
			handler := NewRouter(RouterDeps{
				Logger:             logger,
				Metrics:            observability.NewMetrics(),
				Version:            "test",
				RequestTimeout:     testRequestTimeout,
				TrustedProxyHeader: tt.trustedHeader,
				Readiness:          nil,
				Register:           nil,
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil) // RemoteAddr 192.0.2.1:1234
			req.Header.Set("X-Real-IP", "203.0.113.7")
			handler.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.wantClientIP, findAccessLog(t, buf.String())["client_ip"])
		})
	}
}

// findAccessLog returns the parsed "http request" access-log line from logs.
func findAccessLog(t *testing.T, logs string) map[string]any {
	t.Helper()

	for line := range strings.SplitSeq(strings.TrimSpace(logs), "\n") {
		if line == "" {
			continue
		}

		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry["msg"] == "http request" {
			return entry
		}
	}

	t.Fatalf("no access-log line in:\n%s", logs)

	return nil
}
