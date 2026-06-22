package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// Run starts the HTTP server and blocks until ctx is cancelled, then shuts the
// server down within the configured grace period.
func (a *App) Run(ctx context.Context) error {
	serveErr := make(chan error, 1)
	go func() {
		a.logger.InfoContext(ctx, "http server listening", slog.String("addr", a.server.Addr))

		err := a.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err

			return
		}

		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}

		return nil
	case <-ctx.Done():
		return a.shutdown(ctx)
	}
}

func (a *App) shutdown(ctx context.Context) error {
	a.logger.InfoContext(ctx, "shutting down http server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.grace)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	a.logger.InfoContext(shutdownCtx, "http server stopped")

	return nil
}
