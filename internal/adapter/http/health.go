package http

import (
	"context"
	"encoding/json"
	"net/http"
)

// The infrastructure endpoints (/healthz, /readyz, /metrics) are intentionally
// excluded from the RFC 9457 problem+json error contract that the API routes
// follow: liveness/readiness return plain application/json status objects and
// /metrics returns the Prometheus exposition format, since these are consumed by
// probes and scrapers rather than API clients.

// statusKey is the JSON field name used by the health and readiness responses.
const statusKey = "status"

// ReadinessCheck reports whether a dependency required to serve traffic is
// available: it returns nil when ready and a non-nil error otherwise. Checks
// receive the request context so probes (for example, a database ping) can honor
// cancellation and deadlines.
type ReadinessCheck func(ctx context.Context) error

// handleHealthz reports liveness: the process is up and serving.
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{statusKey: "ok"})
}

// handleReadyz reports readiness by evaluating the supplied checks. With no
// checks (the in-memory slice), the server is always ready.
func handleReadyz(checks []ReadinessCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, check := range checks {
			if err := check(r.Context()); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{statusKey: "unavailable"})

				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]string{statusKey: "ready"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
