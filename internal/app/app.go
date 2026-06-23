// Package app is the composition root: it wires the domain service, the selected
// persistence adapter (in-memory or PostgreSQL), observability, and the HTTP
// server into a runnable App.
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
	"github.com/meigma/template-go-api/internal/todo/memory"
	todopostgres "github.com/meigma/template-go-api/internal/todo/postgres"
)

// App is a fully wired API server ready to Run.
type App struct {
	server        *http.Server
	metricsServer *http.Server
	logger        *slog.Logger
	grace         time.Duration
	// pool is the PostgreSQL connection pool when store=postgres, closed during
	// graceful shutdown. It is nil for the in-memory store.
	pool *pgxpool.Pool
}

// New wires the application from cfg and logger. version is reported in the
// OpenAPI document served by the API. When the postgres store is selected it
// connects a connection pool, which can fail. The caller owns running and
// shutting the App down, which closes the pool.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger, version string) (*App, error) {
	repo, pool, readiness, err := selectStore(ctx, cfg)
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
		// The in-memory store has nothing to probe, so /readyz is always ready;
		// the postgres store contributes a real connectivity check here.
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

// selectStore builds the todo.Repository for the configured store. For postgres
// it also returns the pool (for shutdown) and a readiness check; for memory both
// are nil and readiness is empty.
func selectStore(
	ctx context.Context,
	cfg config.Config,
) (todo.Repository, *pgxpool.Pool, []adapterhttp.ReadinessCheck, error) {
	if cfg.Store != config.StorePostgres {
		return memory.NewTodoRepository(), nil, nil, nil
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
// OpenAPI 3.0.3 specification as YAML.
func OpenAPIYAML(version string) ([]byte, error) {
	service := todo.NewService(memory.NewTodoRepository(), nil)

	spec, err := adapterhttp.SpecYAML(version, registerResources(service))
	if err != nil {
		return nil, fmt.Errorf("build openapi spec: %w", err)
	}

	return spec, nil
}

// registerResources composes the per-resource HTTP adapters mounted on the API.
// Add a new resource by constructing its service above and adding one Register
// call here.
func registerResources(todoService *todo.Service) adapterhttp.Registrar {
	return func(api huma.API) {
		httpapi.Register(api, todoService)
	}
}
