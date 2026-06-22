package http

import (
	"log/slog"
	"net/http"
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
	Readiness []func() error
	// Register mounts resource operations onto the Huma API.
	Register Registrar
}

// NewRouter assembles the chi router: the core middleware stack, the Huma API
// with its registered resource operations (which appear in the OpenAPI spec),
// and the raw infrastructure routes (/healthz, /readyz, /metrics) that bypass it.
func NewRouter(deps RouterDeps) http.Handler {
	mux := chi.NewMux()

	// Core middleware, outermost first. Deferred seams (insert here in later
	// slices): authn/authz, CORS, client-IP (RealIP), and rate limiting.
	mux.Use(middleware.RequestID)
	mux.Use(observability.Recoverer(deps.Logger))
	mux.Use(observability.RequestLogger(deps.Logger))
	mux.Use(deps.Metrics.Middleware())
	mux.Use(middleware.Timeout(deps.RequestTimeout))

	// Resource operations are mounted by their adapter packages via the Registrar.
	api := NewAPI(mux, deps.Version)
	if deps.Register != nil {
		deps.Register(api)
	}

	// Infrastructure routes stay raw chi and are excluded from the spec.
	mountInfra(mux, deps.Metrics, deps.Readiness)

	return mux
}

func mountInfra(mux chi.Router, metrics *observability.Metrics, readiness []func() error) {
	mux.Get("/healthz", handleHealthz)
	mux.Get("/readyz", handleReadyz(readiness))
	mux.Handle("/metrics", metrics.Handler())
}
