package session

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"

	"github.com/dunamismax/lorehallow/server/internal/mapdata"
	"github.com/dunamismax/lorehallow/server/internal/pb"
	"github.com/dunamismax/lorehallow/server/internal/sim"
	"github.com/dunamismax/lorehallow/server/internal/wire"
	"github.com/dunamismax/lorehallow/server/internal/zone"
)

// fakeConn stands in for a WebSocket. The test is the client: it puts
// messages on in and reads what the server wrote from out.
type fakeConn struct {
	in     chan clientMsg
	closed chan struct{}

	mu        sync.Mutex
	out       [][]byte
	stalled   bool // Write blocks until the connection closes
	limit     int64
	closeCode websocket.StatusCode
	once      sync.Once
}

type clientMsg struct {
	typ websocket.MessageType
	b   []byte
}

func newFakeConn() *fakeConn {
	return &fakeConn{in: make(chan clientMsg, 4096), closed: make(chan struct{})}
}

func (c *fakeConn) Reader(context.Context) (websocket.MessageType, io.Reader, error) {
	select {
	case m := <-c.in:
		c.mu.Lock()
		limit := c.limit
		c.mu.Unlock()
		if int64(len(m.b)) > limit {
			_ = c.CloseNow()
			return 0, nil, websocket.ErrMessageTooBig
		}
		return m.typ, bytes.NewReader(m.b), nil
	case <-c.closed:
		return 0, nil, net.ErrClosed
	}
}

func (c *fakeConn) Write(ctx context.Context, _ websocket.MessageType, p []byte) error {
	c.mu.Lock()
	stalled := c.stalled
	c.mu.Unlock()
	if stalled {
		select {
		case <-c.closed:
			return net.ErrClosed
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	select {
	case <-c.closed:
		return net.ErrClosed
	default:
	}
	c.mu.Lock()
	c.out = append(c.out, bytes.Clone(p))
	c.mu.Unlock()
	return nil
}

func (c *fakeConn) Close(code websocket.StatusCode, _ string) error {
	c.mu.Lock()
	c.closeCode = code
	c.mu.Unlock()
	return c.CloseNow()
}

func (c *fakeConn) CloseNow() error {
	c.once.Do(func() { close(c.closed) })
	return nil
}

func (c *fakeConn) SetReadLimit(n int64) {
	c.mu.Lock()
	c.limit = n
	c.mu.Unlock()
}

func (c *fakeConn) send(t *testing.T, env *pb.Envelope) {
	t.Helper()
	b, err := wire.Encode(nil, env)
	if err != nil {
		t.Fatal(err)
	}
	c.in <- clientMsg{websocket.MessageBinary, b}
}

func (c *fakeConn) received(t *testing.T) []*pb.Envelope {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*pb.Envelope, 0, len(c.out))
	for _, b := range c.out {
		var env pb.Envelope
		if err := wire.DecodeServer(b, &env); err != nil {
			t.Fatal(err)
		}
		out = append(out, &env)
	}
	return out
}

func (c *fakeConn) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

func (c *fakeConn) disconnect(t *testing.T) *pb.Disconnect {
	t.Helper()
	msgs := c.received(t)
	for i := len(msgs) - 1; i >= 0; i-- {
		if d := msgs[i].GetDisconnect(); d != nil {
			return d
		}
	}
	return nil
}

func count[T any](msgs []*pb.Envelope, get func(*pb.Envelope) *T) int {
	n := 0
	for _, m := range msgs {
		if get(m) != nil {
			n++
		}
	}
	return n
}

// heard reports whether msgs introduced entity id.
func heard(msgs []*pb.Envelope, id uint32) bool {
	for _, m := range msgs {
		for _, e := range m.GetEntitiesJoined().GetEntities() {
			if e.GetState().GetId() == id {
				return true
			}
		}
	}
	return false
}

func hello(name string) *pb.Envelope {
	return &pb.Envelope{Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{ProtocolVersion: pb.ProtocolVersion, ClientVersion: "test", DisplayName: name}}}
}

