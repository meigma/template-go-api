package observability

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// contextKey is a private type for context keys defined in this package.
type contextKey int

const loggerContextKey contextKey = iota

// LoggerFrom returns the request-scoped logger stored in ctx by RequestLogger.
// The boolean reports whether a logger was present.
func LoggerFrom(ctx context.Context) (*slog.Logger, bool) {
	logger, ok := ctx.Value(loggerContextKey).(*slog.Logger)

	return logger, ok
}

// RequestLogger returns middleware that derives a request-scoped child logger
// carrying the chi request id, stores it in the request context, and logs one
// structured line per request after the handler returns.
func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			logger := base.With(slog.String("request_id", middleware.GetReqID(r.Context())))
			ctx := context.WithValue(r.Context(), loggerContextKey, logger)
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(wrapped, r.WithContext(ctx))

			logger.LogAttrs(ctx, slog.LevelInfo, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.Status()),
				slog.Int("bytes", wrapped.BytesWritten()),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

// Recoverer returns middleware that converts a panic into a 500 response and
// logs it through the request-scoped logger, falling back to base. It re-panics
// on [http.ErrAbortHandler] so the server can abort the connection as intended.
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
				if scoped, ok := LoggerFrom(r.Context()); ok {
					logger = scoped
				}
				logger.ErrorContext(r.Context(), "recovered from panic", slog.Any("panic", recovered))
				w.WriteHeader(http.StatusInternalServerError)
			}()

			next.ServeHTTP(w, r)
		})
	}
}
