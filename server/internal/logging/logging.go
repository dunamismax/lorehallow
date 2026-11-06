// Package logging builds the server's slog logger: human-readable text
// for local development and JSON for production.
package logging

import (
	"io"
	"log/slog"
)

// New returns a logger writing to w in the given format ("text" or "json").
// Any format other than "json" produces text.
func New(w io.Writer, format string, level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}