func input(seq uint32, dx, dy int32) *pb.Envelope {
	return &pb.Envelope{Payload: &pb.Envelope_Input{Input: &pb.PlayerInput{Seq: seq, MoveX: dx, MoveY: dy}}}
}

// world is a running zone plus everything a session needs.
type world struct {
	z      *zone.Zone
	stop   func()
	mu     sync.Mutex
	next   uint32
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	m      *Metrics
}

func openMap(t *testing.T) *mapdata.Map {
	t.Helper()
	w, h := 40, 10
	m, err := mapdata.New("t", w, h, make([]bool, w*h), []mapdata.Point{{X: 2*512 + 256, Y: 5*512 + 256}, {X: 30*512 + 256, Y: 5*512 + 256}})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func newWorld(t *testing.T, cfg Config) *world {
	t.Helper()
	z := zone.New(zone.Config{Map: openMap(t)})
	zctx, zcancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { z.Run(zctx); close(done) }()
	w := &world{z: z, cfg: cfg, m: NewMetrics(nil)}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	w.stop = func() {
		w.cancel()
		w.wg.Wait()
		zcancel()
		<-done
	}
	return w
}

// connect starts a session on a fake connection and returns the
// connection and a channel with Serve's result.
func (w *world) connect() (*fakeConn, <-chan pb.DisconnectReason) {
	c := newFakeConn()
	res := make(chan pb.DisconnectReason, 1)
	w.wg.Go(func() {
		res <- Serve(w.ctx, c, w.cfg, Deps{World: w.z, Metrics: w.m, NextID: func() uint32 {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.next++
			return w.next
		}})
	})
	return c, res
}

func (w *world) join(t *testing.T, name string) (*fakeConn, <-chan pb.DisconnectReason) {
	t.Helper()
	c, res := w.connect()
	c.send(t, hello(name))
	synctest.Wait()
	if msgs := c.received(t); len(msgs) == 0 || msgs[0].GetServerHello() == nil {
		t.Fatalf("%s: no ServerHello, got %v", name, msgs)
	}
	time.Sleep(50 * time.Millisecond) // the next tick announces the arrival
	synctest.Wait()
	return c, res
}

func TestHandshakeAndSnapshots(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, Config{})
		defer w.stop()
		a, _ := w.join(t, "Ash")
		b, _ := w.join(t, "Birch")
		for seq := uint32(1); seq <= 10; seq++ {
			a.send(t, input(seq, 1, 0))
			time.Sleep(50 * time.Millisecond)
		}
		synctest.Wait()
		am, bm := a.received(t), b.received(t)
		if !heard(am, 2) || !heard(bm, 1) {
			t.Fatal("players didn't hear about each other")
		}
		var last *pb.Snapshot
		for _, m := range am {
			if s := m.GetSnapshot(); s != nil {
				last = s
			}
		}
		// Only a moved, so a snapshot carries only a (ADR 0008).
		if last == nil || last.GetAckInputSeq() != 10 || len(last.GetEntities()) != 1 || last.GetEntities()[0].GetId() != 1 {
			t.Fatalf("last snapshot for a: %v", last)
		}
		if n := count(bm, (*pb.Envelope).GetSnapshot); n < 9 || n > 11 {
			t.Fatalf("b got %d snapshots in 10 ticks", n)
		}
	})
}

