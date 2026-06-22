// Package app is the composition root: it wires the domain service, the
// in-memory adapter, observability, and the HTTP server into a runnable App.
package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	adapterhttp "github.com/meigma/template-go-api/internal/adapter/http"
	"github.com/meigma/template-go-api/internal/adapter/http/todoapi"
	"github.com/meigma/template-go-api/internal/adapter/memory"
	"github.com/meigma/template-go-api/internal/config"
	"github.com/meigma/template-go-api/internal/observability"
	"github.com/meigma/template-go-api/internal/todo"
)

// App is a fully wired API server ready to Run.
type App struct {
	server *http.Server
	logger *slog.Logger
	grace  time.Duration
}

// New wires the application from cfg and logger. version is reported in the
// OpenAPI document served by the API.
func New(cfg config.Config, logger *slog.Logger, version string) *App {
	service := todo.NewService(memory.NewTodoRepository(), logger)
	metrics := observability.NewMetrics()
	handler := adapterhttp.NewRouter(adapterhttp.RouterDeps{
		Logger:             logger,
		Metrics:            metrics,
		Version:            version,
		RequestTimeout:     cfg.RequestTimeout,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		TrustedProxyHeader: cfg.TrustedProxyHeader,
		Readiness:          nil,
		Register:           registerResources(service),
	})

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	return &App{
		server: server,
		logger: logger,
		grace:  cfg.ShutdownGrace,
	}
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
		todoapi.Register(api, todoService)
	}
}
