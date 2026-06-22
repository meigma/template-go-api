package http

import (
	"net/http"

	"github.com/go-chi/cors"
)

// corsMaxAgeSeconds is how long browsers may cache a CORS preflight response.
const corsMaxAgeSeconds = 300

// corsMiddleware returns CORS middleware restricted to origins. It is installed
// only when origins is non-empty, so the server emits no CORS headers by
// default. Methods, headers, and credentials use conservative defaults suited to
// a JSON API; widen them per deployment as needed.
func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: origins,
		AllowedMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
		},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           corsMaxAgeSeconds,
	})
}
