package main

// End-to-end tests: the real server (run) and real bots over loopback
// WebSockets. Real sockets mean real time, so these tests wait for
// conditions with deadlines (eventually); everything that can be tested
// with fake time is tested in the packages, under testing/synctest.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/dunamismax/lorehallow/server/internal/bot"
	"github.com/dunamismax/lorehallow/server/internal/mapdata"
	"github.com/dunamismax/lorehallow/server/internal/pb"
	"github.com/dunamismax/lorehallow/server/internal/sim"
)

const testMap = "../../../content/maps/starter.tmj"

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

type server struct {
	game, admin string
	cancel      context.CancelFunc
	done        chan error
	logs        *syncBuffer
}

func start(t *testing.T, args ...string) *server {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	s := &server{cancel: cancel, done: make(chan error, 1), logs: &syncBuffer{}}
	ready := make(chan Listening, 1)
	args = append([]string{"-addr", "127.0.0.1:0", "-admin-addr", "127.0.0.1:0", "-map", testMap}, args...)
	go func() {
		s.done <- run(ctx, args, func(string) string { return "" }, s.logs, func(l Listening) { ready <- l })
	}()
	select {
	case l := <-ready:
		s.game, s.admin = l.Game.String(), l.Admin.String()
	case err := <-s.done:
		t.Fatalf("run exited early: %v\n%s", err, s.logs.String())
	}
	t.Cleanup(func() { s.stop(t) })
	return s
}

func (s *server) stop(t *testing.T) {
	s.cancel()
	if err := <-s.done; err != nil {
		t.Errorf("run returned %v", err)
	}
	s.done <- nil // stop is idempotent
}

func (s *server) url() string { return "ws://" + s.game + "/ws" }

func dial(t *testing.T, s *server, name string) *bot.Bot {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	b, err := bot.Dial(ctx, bot.Config{URL: s.url(), Name: name})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	t.Cleanup(b.CloseNow)
	return b
}

// eventually polls cond until it holds or the deadline passes.
func eventually(t *testing.T, within time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(within)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %v waiting for %s", within, what)
		}
		<-tick.C
	}
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestRunServesAndShutsDown(t *testing.T) {
	s := start(t)
	if code, body := get(t, "http://"+s.game+"/healthz"); code != http.StatusOK || body != "ok\n" {
		t.Fatalf("healthz = %d %q", code, body)
	}
	s.stop(t)
	if out := s.logs.String(); !strings.Contains(out, "server stopped") {
		t.Errorf("missing shutdown log:\n%s", out)
	}
}

func TestRunBadConfig(t *testing.T) {
	err := run(t.Context(), []string{"-log-format", "xml"}, func(string) string { return "" }, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "log-format") {
		t.Fatalf("run() error = %v, want log-format error", err)
	}
}

func TestRunBadMap(t *testing.T) {
	err := run(t.Context(), []string{"-map", "nope.tmj"}, func(string) string { return "" }, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "load map") {
		t.Fatalf("run() error = %v, want a map error", err)
	}
}

func TestRunAddrInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	err = run(t.Context(), []string{"-addr", ln.Addr().String(), "-map", testMap}, func(string) string { return "" }, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "listen") {
		t.Fatalf("run() error = %v, want listen error", err)
	}
}

// TestTwoPlayersSeeEachOtherMoveAndLeave is BUILD.md 2.5's integration
// test: each bot sees the other move, and sees it leave within one second
// of a disconnect.
func TestTwoPlayersSeeEachOtherMoveAndLeave(t *testing.T) {
	s := start(t)
	a := dial(t, s, "Ash")
	b := dial(t, s, "Birch")
	eventually(t, 2*time.Second, "both bots to see two entities", func() bool { return a.Visible() == 2 && b.Visible() == 2 })

	before, _ := b.Known(a.ID)
	for range 10 {
		if err := a.Move(t.Context(), 0, 1); err != nil {
			t.Fatal(err)
		}
	}
	eventually(t, 2*time.Second, "b to see a move", func() bool {
		e, _ := b.Known(a.ID)
		return e.Y != before.Y
	})
	eventually(t, 2*time.Second, "a to see b standing", func() bool {
		e, ok := a.Known(b.ID)
		return ok && e.Anim == pb.Anim_ANIM_IDLE
	})

	gone := time.Now()
	a.CloseNow() // a crash: no close handshake
	eventually(t, time.Second, "b to see a leave", func() bool { _, ok := b.LeftAt(a.ID); return ok })
	at, _ := b.LeftAt(a.ID)
	if took := at.Sub(gone); took > time.Second {
		t.Fatalf("leave took %v, want under 1 s", took)
	}
}

