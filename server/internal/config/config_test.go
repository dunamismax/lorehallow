package config

import (
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"
)

// base is the default configuration with the given changes.
func base(edit func(c *Config)) Config {
	c := Config{
		Addr: DefaultAddr, LogFormat: "text", LogLevel: slog.LevelInfo, ShutdownTimeout: 20 * time.Second,
		MapPath: DefaultMapPath, AdminAddr: DefaultAdminAddr, MaxConnections: 5000, MaxPendingPerIP: 16,
	}
	if edit != nil {
		edit(&c)
	}
	return c
}

func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "defaults",
			want: base(nil),
		},
		{
			name: "environment overrides defaults",
			env: map[string]string{
				"LOREHALLOW_ADDR":             ":9000",
				"LOREHALLOW_LOG_FORMAT":       "json",
				"LOREHALLOW_LOG_LEVEL":        "debug",
				"LOREHALLOW_SHUTDOWN_TIMEOUT": "3s",
			},
			want: base(func(c *Config) {
				c.Addr, c.LogFormat, c.LogLevel, c.ShutdownTimeout = ":9000", "json", slog.LevelDebug, 3*time.Second
			}),
		},
		{
			name: "flags override environment",
			args: []string{"-addr", "0.0.0.0:1234", "-log-level", "warn"},
			env:  map[string]string{"LOREHALLOW_ADDR": ":9000", "LOREHALLOW_LOG_LEVEL": "debug"},
			want: base(func(c *Config) { c.Addr, c.LogLevel = "0.0.0.0:1234", slog.LevelWarn }),
		},
		{
			name: "game server settings",
			args: []string{"-map", "x.tmj", "-max-connections", "100", "-allowed-origins", "lorehallow.com, www.lorehallow.com", "-trust-proxy", "true", "-admin-addr", ""},
			env:  map[string]string{"LOREHALLOW_MAX_PENDING_PER_IP": "4"},
			want: base(func(c *Config) {
				c.MapPath, c.MaxConnections, c.MaxPendingPerIP, c.TrustProxy, c.AdminAddr = "x.tmj", 100, 4, true, ""
				c.AllowedOrigins = []string{"lorehallow.com", "www.lorehallow.com"}
			}),
		},
		{name: "public admin address", args: []string{"-admin-addr", "0.0.0.0:7421"}, wantErr: "loopback"},
		{name: "admin on localhost is fine", args: []string{"-admin-addr", "localhost:9"}, want: base(func(c *Config) { c.AdminAddr = "localhost:9" })},
		{name: "zero connections", args: []string{"-max-connections", "0"}, wantErr: "max-connections"},
		{name: "bad trust-proxy", args: []string{"-trust-proxy", "maybe"}, wantErr: "trust-proxy"},
		{name: "empty map", args: []string{"-map", ""}, wantErr: "map must not be empty"},
		{name: "bad log format", args: []string{"-log-format", "xml"}, wantErr: "log-format"},
		{name: "bad log level", env: map[string]string{"LOREHALLOW_LOG_LEVEL": "loud"}, wantErr: "log-level"},
		{name: "bad timeout", args: []string{"-shutdown-timeout", "soon"}, wantErr: "shutdown-timeout"},
		{name: "negative timeout", args: []string{"-shutdown-timeout", "-1s"}, wantErr: "must be positive"},
		{name: "empty addr", args: []string{"-addr", ""}, wantErr: "addr must not be empty"},
		{name: "unknown flag", args: []string{"-nope"}, wantErr: "flag provided but not defined"},
		{name: "stray argument", args: []string{"extra"}, wantErr: "unexpected arguments"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.args, envMap(tt.env), io.Discard)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Load() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadHelp(t *testing.T) {
	var out strings.Builder
	_, err := Load([]string{"-h"}, envMap(nil), &out)
	if !errors.Is(err, ErrHelp) {
		t.Fatalf("Load(-h) error = %v, want ErrHelp", err)
	}
	if !strings.Contains(out.String(), "LOREHALLOW_ADDR") {
		t.Errorf("usage does not mention environment variables:\n%s", out.String())
	}
}
