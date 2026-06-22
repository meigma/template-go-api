package http

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/meigma/template-go-api/internal/todo"
)

// apiTitle is the OpenAPI document title for this service.
const apiTitle = "template-go-api"

// NewAPI registers the todo operations on mux and returns the Huma API. The
// returned API serves requests through mux and can export the OpenAPI spec.
func NewAPI(mux chi.Router, service *todo.Service, version string) huma.API {
	api := humachi.New(mux, huma.DefaultConfig(apiTitle, version))
	handlers := &todoHandlers{service: service}
	handlers.register(api)

	return api
}

// SpecYAML builds the API on a throwaway router and returns the OpenAPI 3.0.3
// specification as YAML, without binding a network listener.
func SpecYAML(service *todo.Service, version string) ([]byte, error) {
	api := NewAPI(chi.NewMux(), service, version)

	spec, err := api.OpenAPI().DowngradeYAML()
	if err != nil {
		return nil, fmt.Errorf("downgrade openapi spec to yaml: %w", err)
	}

	return spec, nil
}
