package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/meigma/template-go-api/internal/adapter/http/middleware"
	"github.com/meigma/template-go-api/internal/adapter/http/problem"
	"github.com/meigma/template-go-api/internal/observability"
)

// RouterDeps carries the dependencies needed to assemble the HTTP handler.
type RouterDeps struct {
	// Logger is the base logger for the recover and access-log middleware.
	Logger *slog.Logger
	// Metrics provides the metrics middleware and, when ServeMetricsEndpoint is
	// set, the /metrics handler.
	Metrics *observability.Metrics
	// ServeMetricsEndpoint mounts /metrics on this router. Leave it false when a
	// dedicated metrics listener serves /metrics instead; the metrics middleware
	// runs either way, so API requests are always recorded.
	ServeMetricsEndpoint bool
	// Version is reported in the OpenAPI document.
	Version string
	// RequestTimeout bounds per-request processing in the timeout middleware.
	RequestTimeout time.Duration
	// CORSAllowedOrigins lists origins for the CORS middleware; empty disables it.
	CORSAllowedOrigins []string
	// TrustedProxyHeader names the proxy header to read the client IP from; empty
	// trusts only the direct TCP peer.
	TrustedProxyHeader string
	// Readiness lists checks evaluated by /readyz; empty means always ready.
	Readiness []ReadinessCheck
	// Register mounts resource operations onto the Huma API.
	Register Registrar
}

// NewRouter assembles the chi router: the core middleware stack, RFC 9457 error
// fallbacks, the Huma API with its registered resource operations (which appear
// in the OpenAPI spec), and the raw infrastructure routes (/healthz, /readyz,
// and — when ServeMetricsEndpoint is set — /metrics) that bypass the spec.
func NewRouter(deps RouterDeps) http.Handler {
	mux := chi.NewMux()

	// Core middleware, outermost first. Deferred seams (insert here in later
	// slices): authn/authz and rate limiting.
	//
	// Client-IP runs first so the request id, access log, and metrics all see
	// the resolved IP. CORS sits after the access log (so preflight responses are
	// logged and metered) and is installed only when origins are configured.
	mux.Use(middleware.ClientIP(deps.TrustedProxyHeader))
	mux.Use(chimiddleware.RequestID)
	mux.Use(middleware.Recoverer(deps.Logger))
	mux.Use(observability.RequestLogger(deps.Logger))
	if len(deps.CORSAllowedOrigins) > 0 {
		mux.Use(middleware.CORS(deps.CORSAllowedOrigins))
	}
	mux.Use(deps.Metrics.Middleware())
	mux.Use(middleware.Timeout(deps.RequestTimeout))

	// Error fallbacks: emit RFC 9457 problem+json instead of chi's text/plain 404
	// and empty 405, so every API error response shares Huma's error shape.
	mux.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		problem.Write(w, http.StatusNotFound, "the requested resource was not found")
	})
	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		// chi does not pass the allowed methods to a custom handler, so rebuild
		// the Allow header (required on a 405 by RFC 9110) by probing the routes.
		if allow := allowedMethods(mux, r.URL.Path); allow != "" {
			w.Header().Set("Allow", allow)
		}
		problem.Write(w, http.StatusMethodNotAllowed, "the method is not allowed for this resource")
	})

	// Resource operations are mounted by their adapter packages via the Registrar.
	api := NewAPI(mux, deps.Version)
	if deps.Register != nil {
		deps.Register(api)
	}

	// Infrastructure routes stay raw chi and are excluded from the spec.
	mountInfra(mux, deps.Metrics, deps.Readiness, deps.ServeMetricsEndpoint)

	return mux
}

func mountInfra(
	mux chi.Router,
	metrics *observability.Metrics,
	readiness []ReadinessCheck,
	serveMetrics bool,
) {
	mux.Get("/healthz", handleHealthz)
	mux.Get("/readyz", handleReadyz(readiness))
	if serveMetrics {
		mux.Handle("/metrics", metrics.Handler())
	}
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
