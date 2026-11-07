package sim

import (
	"hash/fnv"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/dunamismax/lorehallow/server/internal/mapdata"
	"github.com/dunamismax/lorehallow/server/internal/pb"
)

const tile = mapdata.TileSubpixels

// grid builds a map from rows of '.' (open) and '#' (solid).
func grid(t testing.TB, rows ...string) *mapdata.Map {
	t.Helper()
	w, h := len(rows[0]), len(rows)
	solid := make([]bool, w*h)
	var spawn mapdata.Point
	found := false
	for y, r := range rows {
		for x, c := range r {
			solid[y*w+x] = c == '#'
			if c == '.' && !found {
				spawn, found = mapdata.Point{X: int32(x)*tile + tile/2, Y: int32(y)*tile + tile/2}, true
			}
		}
	}
	m, err := mapdata.New("test", w, h, solid, []mapdata.Point{spawn})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// center is the feet position at the middle of tile (tx, ty).
func center(tx, ty int) mapdata.Point {
	return mapdata.Point{X: int32(tx)*tile + tile/2, Y: int32(ty)*tile + tile/2}
}

func open(t testing.TB, w, h int) *mapdata.Map {
	rows := make([]string, h)
	for i := range rows {
		rows[i] = strings.Repeat(".", w)
	}
	return grid(t, rows...)
}

// walk queues and applies n ticks of the same intent for player id.
func walk(w *World, id uint32, n int, dx, dy int32) {
	p := w.Player(id)
	for range n {
		w.Queue(id, Input{Seq: p.LastSeq + uint32(len(p.pending)) + 1, DX: dx, DY: dy})
		w.Step()
	}
}

func TestFacingAndAnimMatchTheProtocol(t *testing.T) {
	if FacingDown != Facing(pb.Facing_FACING_DOWN) || FacingUp != Facing(pb.Facing_FACING_UP) ||
		FacingLeft != Facing(pb.Facing_FACING_LEFT) || FacingRight != Facing(pb.Facing_FACING_RIGHT) ||
		AnimIdle != Anim(pb.Anim_ANIM_IDLE) || AnimWalk != Anim(pb.Anim_ANIM_WALK) {
		t.Fatal("sim enums drifted from lorehallow.v1")
	}
}

func TestMovementSpeeds(t *testing.T) {
	cases := []struct {
		name           string
		dx, dy         int32
		wantDX, wantDY int32
		facing         Facing
	}{
		{"right", 1, 0, SpeedStraight, 0, FacingRight},
		{"left", -1, 0, -SpeedStraight, 0, FacingLeft},
		{"down", 0, 1, 0, SpeedStraight, FacingDown},
		{"up", 0, -1, 0, -SpeedStraight, FacingUp},
		{"up-right", 1, -1, SpeedDiagonal, -SpeedDiagonal, FacingRight},
		{"down-left", -1, 1, -SpeedDiagonal, SpeedDiagonal, FacingLeft},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := NewWorld(open(t, 20, 20))
			start := center(10, 10)
			w.Add(1, start)
			walk(w, 1, 3, c.dx, c.dy)
			p := w.Player(1)
			if p.X-start.X != 3*c.wantDX || p.Y-start.Y != 3*c.wantDY {
				t.Fatalf("moved (%d, %d), want (%d, %d)", p.X-start.X, p.Y-start.Y, 3*c.wantDX, 3*c.wantDY)
			}
			if p.Facing != c.facing || p.Anim != AnimWalk || p.LastSeq != 3 {
				t.Fatalf("facing %d anim %d lastSeq %d", p.Facing, p.Anim, p.LastSeq)
			}
		})
	}
}

func TestStandingStillIsIdle(t *testing.T) {
	w := NewWorld(open(t, 5, 5))
	w.Add(1, center(2, 2))
	walk(w, 1, 1, 1, 0)
	walk(w, 1, 1, 0, 0)
	if p := w.Player(1); p.Anim != AnimIdle || p.Facing != FacingRight {
		t.Fatalf("after stopping: anim %d facing %d", p.Anim, p.Facing)
	}
	w.Step() // no input at all
	if w.Player(1).Anim != AnimIdle {
		t.Fatal("no input should be idle")
	}
}

func TestWallsStopMovementFlush(t *testing.T) {
	w := NewWorld(grid(t,
		"#####",
		"#...#",
		"#...#",
		"#...#",
		"#####",
	))
	w.Add(1, center(2, 2))
	walk(w, 1, 50, 1, 0)
	if p := w.Player(1); p.X != 4*tile-HalfWidth {
		t.Fatalf("x = %d, want flush with the wall at %d", p.X, 4*tile-HalfWidth)
	}
	walk(w, 1, 50, -1, 0)
	if p := w.Player(1); p.X != tile+HalfWidth {
		t.Fatalf("x = %d, want %d", p.X, tile+HalfWidth)
	}
	walk(w, 1, 50, 0, 1)
	if p := w.Player(1); p.Y != 4*tile {
		t.Fatalf("y = %d, want %d", p.Y, 4*tile)
	}
	walk(w, 1, 50, 0, -1)
	if p := w.Player(1); p.Y != tile+BoxHeight {
		t.Fatalf("y = %d, want %d", p.Y, tile+BoxHeight)
	}
	if w.Overlaps(w.Player(1).X, w.Player(1).Y) {
		t.Fatal("ended inside a wall")
	}
}