// TestCheatersCannotCheat runs every cheat mode against the real server
// next to a fair player (BUILD.md 2.5).
func TestCheatersCannotCheat(t *testing.T) {
	s := start(t)
	fair := dial(t, s, "Fair")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	type result struct {
		b      *bot.Bot
		cheat  bot.Cheat
		ticks0 uint32
	}
	var cheaters []result
	for _, c := range []bot.Cheat{bot.Flood, bot.ForgeSeq, bot.Oversize, bot.Unknown, bot.BadValues} {
		b, err := bot.Dial(ctx, bot.Config{URL: s.url(), Name: fmt.Sprintf("Cheat %d", c), Cheat: c, Seed: uint64(c)})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(b.CloseNow)
		cheaters = append(cheaters, result{b: b, cheat: c})
	}
	for i := range cheaters {
		eventually(t, 2*time.Second, "cheater's first snapshot", func() bool { return cheaters[i].b.LastTick() > 0 })
		cheaters[i].ticks0 = cheaters[i].b.LastTick()
	}
	// 40 ticks of cheating, two seconds. After every tick, no cheater may
	// stand inside a wall of the real map.
	m, err := mapdata.Load(testMap)
	if err != nil {
		t.Fatal(err)
	}
	walls := sim.NewWorld(m)
	tick := time.NewTicker(50 * time.Millisecond)
	for range 40 {
		<-tick.C
		for _, c := range cheaters {
			_ = c.b.Step(ctx)
			if self, ok := c.b.Self(); ok && walls.Overlaps(self.X, self.Y) {
				t.Fatalf("cheat %d is inside a wall at (%d, %d)", c.cheat, self.X, self.Y)
			}
		}
		_ = fair.Move(ctx, 0, 0)
	}
	tick.Stop()

	for _, c := range cheaters {
		switch c.cheat {
		case bot.Oversize, bot.Unknown, bot.BadValues:
			eventually(t, 3*time.Second, fmt.Sprintf("cheat %d disconnected", c.cheat), func() bool {
				select {
				case <-c.b.Done():
					return true
				default:
					return false
				}
			})
			if d := c.b.Disconnect(); c.cheat != bot.Oversize && (d == nil || d.Reason != pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE) {
				t.Errorf("cheat %d: disconnect %v, want BAD_MESSAGE", c.cheat, d)
			}
		case bot.Flood, bot.ForgeSeq:
			// Still connected or kicked, either way never faster than allowed.
			self, _ := c.b.Self()
			ticks := int32(c.b.LastTick() - c.ticks0)
			moved := self.X - c.b.Spawn.X
			if moved < 0 {
				moved = -moved
			}
			if limit := (ticks + 1) * sim.SpeedStraight; moved > limit {
				t.Errorf("cheat %d moved %d in %d ticks, limit %d", c.cheat, moved, ticks, limit)
			}
		}
	}
	select {
	case <-fair.Done():
		t.Fatalf("the fair player was disconnected: %v %v", fair.Disconnect(), fair.Err())
	default:
	}
	if n, ack := fair.Snapshots(); n < 30 || ack != 40 {
		t.Fatalf("fair player: %d snapshots, ack %d", n, ack)
	}
}

func TestProtocolMismatchIsRejected(t *testing.T) {
	s := start(t)
	_, err := bot.Dial(t.Context(), bot.Config{URL: s.url(), Name: "Old", ProtocolVersion: pb.ProtocolVersion + 1})
	var de *bot.DisconnectError
	if !errors.As(err, &de) || de.Reason != pb.DisconnectReason_DISCONNECT_REASON_PROTOCOL_MISMATCH || !strings.Contains(de.Text, "Update required") {
		t.Fatalf("err = %v, want PROTOCOL_MISMATCH with Update required", err)
	}
}

