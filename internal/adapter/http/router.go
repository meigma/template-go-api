package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/meigma/template-go-api/internal/observability"
)

// RouterDeps carries the dependencies needed to assemble the HTTP handler.
type RouterDeps struct {
	// Logger is the base logger for the recover and access-log middleware.
	Logger *slog.Logger
	// Metrics provides the metrics middleware and the /metrics handler.
	Metrics *observability.Metrics
	// Version is reported in the OpenAPI document.
	Version string
	// RequestTimeout bounds per-request processing in the timeout middleware.
	RequestTimeout time.Duration
	// Readiness lists checks evaluated by /readyz; empty means always ready.
	Readiness []ReadinessCheck
	// Register mounts resource operations onto the Huma API.
	Register Registrar
}

// NewRouter assembles the chi router: the core middleware stack, RFC 9457 error
// fallbacks, the Huma API with its registered resource operations (which appear
// in the OpenAPI spec), and the raw infrastructure routes (/healthz, /readyz,
// /metrics) that bypass the spec.
func NewRouter(deps RouterDeps) http.Handler {
	mux := chi.NewMux()

	// Core middleware, outermost first. Deferred seams (insert here in later
	// slices): authn/authz, CORS, client-IP (RealIP), and rate limiting.
	mux.Use(middleware.RequestID)
	mux.Use(Recoverer(deps.Logger))
	mux.Use(observability.RequestLogger(deps.Logger))
	mux.Use(deps.Metrics.Middleware())
	mux.Use(timeout(deps.RequestTimeout))

	// Error fallbacks: emit RFC 9457 problem+json instead of chi's text/plain 404
	// and empty 405, so every API error response shares Huma's error shape.
	mux.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeProblem(w, http.StatusNotFound, "the requested resource was not found")
	})
	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		// chi does not pass the allowed methods to a custom handler, so rebuild
		// the Allow header (required on a 405 by RFC 9110) by probing the routes.
		if allow := allowedMethods(mux, r.URL.Path); allow != "" {
			w.Header().Set("Allow", allow)
		}
		writeProblem(w, http.StatusMethodNotAllowed, "the method is not allowed for this resource")
	})

	// Resource operations are mounted by their adapter packages via the Registrar.
	api := NewAPI(mux, deps.Version)
	if deps.Register != nil {
		deps.Register(api)
	}

	// Infrastructure routes stay raw chi and are excluded from the spec.
	mountInfra(mux, deps.Metrics, deps.Readiness)

	return mux
}

func mountInfra(mux chi.Router, metrics *observability.Metrics, readiness []ReadinessCheck) {
	mux.Get("/healthz", handleHealthz)
	mux.Get("/readyz", handleReadyz(readiness))
	mux.Handle("/metrics", metrics.Handler())
}

// allowedMethods returns a comma-separated Allow header value for path by probing
// which standard methods the router has registered for it.
func allowedMethods(routes chi.Routes, path string) string {
	probe := []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions,
	}

	allowed := make([]string, 0, len(probe))
	for _, method := range probe {
		if routes.Match(chi.NewRouteContext(), method, path) {
			allowed = append(allowed, method)
		}
	}

	return strings.Join(allowed, ", ")
}
