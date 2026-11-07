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
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/dunamismax/lorehallow/server/internal/buildinfo"
	"github.com/dunamismax/lorehallow/server/internal/config"
	"github.com/dunamismax/lorehallow/server/internal/gameserver"
	"github.com/dunamismax/lorehallow/server/internal/httpapi"
	"github.com/dunamismax/lorehallow/server/internal/logging"
	"github.com/dunamismax/lorehallow/server/internal/mapdata"
	"github.com/dunamismax/lorehallow/server/internal/metrics"
	"github.com/dunamismax/lorehallow/server/internal/procstat"
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

// Listening reports the bound addresses (tests use it to avoid sleeping).
type Listening struct {
	Game, Admin net.Addr
}

// run starts the server and blocks until ctx is cancelled, then shuts down
// gracefully: stop accepting, disconnect every player with SHUTTING_DOWN,
// stop the zone, stop the HTTP servers. onListen, if non-nil, is called
// once the listeners are open.
func run(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer, onListen func(Listening)) error {
	cfg, err := config.Load(args, getenv, stderr)
	if err != nil {
		return err
	}
	log := logging.New(stderr, cfg.LogFormat, cfg.LogLevel)
	info := buildinfo.Get()

	m, err := mapdata.Load(cfg.MapPath)
	if err != nil {
		return fmt.Errorf("load map: %w", err)
	}
	reg := metrics.NewRegistry()
	registerRuntimeMetrics(reg)
	gs := gameserver.New(gameserver.Config{
		Map:             m,
		MaxConnections:  cfg.MaxConnections,
		MaxPendingPerIP: cfg.MaxPendingPerIP,
		OriginPatterns:  cfg.AllowedOrigins,
		TrustProxy:      cfg.TrustProxy,
		Version:         info.Version,
		Log:             log,
		Metrics:         reg,
	})

	mux := http.NewServeMux()
	mux.Handle("/", httpapi.NewHandler(log))
	mux.Handle("GET /ws", gs.Handler())

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Addr, err)
	}
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	var admin *http.Server
	var adminLn net.Listener
	if cfg.AdminAddr != "" {
		adminLn, err = net.Listen("tcp", cfg.AdminAddr)
		if err != nil {
			_ = ln.Close()
			return fmt.Errorf("listen on admin %s: %w", cfg.AdminAddr, err)
		}
		admin = &http.Server{Handler: adminHandler(reg), ReadHeaderTimeout: 5 * time.Second}
	}

	log.Info("server starting",
		"addr", ln.Addr().String(),
		"admin_addr", cfg.AdminAddr,
		"map", m.ID,
		"version", info.Version,
		"commit", info.Commit,
		"protocol_version", info.ProtocolVersion,
		"max_connections", cfg.MaxConnections)
	if onListen != nil {
		l := Listening{Game: ln.Addr()}
		if adminLn != nil {
			l.Admin = adminLn.Addr()
		}
		onListen(l)
	}

	gameDone := make(chan struct{})
	go func() { gs.Run(ctx); close(gameDone) }()
	serveErr := make(chan error, 2)
	go func() { serveErr <- srv.Serve(ln) }()
	if admin != nil {
		go func() { serveErr <- admin.Serve(adminLn) }()
	}

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	log.Info("shutting down", "timeout", cfg.ShutdownTimeout, "online", gs.Online())
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	// Stop accepting first. Shutdown doesn't wait for WebSockets (they are
	// hijacked); the game server drains those itself.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	select {
	case <-gameDone:
	case <-shutdownCtx.Done():
		return errors.New("graceful shutdown: game server did not stop in time")
	}
	if admin != nil {
		_ = admin.Shutdown(shutdownCtx)
	}
	log.Info("server stopped")
	return nil
}

// adminHandler serves the loopback-only admin endpoints.
func adminHandler(reg *metrics.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		reg.WriteText(w)
	})
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

// registerRuntimeMetrics exports the Go runtime numbers load tests need.
func registerRuntimeMetrics(reg *metrics.Registry) {
	reg.NewGaugeFunc("lorehallow_goroutines", "Goroutines in the process.", func() float64 { return float64(runtime.NumGoroutine()) })
	reg.NewGaugeFunc("lorehallow_process_cpu_seconds_total", "User and system CPU time used by the process.", func() float64 { return procstat.SelfCPU().Seconds() })
	// ReadMemStats stops the world briefly; read it at most once a second
	// rather than once per gauge per scrape.
	var mu sync.Mutex
	var ms runtime.MemStats
	var at time.Time
	stats := func(f func(*runtime.MemStats) float64) func() float64 {
		return func() float64 {
			mu.Lock()
			defer mu.Unlock()
			if time.Since(at) > time.Second {
				runtime.ReadMemStats(&ms)
				at = time.Now()
			}
			return f(&ms)
		}
	}
	reg.NewGaugeFunc("lorehallow_heap_inuse_bytes", "Heap bytes in use.", stats(func(m *runtime.MemStats) float64 { return float64(m.HeapInuse) }))
	reg.NewGaugeFunc("lorehallow_stack_inuse_bytes", "Goroutine stack bytes in use.", stats(func(m *runtime.MemStats) float64 { return float64(m.StackInuse) }))
	reg.NewGaugeFunc("lorehallow_sys_bytes", "Bytes obtained from the OS by the Go runtime.", stats(func(m *runtime.MemStats) float64 { return float64(m.Sys) }))
	reg.NewGaugeFunc("lorehallow_gc_cpu_fraction", "Fraction of CPU time used by the GC since start.", stats(func(m *runtime.MemStats) float64 { return m.GCCPUFraction }))
	reg.NewGaugeFunc("lorehallow_gc_cycles", "Completed GC cycles.", stats(func(m *runtime.MemStats) float64 { return float64(m.NumGC) }))
	reg.NewGaugeFunc("lorehallow_gc_pause_total_seconds", "Total GC stop-the-world pause time.", stats(func(m *runtime.MemStats) float64 { return float64(m.PauseTotalNs) / 1e9 }))
	reg.NewGaugeFunc("lorehallow_mallocs_total", "Cumulative heap allocations.", stats(func(m *runtime.MemStats) float64 { return float64(m.Mallocs) }))
}