func TestServerFull(t *testing.T) {
	s := start(t, "-max-connections", "1")
	_ = dial(t, s, "First")
	_, err := bot.Dial(t.Context(), bot.Config{URL: s.url(), Name: "Second"})
	var de *bot.DisconnectError
	if !errors.As(err, &de) || de.Reason != pb.DisconnectReason_DISCONNECT_REASON_SERVER_FULL {
		t.Fatalf("err = %v, want SERVER_FULL", err)
	}
}

func TestPendingHandshakesPerIPAreCapped(t *testing.T) {
	s := start(t, "-max-pending-per-ip", "2")
	var silent []*websocket.Conn
	for range 2 { // connect, never say hello
		c, _, err := websocket.Dial(t.Context(), s.url(), nil)
		if err != nil {
			t.Fatal(err)
		}
		silent = append(silent, c)
	}
	_, resp, err := websocket.Dial(t.Context(), s.url(), nil)
	if err == nil || resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("third pending handshake from one IP: err %v, want HTTP 429", err)
	}
	for _, c := range silent {
		_ = c.CloseNow()
	}
	// Once they're gone, a real player gets in.
	eventually(t, 2*time.Second, "pending slots freed", func() bool {
		b, err := bot.Dial(t.Context(), bot.Config{URL: s.url(), Name: "Real"})
		if err != nil {
			return false
		}
		b.CloseNow()
		return true
	})
}

func TestShutdownTellsEveryPlayer(t *testing.T) {
	s := start(t)
	a := dial(t, s, "A")
	b := dial(t, s, "B")
	s.stop(t)
	for _, p := range []*bot.Bot{a, b} {
		<-p.Done()
		if d := p.Disconnect(); d == nil || d.Reason != pb.DisconnectReason_DISCONNECT_REASON_SHUTTING_DOWN {
			t.Fatalf("player %d: %v", p.ID, d)
		}
	}
}

func TestAdminEndpoints(t *testing.T) {
	s := start(t)
	_ = dial(t, s, "Metric")
	eventually(t, 2*time.Second, "a few ticks", func() bool {
		_, body := get(t, "http://"+s.admin+"/metrics")
		return strings.Contains(body, "lorehallow_zone_players 1")
	})
	_, body := get(t, "http://"+s.admin+"/metrics")
	for _, want := range []string{"lorehallow_zone_tick_seconds_bucket", "lorehallow_connections_open 1", "lorehallow_goroutines", "lorehallow_disconnects_too_slow_total"} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics missing %q", want)
		}
	}
	if code, _ := get(t, "http://"+s.admin+"/debug/pprof/"); code != http.StatusOK {
		t.Fatalf("pprof index: %d", code)
	}
	// The admin port is not the game port.
	if code, _ := get(t, "http://"+s.game+"/metrics"); code == http.StatusOK {
		t.Fatal("metrics served on the public port")
	}
}

func TestSwarmOfBots(t *testing.T) {
	// Every bot dials from 127.0.0.1, so lift the per-IP handshake cap the
	// way load tests do.
	s := start(t, "-max-pending-per-ip", "1000")
	ctx, cancel := context.WithCancel(t.Context())
	stats := &bot.Stats{}
	done := make(chan *bot.Swarm, 1)
	go func() {
		done <- bot.RunSwarm(ctx, bot.SwarmConfig{URL: s.url(), N: 50, Behavior: bot.Wander, Stats: stats}, func(*bot.Swarm) {})
	}()
	eventually(t, 10*time.Second, "50 bots joined and receiving", func() bool { return stats.Joined.Load() == 50 && stats.Snapshots.Load() > 50*20 })
	cancel()
	sw := <-done
	if len(sw.Errs) > 0 {
		t.Fatalf("dial errors: %v", sw.Errs[:min(3, len(sw.Errs))])
	}
	eventually(t, 5*time.Second, "server to see everyone leave", func() bool {
		_, body := get(t, "http://"+s.admin+"/metrics")
		return strings.Contains(body, "lorehallow_zone_players 0")
	})
}