func TestSlidesAlongWalls(t *testing.T) {
	w := NewWorld(grid(t,
		"######",
		"#....#",
		"#....#",
		"######",
	))
	w.Add(1, center(1, 1))
	walk(w, 1, 4, 0, -1) // flush against the top wall
	y := w.Player(1).Y
	x := w.Player(1).X
	walk(w, 1, 5, 1, -1) // diagonal into the wall: slides right
	p := w.Player(1)
	if p.Y != y || p.X != x+5*SpeedDiagonal {
		t.Fatalf("slide: (%d, %d), want (%d, %d)", p.X, p.Y, x+5*SpeedDiagonal, y)
	}
}

func TestMapEdgeIsAWall(t *testing.T) {
	w := NewWorld(open(t, 3, 3))
	w.Add(1, center(0, 0))
	walk(w, 1, 20, -1, -1)
	if p := w.Player(1); p.X != HalfWidth || p.Y != BoxHeight {
		t.Fatalf("at the corner: (%d, %d), want (%d, %d)", p.X, p.Y, HalfWidth, BoxHeight)
	}
}

func TestQueueRejections(t *testing.T) {
	w := NewWorld(open(t, 5, 5))
	w.Add(1, center(2, 2))
	cases := []struct {
		in   Input
		want QueueResult
	}{
		{Input{Seq: 1, DX: 2}, Invalid},
		{Input{Seq: 1, DY: -5}, Invalid},
		{Input{Seq: 0}, Stale},
		{Input{Seq: 1, DX: 1}, Accepted},
		{Input{Seq: 1, DX: -1}, Duplicate},
	}
	for _, c := range cases {
		if got := w.Queue(1, c.in); got != c.want {
			t.Errorf("Queue(%+v) = %d, want %d", c.in, got, c.want)
		}
	}
	if got := w.Queue(99, Input{Seq: 1}); got != NoPlayer {
		t.Errorf("unknown player: %d", got)
	}
	w.Step()
	if got := w.Queue(1, Input{Seq: 1}); got != Stale {
		t.Errorf("replayed applied input: %d, want Stale", got)
	}
}

func TestInputsApplyInSeqOrder(t *testing.T) {
	w := NewWorld(open(t, 10, 10))
	start := center(5, 5)
	w.Add(1, start)
	w.Queue(1, Input{Seq: 2, DY: 1}) // arrives first, applies second
	w.Queue(1, Input{Seq: 1, DX: 1})
	w.Step()
	p := w.Player(1)
	if p.LastSeq != 1 || p.X != start.X+SpeedStraight || p.Y != start.Y {
		t.Fatalf("tick 1 applied seq %d at (%d, %d)", p.LastSeq, p.X-start.X, p.Y-start.Y)
	}
	w.Step()
	if p.LastSeq != 2 || p.Y != start.Y+SpeedStraight {
		t.Fatalf("tick 2 applied seq %d", p.LastSeq)
	}
}

func TestFloodNeverMovesFasterThanOneInputPerTick(t *testing.T) {
	w := NewWorld(open(t, 200, 3))
	start := center(1, 1)
	w.Add(1, start)
	seq := uint32(0)
	accepted, full := 0, 0
	const ticks = 100
	for range ticks {
		for range 50 { // 50 inputs per tick: a 1,000-per-second flood
			seq++
			switch w.Queue(1, Input{Seq: seq, DX: 1}) {
			case Accepted:
				accepted++
			case Full:
				full++
			}
		}
		w.Step()
	}
	moved := w.Player(1).X - start.X
	if moved > ticks*SpeedStraight {
		t.Fatalf("moved %d in %d ticks, more than %d", moved, ticks, ticks*SpeedStraight)
	}
	if full == 0 || w.Player(1).Pending() > MaxPending {
		t.Fatalf("flood not bounded: %d full, %d pending", full, w.Player(1).Pending())
	}
}

func TestCreditCatchesUpAfterJitterButNoMore(t *testing.T) {
	w := NewWorld(open(t, 50, 3))
	start := center(1, 1)
	w.Add(1, start)
	for range 10 { // 10 ticks with no input: credit caps at MaxCredit
		w.Step()
	}
	for s := uint32(1); s <= 6; s++ {
		w.Queue(1, Input{Seq: s, DX: 1})
	}
	w.Step()
	p := w.Player(1)
	if p.X-start.X != MaxCredit*SpeedStraight || p.LastSeq != MaxCredit {
		t.Fatalf("burst tick moved %d (seq %d), want %d", p.X-start.X, p.LastSeq, MaxCredit*SpeedStraight)
	}
	w.Step()
	if p.LastSeq != MaxCredit+1 {
		t.Fatalf("after the burst, one input per tick: lastSeq %d", p.LastSeq)
	}
}

