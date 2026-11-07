package zone

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/dunamismax/lorehallow/server/internal/mapdata"
	"github.com/dunamismax/lorehallow/server/internal/pb"
	"github.com/dunamismax/lorehallow/server/internal/sim"
	"github.com/dunamismax/lorehallow/server/internal/wire"
)

const tick = 50 * time.Millisecond

// fakeOut is an Outbox that decodes and keeps everything sent to it, up
// to limit messages (0 = unlimited), after which Send reports full.
type fakeOut struct {
	mu     sync.Mutex
	limit  int
	msgs   []*pb.Envelope
	full   int
	kicked pb.DisconnectReason
}

func (f *fakeOut) Send(o wire.Outgoing) bool {
	defer o.Release()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.limit > 0 && len(f.msgs) >= f.limit {
		f.full++
		return false
	}
	var env pb.Envelope
	if err := wire.DecodeServer(o.AppendTo(nil), &env); err != nil {
		panic(err)
	}
	f.msgs = append(f.msgs, &env)
	return true
}

func (f *fakeOut) Kick(r pb.DisconnectReason, _ string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kicked = r
}

func (f *fakeOut) all() []*pb.Envelope {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*pb.Envelope(nil), f.msgs...)
}

func (f *fakeOut) lastSnapshot() *pb.Snapshot {
	msgs := f.all()
	for i := len(msgs) - 1; i >= 0; i-- {
		if s := msgs[i].GetSnapshot(); s != nil {
			return s
		}
	}
	return nil
}

func testMap(t *testing.T) *mapdata.Map {
	t.Helper()
	w, h := 10, 10
	solid := make([]bool, w*h)
	for x := range w {
		solid[x], solid[(h-1)*w+x] = true, true
	}
	for y := range h {
		solid[y*w], solid[y*w+w-1] = true, true
	}
	spawns := []mapdata.Point{{X: 2*512 + 256, Y: 2*512 + 256}, {X: 6*512 + 256, Y: 6*512 + 256}}
	m, err := mapdata.New("test", w, h, solid, spawns)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type player struct {
	id     uint32
	out    *fakeOut
	inputs chan sim.Input
}

func join(t *testing.T, z *Zone, id uint32, name string) *player {
	t.Helper()
	p := &player{id: id, out: &fakeOut{}, inputs: make(chan sim.Input, 16)}
	if _, err := z.Join(t.Context(), JoinRequest{ID: id, Name: name, Out: p.out, Inputs: p.inputs}); err != nil {
		t.Fatal(err)
	}
	return p
}

// run starts z in the bubble and returns a function that stops it and
// waits for it to finish.
func run(z *Zone) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { z.Run(ctx); close(done) }()
	return func() { cancel(); <-done }
}

func TestJoinSendsHelloThenEveryone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t), ServerVersion: "test"})
		stop := run(z)
		defer stop()

		a := join(t, z, 1, "Ash")
		b := join(t, z, 2, "Birch")
		synctest.Wait()

		am, bm := a.out.all(), b.out.all()
		if h := am[0].GetServerHello(); h == nil || h.GetPlayerId() != 1 || h.GetMapId() != "test" || h.GetTickRate() != 20 {
			t.Fatalf("first message to a: %v", am[0])
		}
		if h := bm[0].GetServerHello(); h == nil || h.GetPlayerId() != 2 {
			t.Fatalf("first message to b: %v", bm[0])
		}
		if j := bm[1].GetEntityJoined(); j == nil || j.GetDisplayName() != "Ash" || j.GetState().GetId() != 1 {
			t.Fatalf("b should hear about a: %v", bm[1])
		}
		if j := am[1].GetEntityJoined(); j == nil || j.GetDisplayName() != "Birch" {
			t.Fatalf("a should hear about b: %v", am[1])
		}
		// Different spawn points, round-robin.
		if am[0].GetServerHello().GetSpawn().GetX() == bm[0].GetServerHello().GetSpawn().GetX() {
			t.Fatal("both players spawned at the same point")
		}
	})
}