func TestHandshakeRejections(t *testing.T) {
	cases := []struct {
		name string
		send func(t *testing.T, c *fakeConn)
		want pb.DisconnectReason
		code websocket.StatusCode
	}{
		{"protocol mismatch", func(t *testing.T, c *fakeConn) {
			c.send(t, &pb.Envelope{Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{ProtocolVersion: pb.ProtocolVersion + 1, DisplayName: "Ash"}}})
		}, pb.DisconnectReason_DISCONNECT_REASON_PROTOCOL_MISMATCH, 4001},
		{"bad name", func(t *testing.T, c *fakeConn) { c.send(t, hello(" spaced")) }, pb.DisconnectReason_DISCONNECT_REASON_INVALID_HELLO, 0},
		{"input before hello", func(t *testing.T, c *fakeConn) { c.send(t, input(1, 1, 0)) }, pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, 0},
		{"malformed bytes", func(_ *testing.T, c *fakeConn) { c.in <- clientMsg{websocket.MessageBinary, []byte{0xff, 0xff}} }, pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, 0},
		{"text message", func(_ *testing.T, c *fakeConn) { c.in <- clientMsg{websocket.MessageText, []byte("hi")} }, pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, 0},
		{"oversized", func(_ *testing.T, c *fakeConn) {
			c.in <- clientMsg{websocket.MessageBinary, make([]byte, wire.MaxClientMessage+1)}
		}, pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, 0},
		{"no hello at all", func(*testing.T, *fakeConn) {}, pb.DisconnectReason_DISCONNECT_REASON_TIMED_OUT, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := newWorld(t, Config{})
				defer w.stop()
				c, res := w.connect()
				tc.send(t, c)
				start := time.Now()
				if got := <-res; got != tc.want {
					t.Fatalf("Serve returned %v, want %v", got, tc.want)
				}
				if tc.want == pb.DisconnectReason_DISCONNECT_REASON_TIMED_OUT && time.Since(start) != DefaultHandshakeTimeout {
					t.Fatalf("handshake timeout after %v", time.Since(start))
				}
				if !c.isClosed() {
					t.Fatal("connection left open")
				}
				if tc.want != pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE || tc.name != "oversized" {
					if d := c.disconnect(t); d == nil || d.GetReason() != tc.want || d.GetText() == "" {
						t.Fatalf("Disconnect = %v, want reason %v with text", d, tc.want)
					}
				}
				if tc.code != 0 && c.closeCode != tc.code {
					t.Fatalf("close code %d, want %d", c.closeCode, tc.code)
				}
			})
		})
	}
}

func TestValidName(t *testing.T) {
	good := []string{"Ash", "Mary-Jane", "O'Brien", "Élodie", "森の人", "A", "Sixteen Letters!"[:15], "R2 D2"}
	bad := []string{"", " Ash", "Ash ", "Two  Spaces", "Seventeen letters", "tab\tname", "new\nline", "emoji🙂", "semi;colon", "\xff"}
	for _, n := range good {
		if _, ok := ValidName(n); !ok {
			t.Errorf("ValidName(%q) = false", n)
		}
	}
	for _, n := range bad {
		if _, ok := ValidName(n); ok {
			t.Errorf("ValidName(%q) = true", n)
		}
	}
}

func TestSilentClientTimesOutAndLeaves(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, Config{})
		defer w.stop()
		quiet, res := w.join(t, "Quiet")
		other, _ := w.join(t, "Other")
		stop := make(chan struct{})
		defer close(stop)
		go func() { // other stays alive
			for {
				select {
				case <-stop:
					return
				case <-time.After(time.Second):
					other.send(t, &pb.Envelope{Payload: &pb.Envelope_Ping{Ping: &pb.Ping{}}})
				}
			}
		}()
		start := time.Now()
		if got := <-res; got != pb.DisconnectReason_DISCONNECT_REASON_TIMED_OUT {
			t.Fatalf("silent client: %v", got)
		}
		if waited := time.Since(start); waited < DefaultIdleTimeout || waited > DefaultIdleTimeout+DefaultPingInterval {
			t.Fatalf("timed out after %v, want %v to %v", waited, DefaultIdleTimeout, DefaultIdleTimeout+DefaultPingInterval)
		}
		time.Sleep(50 * time.Millisecond) // departures are announced at the next tick
		synctest.Wait()
		if count(other.received(t), (*pb.Envelope).GetEntitiesLeft) != 1 {
			t.Fatal("others didn't see the silent player leave")
		}
		if count(quiet.received(t), (*pb.Envelope).GetPing) < 5 {
			t.Fatal("server didn't ping the client")
		}
	})
}