func TestAddAndRemove(t *testing.T) {
	w := NewWorld(open(t, 5, 5))
	for _, id := range []uint32{5, 2, 9, 1} {
		if w.Add(id, center(1, 1)) == nil {
			t.Fatalf("Add(%d) failed", id)
		}
	}
	if w.Add(2, center(1, 1)) != nil {
		t.Fatal("duplicate id accepted")
	}
	var ids []uint32
	for _, p := range w.Players() {
		ids = append(ids, p.ID)
	}
	if want := []uint32{1, 2, 5, 9}; !equal(ids, want) {
		t.Fatalf("ids %v, want %v (sorted)", ids, want)
	}
	if !w.Remove(5) || w.Remove(5) || w.Player(5) != nil || len(w.Players()) != 3 {
		t.Fatal("remove")
	}
}

func equal(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// randomMap is open ground with scattered solid tiles and a solid border.
func randomMap(t testing.TB, r *rand.Rand, w, h int) *mapdata.Map {
	rows := make([]string, h)
	for y := range rows {
		var b strings.Builder
		for x := range w {
			if x == 0 || y == 0 || x == w-1 || y == h-1 || (r.IntN(5) == 0 && (x != 1 || y != 1)) {
				b.WriteByte('#')
			} else {
				b.WriteByte('.')
			}
		}
		rows[y] = b.String()
	}
	return grid(t, rows...)
}

// run plays a random game and returns a hash of the final world. It fails
// t if a player ever overlaps a wall or moves too far in one tick.
func run(t testing.TB, seed uint64, players, ticks int) uint64 {
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b9))
	m := randomMap(t, r, 24, 16)
	w := NewWorld(m)
	for id := 1; id <= players; id++ {
		w.Add(uint32(id), center(1, 1))
	}
	seqs := make([]uint32, players+1)
	for range ticks {
		for id := 1; id <= players; id++ {
			for range r.IntN(3) { // 0 to 2 inputs per tick: jitter
				seqs[id]++
				w.Queue(uint32(id), Input{Seq: seqs[id], DX: r.Int32N(3) - 1, DY: r.Int32N(3) - 1})
			}
		}
		before := make(map[uint32][2]int32, players)
		for _, p := range w.Players() {
			before[p.ID] = [2]int32{p.X, p.Y}
		}
		w.Step()
		for _, p := range w.Players() {
			if w.Overlaps(p.X, p.Y) {
				t.Fatalf("seed %d tick %d: player %d inside a wall at (%d, %d)", seed, w.Tick, p.ID, p.X, p.Y)
			}
			b := before[p.ID]
			if abs(p.X-b[0]) > MaxCredit*SpeedStraight || abs(p.Y-b[1]) > MaxCredit*SpeedStraight {
				t.Fatalf("seed %d: player %d moved too far in one tick", seed, p.ID)
			}
		}
	}
	h := fnv.New64a()
	for _, p := range w.Players() {
		for _, v := range []int32{int32(p.ID), p.X, p.Y, int32(p.Facing), int32(p.Anim), int32(p.LastSeq)} {
			_, _ = h.Write([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)})
		}
	}
	return h.Sum64()
}

func abs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func TestRandomPlayNeverEntersWalls(t *testing.T) {
	for seed := range uint64(30) {
		run(t, seed, 8, 400)
	}
}

func TestDeterministic(t *testing.T) {
	for seed := range uint64(5) {
		if a, b := run(t, seed, 10, 300), run(t, seed, 10, 300); a != b {
			t.Fatalf("seed %d: two runs differ (%x vs %x)", seed, a, b)
		}
	}
}

func crowd(t testing.TB, n int) (*World, []uint32) {
	w := NewWorld(open(t, 64, 64))
	seqs := make([]uint32, n+1)
	for id := 1; id <= n; id++ {
		w.Add(uint32(id), center(1+id%60, 1+id/60%60))
	}
	return w, seqs
}

func stepCrowd(w *World, seqs []uint32, tick int) {
	for id := 1; id < len(seqs); id++ {
		seqs[id]++
		dx := int32((id+tick/40)%3) - 1
		w.Queue(uint32(id), Input{Seq: seqs[id], DX: dx, DY: int32((id/3+tick/30)%3) - 1})
	}
	w.Step()
}

func TestStepDoesNotAllocate(t *testing.T) {
	w, seqs := crowd(t, 100)
	tick := 0
	allocs := testing.AllocsPerRun(200, func() {
		tick++
		stepCrowd(w, seqs, tick)
	})
	if allocs != 0 {
		t.Fatalf("a 100-player tick allocated %.1f times, want 0", allocs)
	}
}

func BenchmarkStep1000(b *testing.B) {
	w, seqs := crowd(b, 1000)
	tick := 0
	b.ReportAllocs()
	for b.Loop() {
		tick++
		stepCrowd(w, seqs, tick)
	}
}
