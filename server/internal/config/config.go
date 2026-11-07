// Package config loads server configuration from command-line flags and
// LOREHALLOW_* environment variables. Flags win over the environment, and
// the environment wins over the built-in local-development defaults.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"
)

// Config is the complete server configuration.
type Config struct {
	// Addr is the host:port the HTTP and WebSocket listener binds to.
	Addr string
	// LogFormat is "text" (local) or "json" (production).
	LogFormat string
	// LogLevel is debug, info, warn or error.
	LogLevel slog.Level
	// ShutdownTimeout bounds how long graceful shutdown may take.
	ShutdownTimeout time.Duration
	// MapPath is the zone's Tiled map.
	MapPath string
	// AdminAddr serves /metrics and /debug/pprof on loopback only; empty disables it.
	AdminAddr string
	// MaxConnections caps open game connections.
	MaxConnections int
	// MaxPendingPerIP caps unfinished handshakes from one IP.
	MaxPendingPerIP int
	// AllowedOrigins lists extra browser origins allowed to open /ws.
	AllowedOrigins []string
	// TrustProxy takes client IPs from X-Forwarded-For on loopback connections.
	TrustProxy bool
}

// Documented local-development defaults.
const (
	DefaultAddr      = "127.0.0.1:7420"
	DefaultAdminAddr = "127.0.0.1:7421"
	DefaultMapPath   = "content/maps/starter.tmj"
)

// EnvPrefix is prepended to every environment variable name.
const EnvPrefix = "LOREHALLOW_"

// ErrHelp is returned when -h or -help was requested; usage has been printed.
var ErrHelp = flag.ErrHelp

// Load parses args (without the program name) and reads the environment
// through getenv. Usage and errors are written to stderr.
func Load(args []string, getenv func(string) string, stderr io.Writer) (Config, error) {
	fs := flag.NewFlagSet("lorehallow-server", flag.ContinueOnError)
	fs.SetOutput(stderr)

	env := func(name, def string) string {
		if v := getenv(EnvPrefix + name); v != "" {
			return v
		}
		return def
	}

	var cfg Config
	var level, timeout, maxConns, maxPending, origins, trustProxy string
	fs.StringVar(&cfg.Addr, "addr", env("ADDR", DefaultAddr), "listen address (env LOREHALLOW_ADDR)")
	fs.StringVar(&cfg.LogFormat, "log-format", env("LOG_FORMAT", "text"), "log format: text or json (env LOREHALLOW_LOG_FORMAT)")
	fs.StringVar(&level, "log-level", env("LOG_LEVEL", "info"), "log level: debug, info, warn, error (env LOREHALLOW_LOG_LEVEL)")
	fs.StringVar(&timeout, "shutdown-timeout", env("SHUTDOWN_TIMEOUT", "20s"), "graceful shutdown timeout (env LOREHALLOW_SHUTDOWN_TIMEOUT)")
	fs.StringVar(&cfg.MapPath, "map", env("MAP", DefaultMapPath), "zone map, a Tiled .tmj file (env LOREHALLOW_MAP)")
	fs.StringVar(&cfg.AdminAddr, "admin-addr", env("ADMIN_ADDR", DefaultAdminAddr), "loopback address for /metrics and /debug/pprof; empty disables (env LOREHALLOW_ADMIN_ADDR)")
	fs.StringVar(&maxConns, "max-connections", env("MAX_CONNECTIONS", "5000"), "maximum open game connections (env LOREHALLOW_MAX_CONNECTIONS)")
	fs.StringVar(&maxPending, "max-pending-per-ip", env("MAX_PENDING_PER_IP", "16"), "maximum unfinished handshakes per client IP (env LOREHALLOW_MAX_PENDING_PER_IP)")
	fs.StringVar(&origins, "allowed-origins", env("ALLOWED_ORIGINS", ""), "comma-separated extra browser origins allowed on /ws (env LOREHALLOW_ALLOWED_ORIGINS)")
	fs.StringVar(&trustProxy, "trust-proxy", env("TRUST_PROXY", "false"), "take client IPs from X-Forwarded-For on loopback connections (env LOREHALLOW_TRUST_PROXY)")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	var errs []error
	if cfg.Addr == "" {
		errs = append(errs, errors.New("addr must not be empty"))
	}
	switch cfg.LogFormat {
	case "text", "json":
	default:
		errs = append(errs, fmt.Errorf("log-format must be text or json, got %q", cfg.LogFormat))
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(level)); err != nil {
		errs = append(errs, fmt.Errorf("log-level: %w", err))
	}
	d, err := time.ParseDuration(timeout)
	switch {
	case err != nil:
		errs = append(errs, fmt.Errorf("shutdown-timeout: %w", err))
	case d <= 0:
		errs = append(errs, fmt.Errorf("shutdown-timeout must be positive, got %s", d))
	default:
		cfg.ShutdownTimeout = d
	}
	if cfg.MapPath == "" {
		errs = append(errs, errors.New("map must not be empty"))
	}
	if cfg.AdminAddr != "" {
		host, _, err := net.SplitHostPort(cfg.AdminAddr)
		ip := net.ParseIP(host)
		if err != nil || (host != "localhost" && (ip == nil || !ip.IsLoopback())) {
			errs = append(errs, fmt.Errorf("admin-addr must be a loopback address (metrics and pprof are never public), got %q", cfg.AdminAddr))
		}
	}
	positive := func(name, v string, dst *int) {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			errs = append(errs, fmt.Errorf("%s must be a positive integer, got %q", name, v))
			return
		}
		*dst = n
	}
	positive("max-connections", maxConns, &cfg.MaxConnections)
	positive("max-pending-per-ip", maxPending, &cfg.MaxPendingPerIP)
	for _, o := range strings.Split(origins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
		}
	}
	if b, err := strconv.ParseBool(trustProxy); err != nil {
		errs = append(errs, fmt.Errorf("trust-proxy must be true or false, got %q", trustProxy))
	} else {
		cfg.TrustProxy = b
	}
	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}
	return cfg, nil
}
