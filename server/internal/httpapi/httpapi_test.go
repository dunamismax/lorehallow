package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dunamismax/lorehallow/server/internal/buildinfo"
)

func newTestHandler() http.Handler {
	return NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestHealthz(t *testing.T) {
	tests := []struct {
		method   string
		wantCode int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodHead, http.StatusOK},
		{http.MethodPost, http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newTestHandler().ServeHTTP(rec, httptest.NewRequest(tt.method, "/healthz", nil))
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.method == http.MethodGet && rec.Body.String() != "ok\n" {
				t.Errorf("body = %q, want %q", rec.Body.String(), "ok\n")
			}
		})
	}
}

func TestVersion(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var got buildinfo.Info
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got != buildinfo.Get() {
		t.Errorf("version = %+v, want %+v", got, buildinfo.Get())
	}
	if got.ProtocolVersion < 1 {
		t.Errorf("protocol version = %d, want >= 1", got.ProtocolVersion)
	}
}

func TestUnknownPath(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
