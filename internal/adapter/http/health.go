package http

import (
	"encoding/json"
	"net/http"
)

// statusKey is the JSON field name used by the health and readiness responses.
const statusKey = "status"

// handleHealthz reports liveness: the process is up and serving.
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{statusKey: "ok"})
}

// handleReadyz reports readiness by evaluating the supplied checks. With no
// checks (the in-memory slice), the server is always ready.
func handleReadyz(checks []func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		for _, check := range checks {
			if err := check(); err != nil {
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