func TestSnapshotsEveryTickWithAcks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		defer stop()
		a := join(t, z, 1, "Ash")
		b := join(t, z, 2, "Birch")
		start := a.out.all()[0].GetServerHello().GetSpawn()

		for seq := uint32(1); seq <= 5; seq++ {
			a.inputs <- sim.Input{Seq: seq, DX: 1}
			time.Sleep(tick)
		}
		synctest.Wait()
		s := a.out.lastSnapshot()
		if s == nil || len(s.GetEntities()) != 2 || s.GetAckInputSeq() != 5 {
			t.Fatalf("a's last snapshot: %v", s)
		}
		var me *pb.EntityState
		for _, e := range s.GetEntities() {
			if e.GetId() == 1 {
				me = e
			}
		}
		if me.GetX() != start.GetX()+5*sim.SpeedStraight || me.GetAnim() != pb.Anim_ANIM_WALK || me.GetFacing() != pb.Facing_FACING_RIGHT {
			t.Fatalf("a after 5 inputs: %v (start x %d)", me, start.GetX())
		}
		// b sees a's movement, and b's own ack is 0.
		if bs := b.out.lastSnapshot(); bs.GetAckInputSeq() != 0 || bs.GetTick() != s.GetTick() {
			t.Fatalf("b's snapshot: %v", bs)
		}
		snaps := 0
		for _, m := range a.out.all() {
			if m.GetSnapshot() != nil {
				snaps++
			}
		}
		if snaps != 5 {
			t.Fatalf("%d snapshots in 5 ticks, want 5", snaps)
		}
	})
}

func TestLeaveTellsEveryoneAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		defer stop()
		a := join(t, z, 1, "Ash")
		_ = join(t, z, 2, "Birch")
		z.Leave(2)
		synctest.Wait() // no tick needed: leave is handled on arrival
		msgs := a.out.all()
		if l := msgs[len(msgs)-1].GetEntityLeft(); l == nil || l.GetId() != 2 {
			t.Fatalf("a's last message: %v", msgs[len(msgs)-1])
		}
		time.Sleep(tick)
		synctest.Wait()
		if s := a.out.lastSnapshot(); len(s.GetEntities()) != 1 {
			t.Fatalf("snapshot after leave has %d entities", len(s.GetEntities()))
		}
		z.Leave(2) // twice, and for someone never there: harmless
		z.Leave(99)
	})
}

func TestFullOutboxNeverStallsTheTick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		defer stop()
		slow := &player{id: 1, out: &fakeOut{limit: 3}, inputs: make(chan sim.Input, 16)}
		if _, err := z.Join(t.Context(), JoinRequest{ID: 1, Name: "Slow", Out: slow.out, Inputs: slow.inputs}); err != nil {
			t.Fatal(err)
		}
		fast := join(t, z, 2, "Fast")
		time.Sleep(20 * tick)
		synctest.Wait()
		snaps := 0
		for _, m := range fast.out.all() {
			if m.GetSnapshot() != nil {
				snaps++
			}
		}
		if snaps != 20 {
			t.Fatalf("healthy player got %d snapshots in 20 ticks, want 20", snaps)
		}
		if slow.out.full == 0 || z.m.sendFull.Value() == 0 {
			t.Fatal("slow player's full queue wasn't counted")
		}
	})
}

func TestBodiesAreReleased(t *testing.T) {
	// Every snapshot body reference is released whether it was queued
	// or refused; Release panics if a body is released one time too many.
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		defer stop()
		_ = join(t, z, 1, "A")
		s := &player{id: 2, out: &fakeOut{limit: 1}, inputs: make(chan sim.Input, 16)}
		_, _ = z.Join(t.Context(), JoinRequest{ID: 2, Name: "B", Out: s.out, Inputs: s.inputs})
		time.Sleep(10 * tick)
		synctest.Wait()
	})
}

