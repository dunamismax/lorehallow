package zone

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
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
		time.Sleep(tick) // arrivals are announced at the start of the next tick
		synctest.Wait()

		am, bm := a.out.all(), b.out.all()
		if h := am[0].GetServerHello(); h == nil || h.GetPlayerId() != 1 || h.GetMapId() != "test" || h.GetTickRate() != 20 {
			t.Fatalf("first message to a: %v", am[0])
		}
		if h := bm[0].GetServerHello(); h == nil || h.GetPlayerId() != 2 {
			t.Fatalf("first message to b: %v", bm[0])
		}
		// Both joined before the tick, so both are newcomers and each gets
		// the whole zone in one message, itself included.
		for name, msgs := range map[string][]*pb.Envelope{"a": am, "b": bm} {
			j := msgs[1].GetEntitiesJoined()
			if j == nil || len(j.GetEntities()) != 2 || j.GetEntities()[0].GetDisplayName() != "Ash" || j.GetEntities()[1].GetDisplayName() != "Birch" {
				t.Fatalf("%s's second message: %v", name, msgs[1])
			}
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
		// Only a moved, so only a is in the snapshot (ADR 0008).
		if s == nil || len(s.GetEntities()) != 1 || s.GetAckInputSeq() != 5 {
			t.Fatalf("a's last snapshot: %v", s)
		}
		me := s.GetEntities()[0]
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
		time.Sleep(tick)
		synctest.Wait() // the tick that announces both has run
		z.Leave(2)
		time.Sleep(tick) // announced at the start of the next tick: within 50 ms
		synctest.Wait()
		msgs := a.out.all()
		var left *pb.EntitiesLeft
		for _, m := range msgs {
			if l := m.GetEntitiesLeft(); l != nil {
				left = l
			}
		}
		if left == nil || len(left.GetIds()) != 1 || left.GetIds()[0] != 2 {
			t.Fatalf("a never heard 2 leave: %v", msgs)
		}
		n := len(msgs)
		a.inputs <- sim.Input{Seq: 1, DX: 1}
		time.Sleep(3 * tick)
		synctest.Wait()
		for _, m := range a.out.all()[n:] {
			for _, e := range m.GetSnapshot().GetEntities() {
				if e.GetId() == 2 {
					t.Fatal("a snapshot after the leave mentions the player who left")
				}
			}
		}
		z.Leave(2) // twice, and for someone never there: harmless
		z.Leave(99)
	})
}

func TestIdlePlayersGetOnlyKeepalives(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		defer stop()
		a := join(t, z, 1, "A")
		_ = join(t, z, 2, "B")
		time.Sleep(100 * tick) // five seconds, nobody moves
		synctest.Wait()
		var ticks []uint32
		for _, m := range a.out.all() {
			if s := m.GetSnapshot(); s != nil {
				if len(s.GetEntities()) != 0 {
					t.Fatalf("an idle world sent entities: %v", s)
				}
				ticks = append(ticks, s.GetTick())
			}
		}
		if len(ticks) != 5 {
			t.Fatalf("%d keepalive snapshots in 5 s, want 5: %v", len(ticks), ticks)
		}
		for i := 1; i < len(ticks); i++ {
			if ticks[i]-ticks[i-1] != 20 {
				t.Fatalf("keepalives at ticks %v, want every 20", ticks)
			}
		}
	})
}

func TestAnAckAloneSendsASnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		defer stop()
		a := join(t, z, 1, "A")
		time.Sleep(3 * tick)
		synctest.Wait()
		n := len(a.out.all())
		a.inputs <- sim.Input{Seq: 1} // standing still: nothing changes but the ack
		time.Sleep(tick)
		synctest.Wait()
		msgs := a.out.all()[n:]
		if len(msgs) != 1 || msgs[0].GetSnapshot().GetAckInputSeq() != 1 || len(msgs[0].GetSnapshot().GetEntities()) != 0 {
			t.Fatalf("after a standing-still input: %v", msgs)
		}
	})
}

