package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// syncBuffer is a goroutine-safe log sink.
type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestRunServesAndShutsDown(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	addrCh := make(chan net.Addr, 1)
	done := make(chan error, 1)
	logs := &syncBuffer{}
	go func() {
		done <- run(ctx, []string{"-addr", "127.0.0.1:0"}, func(string) string { return "" }, logs, func(a net.Addr) { addrCh <- a })
	}()

	var addr net.Addr
	select {
	case addr = <-addrCh:
	case err := <-done:
		t.Fatalf("run exited early: %v", err)
	}

	resp, err := http.Get("http://" + addr.String() + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "ok\n" {
		t.Fatalf("healthz = %d %q", resp.StatusCode, body)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run returned %v after cancel", err)
	}
	if out := logs.String(); !strings.Contains(out, "server stopped") {
		t.Errorf("missing shutdown log:\n%s", out)
	}
}

func TestRunBadConfig(t *testing.T) {
	err := run(t.Context(), []string{"-log-format", "xml"}, func(string) string { return "" }, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "log-format") {
		t.Fatalf("run() error = %v, want log-format error", err)
	}
}

func TestRunAddrInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	err = run(t.Context(), []string{"-addr", ln.Addr().String()}, func(string) string { return "" }, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "listen") {
		t.Fatalf("run() error = %v, want listen error", err)
	}
}
