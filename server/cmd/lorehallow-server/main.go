// Command lorehallow-server runs the Lorehallow game server.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dunamismax/lorehallow/server/internal/buildinfo"
	"github.com/dunamismax/lorehallow/server/internal/config"
	"github.com/dunamismax/lorehallow/server/internal/httpapi"
	"github.com/dunamismax/lorehallow/server/internal/logging"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := run(ctx, os.Args[1:], os.Getenv, os.Stderr, nil)
	switch {
	case errors.Is(err, config.ErrHelp):
		os.Exit(0)
	case err != nil:
		fmt.Fprintln(os.Stderr, "lorehallow-server:", err)
		os.Exit(1)
	}
}

// run starts the server and blocks until ctx is cancelled, then shuts down
// gracefully. onListen, if non-nil, is called with the bound address once
// the listener is open (tests use it to avoid sleeping).
func run(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer, onListen func(net.Addr)) error {
	cfg, err := config.Load(args, getenv, stderr)
	if err != nil {
		return err
	}
	log := logging.New(stderr, cfg.LogFormat, cfg.LogLevel)
	info := buildinfo.Get()

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Addr, err)
	}

	srv := &http.Server{
		Handler:           httpapi.NewHandler(log),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	log.Info("server starting",
		"addr", ln.Addr().String(),
		"version", info.Version,
		"commit", info.Commit,
		"protocol_version", info.ProtocolVersion)
	if onListen != nil {
		onListen(ln.Addr())
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	log.Info("shutting down", "timeout", cfg.ShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	log.Info("server stopped")
	return nil
}