func TestFloodingAndInvalidInputGetKicked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t), MaxStrikes: 10})
		stop := run(z)
		defer stop()
		bad := join(t, z, 1, "Cheat")
		good := join(t, z, 2, "Fair")
		for seq := uint32(1); seq <= 12; seq++ {
			bad.inputs <- sim.Input{Seq: seq, DX: 9} // out of range
		}
		good.inputs <- sim.Input{Seq: 1, DX: 1}
		time.Sleep(tick)
		synctest.Wait()
		if bad.out.kicked != pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE {
			t.Fatalf("cheater not kicked: %v", bad.out.kicked)
		}
		if good.out.kicked != pb.DisconnectReason_DISCONNECT_REASON_UNSPECIFIED {
			t.Fatal("fair player kicked")
		}
		if s := good.out.lastSnapshot(); s.GetAckInputSeq() != 1 {
			t.Fatalf("fair player's input not applied: %v", s)
		}
	})
}

func TestZoneFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t), MaxPlayers: 1})
		stop := run(z)
		defer stop()
		_ = join(t, z, 1, "A")
		_, err := z.Join(t.Context(), JoinRequest{ID: 2, Name: "B", Out: &fakeOut{}, Inputs: make(chan sim.Input, 1)})
		if !errors.Is(err, ErrFull) {
			t.Fatalf("err = %v, want ErrFull", err)
		}
	})
}

func TestStoppedZone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		stop()
		if _, err := z.Join(t.Context(), JoinRequest{ID: 1, Out: &fakeOut{}}); !errors.Is(err, ErrStopped) {
			t.Fatalf("join after stop: %v", err)
		}
		z.Leave(1) // must not block
	})
}

func TestTickMetrics(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		_ = join(t, z, 1, "A")
		time.Sleep(10 * tick)
		synctest.Wait()
		stop()
		if n := z.m.tick.Count(); n != 10 {
			t.Fatalf("%d ticks observed, want 10", n)
		}
		if z.m.players.Value() != 1 || z.m.snapshotSz.Value() == 0 {
			t.Fatal("players or snapshot size gauge not set")
		}
	})
}

// nullOut accepts everything and keeps nothing.
type nullOut struct{}

func (nullOut) Send(o wire.Outgoing) bool        { o.Release(); return true }
func (nullOut) Kick(pb.DisconnectReason, string) {}
func benchZone(b *testing.B, n int) (*Zone, []chan sim.Input) {
	w, h := 64, 64
	spawns := make([]mapdata.Point, 0, 64)
	for i := range 64 {
		spawns = append(spawns, mapdata.Point{X: int32(1+i%60)*512 + 256, Y: int32(1+i/8)*512 + 256})
	}
	m, err := mapdata.New("bench", w, h, make([]bool, w*h), spawns)
	if err != nil {
		b.Fatal(err)
	}
	z := New(Config{Map: m, MaxPlayers: n})
	chans := make([]chan sim.Input, n)
	for i := range n {
		chans[i] = make(chan sim.Input, 16)
		if _, err := z.join(&JoinRequest{ID: uint32(i + 1), Name: "bot", Out: nullOut{}, Inputs: chans[i]}); err != nil {
			b.Fatal(err)
		}
	}
	return z, chans
}

// BenchmarkTick100 and BenchmarkTick1000 measure one whole tick (drain
// inputs, step, encode once, fan out) with everyone moving, without
// sockets: the zone's own cost per tick.
func BenchmarkTick100(b *testing.B)  { benchTick(b, 100) }
func BenchmarkTick1000(b *testing.B) { benchTick(b, 1000) }

func benchTick(b *testing.B, n int) {
	z, chans := benchZone(b, n)
	seq := uint32(0)
	b.ReportAllocs()
	for b.Loop() {
		seq++
		for i, c := range chans {
			c <- sim.Input{Seq: seq, DX: int32(i%3) - 1, DY: int32((int(seq)/40+i)%3) - 1}
		}
		z.tick()
	}
}