// TestDeltasRebuildTheWorld: a client that starts from ServerHello and
// EntityJoined, then applies every Snapshot's changed entities, always
// knows exactly the server's state.
func TestDeltasRebuildTheWorld(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		defer stop()
		const n = 6
		ps := make([]*player, n)
		for i := range ps {
			ps[i] = join(t, z, uint32(i+1), fmt.Sprintf("P%d", i+1))
		}
		seqs := make([]uint32, n)
		r := rand.New(rand.NewPCG(3, 4))
		for range 200 {
			for i, p := range ps {
				if r.IntN(3) > 0 {
					seqs[i]++
					p.inputs <- sim.Input{Seq: seqs[i], DX: r.Int32N(3) - 1, DY: r.Int32N(3) - 1}
				}
			}
			time.Sleep(tick)
		}
		synctest.Wait()
		truth := map[uint32]*pb.EntityState{}
		done := make(chan struct{})
		z.cmds <- command{inspect: func(w *sim.World) {
			for _, p := range w.Players() {
				truth[p.ID] = entityPB(entity(p))
			}
			close(done)
		}}
		<-done
		for _, p := range ps {
			known := map[uint32]*pb.EntityState{}
			for _, m := range p.out.all() {
				switch {
				case m.GetServerHello() != nil:
					known[m.GetServerHello().GetSpawn().GetId()] = m.GetServerHello().GetSpawn()
				case m.GetEntitiesJoined() != nil:
					for _, e := range m.GetEntitiesJoined().GetEntities() {
						known[e.GetState().GetId()] = e.GetState()
					}
				case m.GetEntitiesLeft() != nil:
					for _, id := range m.GetEntitiesLeft().GetIds() {
						delete(known, id)
					}
				case m.GetSnapshot() != nil:
					for _, e := range m.GetSnapshot().GetEntities() {
						known[e.GetId()] = e
					}
				}
			}
			if len(known) != len(truth) {
				t.Fatalf("player %d knows %d entities, want %d", p.id, len(known), len(truth))
			}
			for id, want := range truth {
				if got := known[id]; got.GetX() != want.GetX() || got.GetY() != want.GetY() || got.GetFacing() != want.GetFacing() || got.GetAnim() != want.GetAnim() {
					t.Fatalf("player %d's view of %d: %v, server has %v", p.id, id, got, want)
				}
			}
		}
	})
}

// TestJoinStormIsBatched: 600 players joining between two ticks cost
// every player a couple of messages, not one per arrival. (Before
// batching, newcomers past 256 players were evicted as too slow.)
func TestJoinStormIsBatched(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t), MaxPlayers: 1000})
		stop := run(z)
		defer stop()
		first := join(t, z, 1, "First")
		time.Sleep(tick)
		var ps []*player
		for i := uint32(2); i <= 600; i++ {
			ps = append(ps, join(t, z, i, fmt.Sprintf("P%d", i)))
		}
		time.Sleep(tick)
		synctest.Wait()
		// first: hello, the zone (itself), then the 599 newcomers in 2 messages.
		joined := 0
		for _, m := range first.out.all() {
			if j := m.GetEntitiesJoined(); j != nil {
				joined++
			}
		}
		if joined != 3 {
			t.Fatalf("first player got %d EntitiesJoined messages, want 3 (itself, then 599 in two batches)", joined)
		}
		// A newcomer: hello, then the whole zone of 600 in 2 messages.
		msgs := ps[len(ps)-1].out.all()
		total := 0
		for _, m := range msgs {
			if j := m.GetEntitiesJoined(); j != nil {
				total += len(j.GetEntities())
			}
		}
		if total != 600 || len(msgs) > 4 {
			t.Fatalf("newcomer heard about %d entities in %d messages, want 600 in at most 4", total, len(msgs))
		}
	})
}

func TestLeavingBeforeAnnouncedIsSilent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		z := New(Config{Map: testMap(t)})
		stop := run(z)
		defer stop()
		a := join(t, z, 1, "A")
		time.Sleep(tick)
		_ = join(t, z, 2, "Ghost")
		z.Leave(2) // gone before the tick that would announce it
		time.Sleep(2 * tick)
		synctest.Wait()
		for _, m := range a.out.all() {
			if m.GetEntitiesLeft() != nil {
				t.Fatal("told about a departure of someone never announced")
			}
			for _, e := range m.GetEntitiesJoined().GetEntities() {
				if e.GetState().GetId() == 2 {
					t.Fatal("announced someone who already left")
				}
			}
		}
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
		for seq := uint32(1); seq <= 20; seq++ {
			fast.inputs <- sim.Input{Seq: seq, DX: 1 - int32(seq%2)*2} // back and forth: a change every tick
			time.Sleep(tick)
		}
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
		a := join(t, z, 1, "A")
		a.inputs <- sim.Input{Seq: 1, DX: 1}
		time.Sleep(10 * tick)
		synctest.Wait()
		stop()
		if n := z.m.tick.Count(); n != 10 {
			t.Fatalf("%d ticks observed, want 10", n)
		}
		if z.m.players.Value() != 1 || z.m.skipped.Value() == 0 {
			t.Fatal("players gauge or skipped counter not set")
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
