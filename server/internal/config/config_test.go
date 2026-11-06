package config

import (
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

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
			want: Config{Addr: DefaultAddr, LogFormat: "text", LogLevel: slog.LevelInfo, ShutdownTimeout: 10 * time.Second},
		},
		{
			name: "environment overrides defaults",
			env: map[string]string{
				"LOREHALLOW_ADDR":             ":9000",
				"LOREHALLOW_LOG_FORMAT":       "json",
				"LOREHALLOW_LOG_LEVEL":        "debug",
				"LOREHALLOW_SHUTDOWN_TIMEOUT": "3s",
			},
			want: Config{Addr: ":9000", LogFormat: "json", LogLevel: slog.LevelDebug, ShutdownTimeout: 3 * time.Second},
		},
		{
			name: "flags override environment",
			args: []string{"-addr", "0.0.0.0:1234", "-log-level", "warn"},
			env:  map[string]string{"LOREHALLOW_ADDR": ":9000", "LOREHALLOW_LOG_LEVEL": "debug"},
			want: Config{Addr: "0.0.0.0:1234", LogFormat: "text", LogLevel: slog.LevelWarn, ShutdownTimeout: 10 * time.Second},
		},
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
			if got != tt.want {
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
