// Package app is the composition root: it wires the domain service, the
// PostgreSQL persistence adapter, observability, and the HTTP server into a
// runnable App.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	adapterhttp "github.com/meigma/template-go-api/internal/adapter/http"
	"github.com/meigma/template-go-api/internal/adapter/postgres"
	"github.com/meigma/template-go-api/internal/config"
	"github.com/meigma/template-go-api/internal/observability"
	"github.com/meigma/template-go-api/internal/todo"
	"github.com/meigma/template-go-api/internal/todo/httpapi"
	todopostgres "github.com/meigma/template-go-api/internal/todo/postgres"
)

// App is a fully wired API server ready to Run.
type App struct {
	server        *http.Server
	metricsServer *http.Server
	logger        *slog.Logger
	grace         time.Duration
	// pool is the PostgreSQL connection pool, closed during graceful shutdown.
	// It is nil when a repository is injected with WithRepository (tests).
	pool *pgxpool.Pool
}

// Option configures how New wires the application.
type Option func(*options)

type options struct {
	repo todo.Repository
}

// WithRepository injects a ready-made todo.Repository instead of connecting the
// PostgreSQL adapter. It lets tests wire the full server without a database, and
// gives integrators a seam to plug in an alternative adapter without editing the
// composition root.
func WithRepository(repo todo.Repository) Option {
	return func(o *options) {
		o.repo = repo
	}
}

// New wires the application from cfg and logger. version is reported in the
// OpenAPI document served by the API. Unless a repository is injected with
// WithRepository, it connects a PostgreSQL connection pool, which can fail. The
// caller owns running and shutting the App down, which closes the pool.
func New(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	version string,
	opts ...Option,
) (*App, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	repo, pool, readiness, err := resolveStore(ctx, cfg, o.repo)
	if err != nil {
		return nil, err
	}

	service := todo.NewService(repo, logger)
	metrics := observability.NewMetrics()

	// An empty metrics-addr co-locates /metrics on the API listener; otherwise a
	// dedicated metrics server (below) serves it off the API surface.
	serveMetricsInline := cfg.MetricsAddr == ""
	handler := adapterhttp.NewRouter(adapterhttp.RouterDeps{
		Logger:               logger,
		Metrics:              metrics,
		ServeMetricsEndpoint: serveMetricsInline,
		Version:              version,
		RequestTimeout:       cfg.RequestTimeout,
		CORSAllowedOrigins:   cfg.CORSAllowedOrigins,
		TrustedProxyHeader:   cfg.TrustedProxyHeader,
		// The postgres store contributes a real connectivity check here; an
		// injected repository (tests) contributes none, so /readyz is always ready.
		Readiness: readiness,
		Register:  registerResources(service),
	})

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	var metricsServer *http.Server
	if !serveMetricsInline {
		metricsServer = &http.Server{
			Addr:              cfg.MetricsAddr,
			Handler:           adapterhttp.NewMetricsHandler(metrics),
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		}
	}

	return &App{
		server:        server,
		metricsServer: metricsServer,
		logger:        logger,
		grace:         cfg.ShutdownGrace,
		pool:          pool,
	}, nil
}

// resolveStore returns the todo.Repository to wire. An injected repository is
// used as-is with no pool or readiness check (tests, or an integrator-supplied
// adapter); otherwise it connects the PostgreSQL adapter and returns the pool
// (for shutdown) and a connectivity readiness check.
func resolveStore(
	ctx context.Context,
	cfg config.Config,
	injected todo.Repository,
) (todo.Repository, *pgxpool.Pool, []adapterhttp.ReadinessCheck, error) {
	if injected != nil {
		return injected, nil, nil, nil
	}

	pool, err := postgres.Connect(ctx, postgres.Config{URL: cfg.DatabaseURL, MaxConns: cfg.DBMaxConns})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("connect postgres: %w", err)
	}

	repo := todopostgres.NewTodoRepository(pool)
	readiness := []adapterhttp.ReadinessCheck{{Name: "postgres", Check: repo.Ping}}

	return repo, pool, readiness, nil
}

// Handler returns the assembled HTTP handler, primarily for functional tests.
func (a *App) Handler() http.Handler {
	return a.server.Handler
}

// OpenAPIYAML builds the API without binding a listener and returns the
// OpenAPI 3.0.3 specification as YAML. The repository is never invoked while
// generating the spec, so a no-op stub stands in for the real adapter and no
// database connection is required.
func OpenAPIYAML(version string) ([]byte, error) {
	service := todo.NewService(noopRepository{}, nil)

	spec, err := adapterhttp.SpecYAML(version, registerResources(service))
	if err != nil {
		return nil, fmt.Errorf("build openapi spec: %w", err)
	}

	return spec, nil
}

// noopRepository is a todo.Repository that performs no persistence. It exists
// solely to construct the service when generating the OpenAPI document
// server-lessly, where the repository is never invoked.
type noopRepository struct{}

func (noopRepository) Save(_ context.Context, _ todo.Todo) error { return nil }

func (noopRepository) FindByID(_ context.Context, _ string) (todo.Todo, error) {
	return todo.Todo{}, todo.ErrNotFound
}

func (noopRepository) List(_ context.Context) ([]todo.Todo, error) { return nil, nil }

// registerResources composes the per-resource HTTP adapters mounted on the API.
// Add a new resource by constructing its service above and adding one Register
// call here.
func registerResources(todoService *todo.Service) adapterhttp.Registrar {
	return func(api huma.API) {
		httpapi.Register(api, todoService)
	}
}
