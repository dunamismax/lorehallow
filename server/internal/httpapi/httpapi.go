// Package httpapi serves the server's plain HTTP endpoints: health and
// version now, status and the WebSocket upgrade in later phases.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dunamismax/lorehallow/server/internal/buildinfo"
)

// NewHandler returns the root HTTP handler.
func NewHandler(log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, log, buildinfo.Get())
	})
	return mux
}

func writeJSON(w http.ResponseWriter, log *slog.Logger, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Warn("write json response", "err", err)
	}
}
