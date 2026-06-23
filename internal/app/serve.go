package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// namedServer pairs an [http.Server] with a label used in log lines.
type namedServer struct {
	server *http.Server
	name   string
}

// servers returns the servers to run: the API server and, when a metrics-addr is
// configured, the dedicated metrics server.
func (a *App) servers() []namedServer {
	servers := []namedServer{{server: a.server, name: "http server"}}
	if a.metricsServer != nil {
		servers = append(servers, namedServer{server: a.metricsServer, name: "metrics server"})
	}

	return servers
}

// Run starts the configured HTTP servers and blocks until ctx is cancelled or a
// server fails, then shuts every server down within the configured grace period.
func (a *App) Run(ctx context.Context) error {
	// Close the database pool (when postgres) on every exit path, after the
	// servers have returned — including when a server fails to start.
	defer a.closePool(ctx)

	servers := a.servers()
	serveErr := make(chan error, len(servers))
	for _, s := range servers {
		go func() {
			a.logger.InfoContext(ctx, s.name+" listening", slog.String("addr", s.server.Addr))

			err := s.server.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				serveErr <- fmt.Errorf("%s: %w", s.name, err)

				return
			}

			serveErr <- nil
		}()
	}

	select {
	case err := <-serveErr:
		if err != nil {
			return err
		}

		return nil
	case <-ctx.Done():
		return a.shutdown(ctx)
	}
}

func (a *App) shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.grace)
	defer cancel()

	var errs []error
	for _, s := range a.servers() {
		a.logger.InfoContext(ctx, "shutting down "+s.name)
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.name, err))
		}
	}

	a.logger.InfoContext(shutdownCtx, "servers stopped")

	return errors.Join(errs...)
}

// closePool closes the PostgreSQL pool when one is configured. It is deferred in
// Run so it executes on every exit path — graceful shutdown or a server failing
// to start — after the servers have returned, so no in-flight handler loses its
// connection mid-request. It is a no-op when no pool was opened (for example,
// a repository injected with WithRepository in tests).
func (a *App) closePool(ctx context.Context) {
	if a.pool == nil {
		return
	}

	a.logger.InfoContext(ctx, "closing database pool")
	a.pool.Close()
}
