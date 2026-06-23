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
