package gameserver

import (
	"net/http"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name   string
		trust  bool
		remote string
		xff    string
		want   string
	}{
		{"direct", false, "203.0.113.5:4000", "", "203.0.113.5"},
		{"header ignored without trust", false, "127.0.0.1:4000", "198.51.100.7", "127.0.0.1"},
		{"behind Caddy", true, "127.0.0.1:4000", "198.51.100.7", "198.51.100.7"},
		{"last hop wins: a client can't forge its way out of the limit", true, "127.0.0.1:4000", "10.9.9.9, 198.51.100.7", "198.51.100.7"},
		{"header from a non-loopback peer is ignored", true, "203.0.113.5:4000", "198.51.100.7", "203.0.113.5"},
		{"garbage header", true, "[::1]:4000", "not-an-ip", "::1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := New(Config{TrustProxy: c.trust})
			r := &http.Request{RemoteAddr: c.remote, Header: http.Header{}}
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := s.clientIP(r); got != c.want {
				t.Fatalf("clientIP = %q, want %q", got, c.want)
			}
		})
	}
}

func TestPendingCountsPerIP(t *testing.T) {
	s := New(Config{MaxPendingPerIP: 2})
	for i, want := range []bool{true, true, false} {
		if got := s.addPending("a"); got != want {
			t.Fatalf("addPending(a) #%d = %v, want %v", i+1, got, want)
		}
	}
	if !s.addPending("b") {
		t.Fatal("another IP is limited by the first one")
	}
	s.removePending("a")
	if !s.addPending("a") {
		t.Fatal("slot not freed")
	}
	s.removePending("a")
	s.removePending("a")
	s.removePending("b")
	if len(s.pending) != 0 {
		t.Fatalf("pending map not cleaned up: %v", s.pending)
	}
}