func TestHeartbeatsKeepAClientConnected(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, Config{})
		defer w.stop()
		c, res := w.join(t, "Alive")
		answered := 0
		for range 30 { // 60 s, four times the idle timeout
			time.Sleep(DefaultPingInterval)
			synctest.Wait()
			msgs := c.received(t)
			for _, m := range msgs[answered:] {
				if p := m.GetPing(); p != nil {
					c.send(t, &pb.Envelope{Payload: &pb.Envelope_Pong{Pong: &pb.Pong{Nonce: p.GetNonce(), PingSentAtMs: p.GetSentAtMs()}}})
				}
			}
			answered = len(msgs)
		}
		select {
		case r := <-res:
			t.Fatalf("disconnected: %v", r)
		default:
		}
		if w.m.RTT.Count() < 25 {
			t.Fatalf("only %d RTT samples", w.m.RTT.Count())
		}
	})
}

func TestSlowClientIsEvictedWithoutHurtingOthers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, Config{SendQueueLen: 16})
		defer w.stop()
		slow, res := w.join(t, "Slow")
		fast, _ := w.join(t, "Fast")
		// fast walks back and forth, so every tick has a change to send.
		stopWalking := make(chan struct{})
		walking := make(chan struct{})
		go func() {
			defer close(walking)
			for seq := uint32(1); ; seq++ {
				select {
				case <-stopWalking:
					return
				case <-time.After(50 * time.Millisecond):
					fast.send(t, input(seq, 1-int32(seq%2)*2, 0))
				}
			}
		}()
		defer func() { close(stopWalking); <-walking }()
		slow.mu.Lock()
		slow.stalled = true // stops reading: every write blocks
		slow.mu.Unlock()
		start := time.Now()
		if got := <-res; got != pb.DisconnectReason_DISCONNECT_REASON_TOO_SLOW {
			t.Fatalf("stalled client: %v", got)
		}
		// 16 queued snapshots at 20 a second, plus the one blocked in Write.
		if took := time.Since(start); took > time.Second+500*time.Millisecond {
			t.Fatalf("eviction took %v", took)
		}
		before := count(fast.received(t), (*pb.Envelope).GetSnapshot)
		time.Sleep(time.Second)
		synctest.Wait()
		if got := count(fast.received(t), (*pb.Envelope).GetSnapshot) - before; got != 20 {
			t.Fatalf("healthy client got %d snapshots in 1 s after the eviction, want 20", got)
		}
		if w.m.Evicted.Value() != 1 {
			t.Fatalf("evicted = %d", w.m.Evicted.Value())
		}
	})
}

func TestFloodIsRateLimited(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, Config{})
		defer w.stop()
		c, res := w.join(t, "Flood")
		for i := range 2000 {
			c.send(t, &pb.Envelope{Payload: &pb.Envelope_Ping{Ping: &pb.Ping{Nonce: uint32(i)}}})
		}
		if got := <-res; got != pb.DisconnectReason_DISCONNECT_REASON_RATE_LIMITED {
			t.Fatalf("flooding client: %v", got)
		}
		if w.m.RateDropped.Value() <= DefaultMaxRateDrops {
			t.Fatalf("dropped %d", w.m.RateDropped.Value())
		}
	})
}

func TestInputFloodCannotOutrunTheSimulation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, Config{RatePerSecond: 1e6, RateBurst: 1e6}) // let the flood through to the zone
		defer w.stop()
		cheat, _ := w.join(t, "Cheat")
		start := cheat.received(t)[0].GetServerHello().GetSpawn().GetX()
		seq := uint32(0)
		const ticks = 20
		for range ticks {
			for range 30 {
				seq++
				cheat.send(t, input(seq, 1, 0))
			}
			time.Sleep(50 * time.Millisecond)
		}
		synctest.Wait()
		var x int32
		for _, m := range cheat.received(t) {
			for _, e := range m.GetSnapshot().GetEntities() {
				if e.GetId() == 1 {
					x = e.GetX()
				}
			}
		}
		if moved := x - start; moved > ticks*sim.SpeedStraight {
			t.Fatalf("moved %d in %d ticks, the limit is %d", moved, ticks, ticks*sim.SpeedStraight)
		}
	})
}

