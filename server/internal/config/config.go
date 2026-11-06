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
}

// DefaultAddr is the documented local-development listen address.
const DefaultAddr = "127.0.0.1:7420"

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
	var level, timeout string
	fs.StringVar(&cfg.Addr, "addr", env("ADDR", DefaultAddr), "listen address (env LOREHALLOW_ADDR)")
	fs.StringVar(&cfg.LogFormat, "log-format", env("LOG_FORMAT", "text"), "log format: text or json (env LOREHALLOW_LOG_FORMAT)")
	fs.StringVar(&level, "log-level", env("LOG_LEVEL", "info"), "log level: debug, info, warn, error (env LOREHALLOW_LOG_LEVEL)")
	fs.StringVar(&timeout, "shutdown-timeout", env("SHUTDOWN_TIMEOUT", "10s"), "graceful shutdown timeout (env LOREHALLOW_SHUTDOWN_TIMEOUT)")

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
	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}
	return cfg, nil
}
