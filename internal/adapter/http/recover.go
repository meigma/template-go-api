package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/meigma/template-go-api/internal/observability"
)

// Recoverer returns middleware that converts a panic into an RFC 9457 500
// response and logs it through the request-scoped logger, falling back to base.
// It re-panics on [http.ErrAbortHandler] so the server can abort the connection
// as intended. It lives in the transport layer (rather than observability) so it
// can share writeProblem with the router's other non-Huma error surfaces.
func Recoverer(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				if err, ok := recovered.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(recovered)
				}

				logger := base
				if scoped, ok := observability.LoggerFrom(r.Context()); ok {
					logger = scoped
				}
				logger.ErrorContext(r.Context(), "recovered from panic", slog.Any("panic", recovered))
				writeProblem(w, http.StatusInternalServerError, "internal server error")
			}()

			next.ServeHTTP(w, r)
		})
	}
}