func TestShutdownDisconnectsEveryoneCleanly(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, Config{})
		var conns []*fakeConn
		var results []<-chan pb.DisconnectReason
		for _, n := range []string{"A", "B", "C"} {
			c, r := w.join(t, n)
			conns, results = append(conns, c), append(results, r)
		}
		w.stop() // cancels the sessions' context, waits for them, stops the zone
		for i, r := range results {
			if got := <-r; got != pb.DisconnectReason_DISCONNECT_REASON_SHUTTING_DOWN {
				t.Fatalf("session %d: %v", i, got)
			}
			if d := conns[i].disconnect(t); d.GetReason() != pb.DisconnectReason_DISCONNECT_REASON_SHUTTING_DOWN {
				t.Fatalf("session %d got %v", i, d)
			}
			if conns[i].closeCode != CloseCodeBase+5 {
				t.Fatalf("close code %d", conns[i].closeCode)
			}
		}
		if w.m.Open.Value() != 0 {
			t.Fatalf("%d connections still open", w.m.Open.Value())
		}
		// synctest.Test fails if any goroutine from this bubble is still
		// running here: no leaked readers, writers or zone loops.
	})
}

func TestPeerCloseCleansUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, Config{})
		defer w.stop()
		a, res := w.join(t, "A")
		b, _ := w.join(t, "B")
		_ = a.CloseNow() // the client hangs up
		if got := <-res; got != pb.DisconnectReason_DISCONNECT_REASON_UNSPECIFIED {
			t.Fatalf("peer close: %v", got)
		}
		time.Sleep(50 * time.Millisecond)
		synctest.Wait()
		if count(b.received(t), (*pb.Envelope).GetEntitiesLeft) != 1 {
			t.Fatal("other player didn't see the leave")
		}
	})
}

func TestSendQueueLimits(t *testing.T) {
	body := wire.NewBody()
	body.B = append(body.B, make([]byte, 100)...)
	q := newSendQueue(3, 250)
	for i := range 2 {
		if !q.push(wire.Outgoing{Bytes: make([]byte, 100)}) {
			t.Fatalf("push %d under the limits failed", i)
		}
	}
	if q.push(wire.Outgoing{Bytes: make([]byte, 100)}) {
		t.Fatal("byte limit not enforced")
	}
	if !q.push(wire.Outgoing{Bytes: make([]byte, 10)}) {
		t.Fatal("small message under both limits refused")
	}
	body.Retain()
	if q.push(wire.Outgoing{Body: body}) {
		t.Fatal("message limit not enforced")
	}
	if n, b := q.depth(); n != 3 || b != 210 {
		t.Fatalf("depth %d messages %d bytes", n, b)
	}
	got := q.takeAll(nil)
	if len(got) != 3 {
		t.Fatalf("took %d", len(got))
	}
	q.close()
	body.Retain()
	if !q.push(wire.Outgoing{Body: body}) {
		t.Fatal("a closed queue reports full")
	}
	body.Release() // the caller's own reference; the pushes were released by the queue
}

func TestSendQueueGrowsAndKeepsOrder(t *testing.T) {
	q := newSendQueue(100, 1<<20)
	if len(q.ring) != initialQueueSlots {
		t.Fatalf("starts with %d slots", len(q.ring))
	}
	for i := range 5 { // wrap the ring before it grows
		q.push(wire.Outgoing{Bytes: []byte{byte(i)}})
	}
	q.takeAll(nil)
	for i := range 100 {
		if !q.push(wire.Outgoing{Bytes: []byte{byte(i)}}) {
			t.Fatalf("push %d refused under the limit", i)
		}
	}
	if q.push(wire.Outgoing{Bytes: []byte{0}}) {
		t.Fatal("grew past maxMessages")
	}
	got := q.takeAll(nil)
	for i, o := range got {
		if o.Bytes[0] != byte(i) {
			t.Fatalf("message %d is %d: order lost while growing", i, o.Bytes[0])
		}
	}
	if len(q.ring) != initialQueueSlots {
		t.Fatalf("didn't shrink back: %d slots", len(q.ring))
	}
}

func TestEveryReasonHasACounter(t *testing.T) {
	m := NewMetrics(nil)
	for v := range pb.DisconnectReason_name {
		if m.reasons[pb.DisconnectReason(v)] == nil {
			t.Fatalf("no counter for %v", pb.DisconnectReason(v))
		}
	}
}
