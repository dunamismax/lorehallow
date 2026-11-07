// Package sim holds the game rules: a pure, deterministic simulation of
// one zone, advanced one tick at a time.
//
// It does no I/O, reads no clock and uses no randomness of its own, and it
// iterates entities in id order, so the same sequence of calls always
// produces the same world. The zone goroutine (internal/zone) owns a World
// and is the only code that touches it.
package sim

import (
	"slices"

	"github.com/dunamismax/lorehallow/server/internal/mapdata"
)

// TickRate is the number of simulation ticks per second.
const TickRate = 20

// Movement, in subpixels per tick (16 subpixels per pixel).
const (
	// SpeedStraight is 7 pixels per tick: 140 px/s, about 4.4 tiles a second.
	SpeedStraight = 112
	// SpeedDiagonal moves each axis 79 subpixels, so a diagonal step is
	// 111.7 subpixels long, the same speed as a straight one.
	SpeedDiagonal = 79
)

// The collision box at a character's feet, in subpixels: 14 px wide,
// 6 px tall, its bottom edge on the feet position.
const (
	HalfWidth = 7 * mapdata.SubpixelsPerPixel
	BoxHeight = 6 * mapdata.SubpixelsPerPixel
)

// Input buffering limits.
const (
	// MaxPending is how many inputs a player may have waiting. More is a
	// flood; the newest are dropped.
	MaxPending = 16
	// MaxCredit is how many ticks of movement a player can bank while no
	// input arrives (network jitter), to catch up later. A player gains one
	// credit per tick and spends one per input applied, so over any span of
	// ticks it can never move more ticks than have passed.
	MaxCredit = 3
)

// Facing values match lorehallow.v1.Facing.
type Facing uint8

// Facing directions.
const (
	FacingUnspecified Facing = iota
	FacingDown
	FacingUp
	FacingLeft
	FacingRight
)

// Anim values match lorehallow.v1.Anim.
type Anim uint8

// Animations.
const (
	AnimUnspecified Anim = iota
	AnimIdle
	AnimWalk
)

// Input is one PlayerInput as the simulation sees it.
type Input struct {
	Seq    uint32
	DX, DY int32 // each -1, 0 or 1
}

// QueueResult says what happened to a queued input.
type QueueResult uint8

// Queue results. Only Accepted changes anything.
const (
	Accepted  QueueResult = iota
	Stale                 // seq at or below the last applied: late or replayed
	Duplicate             // the same seq is already waiting
	Full                  // MaxPending inputs already waiting: flooding
	Invalid               // a movement value outside -1..1: a modified client
	NoPlayer              // the player isn't in the world
)

// Player is one character in the world.
type Player struct {
	ID      uint32
	X, Y    int32
	Facing  Facing
	Anim    Anim
	LastSeq uint32 // last input seq applied (acknowledged in snapshots)
	// Changed reports whether the last Step changed the player's visible
	// state (position, facing or animation). Snapshots send only these.
	Changed bool

	credit  int
	pending []Input // sorted by Seq, at most MaxPending
}

// World is one zone's simulation state.
type World struct {
	Map  *mapdata.Map
	Tick uint32

	players []*Player // sorted by ID
}

// NewWorld returns an empty world on m.
func NewWorld(m *mapdata.Map) *World { return &World{Map: m} }

// Players returns the players sorted by id. The slice and the players
// belong to the world; don't keep them across ticks.
func (w *World) Players() []*Player { return w.players }

// Player returns the player with id, or nil.
func (w *World) Player(id uint32) *Player {
	i, ok := w.find(id)
	if !ok {
		return nil
	}
	return w.players[i]
}

func (w *World) find(id uint32) (int, bool) {
	return slices.BinarySearchFunc(w.players, id, func(p *Player, id uint32) int {
		switch {
		case p.ID < id:
			return -1
		case p.ID > id:
			return 1
		}
		return 0
	})
}

// Add puts a new player at spawn. It returns nil if the id is taken.
func (w *World) Add(id uint32, spawn mapdata.Point) *Player {
	i, ok := w.find(id)
	if ok {
		return nil
	}
	p := &Player{ID: id, X: spawn.X, Y: spawn.Y, Facing: FacingDown, Anim: AnimIdle, pending: make([]Input, 0, MaxPending)}
	w.players = slices.Insert(w.players, i, p)
	return p
}

// Remove takes a player out of the world. It reports whether it was there.
func (w *World) Remove(id uint32) bool {
	i, ok := w.find(id)
	if ok {
		w.players = slices.Delete(w.players, i, i+1)
	}
	return ok
}

// Queue adds an input for the next ticks. Inputs are applied in Seq
// order, one tick of movement each, whatever order they arrive in.
func (w *World) Queue(id uint32, in Input) QueueResult {
	p := w.Player(id)
	if p == nil {
		return NoPlayer
	}
	if in.DX < -1 || in.DX > 1 || in.DY < -1 || in.DY > 1 {
		return Invalid
	}
	if in.Seq <= p.LastSeq {
		return Stale
	}
	i, found := slices.BinarySearchFunc(p.pending, in.Seq, func(q Input, seq uint32) int {
		switch {
		case q.Seq < seq:
			return -1
		case q.Seq > seq:
			return 1
		}
		return 0
	})
	if found {
		return Duplicate
	}
	if len(p.pending) >= MaxPending {
		return Full
	}
	p.pending = slices.Insert(p.pending, i, in)
	return Accepted
}

// Pending reports how many inputs a player has waiting.
func (p *Player) Pending() int { return len(p.pending) }

// Step advances the world one tick: every player, in id order, gains one
// movement credit and applies waiting inputs while it has credit.
func (w *World) Step() {
	w.Tick++
	for _, p := range w.players {
		x, y, facing, anim := p.X, p.Y, p.Facing, p.Anim
		p.credit = min(p.credit+1, MaxCredit)
		moving := false
		applied := 0
		for p.credit > 0 && applied < len(p.pending) {
			in := p.pending[applied]
			applied++
			p.credit--
			p.LastSeq = in.Seq
			if in.DX != 0 || in.DY != 0 {
				moving = true
				w.move(p, in.DX, in.DY)
			}
		}
		p.pending = append(p.pending[:0], p.pending[applied:]...)
		if moving {
			p.Anim = AnimWalk
		} else {
			p.Anim = AnimIdle
		}
		p.Changed = p.X != x || p.Y != y || p.Facing != facing || p.Anim != anim
	}
}

// move applies one tick of movement: x first, then y, each clamped
// against solid tiles (axis-separated collision), so a player slides
// along a wall instead of sticking to it.
func (w *World) move(p *Player, dx, dy int32) {
	speed := int32(SpeedStraight)
	if dx != 0 && dy != 0 {
		speed = SpeedDiagonal
	}
	switch {
	case dx < 0:
		p.Facing = FacingLeft
	case dx > 0:
		p.Facing = FacingRight
	case dy < 0:
		p.Facing = FacingUp
	case dy > 0:
		p.Facing = FacingDown
	}
	if dx != 0 {
		p.X = w.sweepX(p.X, p.Y, dx*speed)
	}
	if dy != 0 {
		p.Y = w.sweepY(p.X, p.Y, dy*speed)
	}
}

// Box edges: [x-HalfWidth, x+HalfWidth) × [y-BoxHeight, y).

func tileOf(v int32) int {
	if v < 0 {
		return int((v - mapdata.TileSubpixels + 1) / mapdata.TileSubpixels)
	}
	return int(v / mapdata.TileSubpixels)
}

// sweepX moves the box from x by d (|d| < one tile) and returns the new x,
// stopped flush against the first solid tile column in the way.
func (w *World) sweepX(x, y, d int32) int32 {
	top, bottom := tileOf(y-BoxHeight), tileOf(y-1)
	nx := x + d
	if d > 0 {
		col := tileOf(nx + HalfWidth - 1)
		if col != tileOf(x+HalfWidth-1) && w.columnSolid(col, top, bottom) {
			return int32(col)*mapdata.TileSubpixels - HalfWidth
		}
	} else {
		col := tileOf(nx - HalfWidth)
		if col != tileOf(x-HalfWidth) && w.columnSolid(col, top, bottom) {
			return int32(col+1)*mapdata.TileSubpixels + HalfWidth
		}
	}
	return nx
}

// sweepY is sweepX for the vertical axis.
func (w *World) sweepY(x, y, d int32) int32 {
	left, right := tileOf(x-HalfWidth), tileOf(x+HalfWidth-1)
	ny := y + d
	if d > 0 {
		row := tileOf(ny - 1)
		if row != tileOf(y-1) && w.rowSolid(row, left, right) {
			return int32(row) * mapdata.TileSubpixels
		}
	} else {
		row := tileOf(ny - BoxHeight)
		if row != tileOf(y-BoxHeight) && w.rowSolid(row, left, right) {
			return int32(row+1)*mapdata.TileSubpixels + BoxHeight
		}
	}
	return ny
}

func (w *World) columnSolid(col, top, bottom int) bool {
	for row := top; row <= bottom; row++ {
		if w.Map.Solid(col, row) {
			return true
		}
	}
	return false
}

func (w *World) rowSolid(row, left, right int) bool {
	for col := left; col <= right; col++ {
		if w.Map.Solid(col, row) {
			return true
		}
	}
	return false
}

// Overlaps reports whether a box with feet at (x, y) overlaps any solid
// tile. A valid world never lets a player's box overlap one.
func (w *World) Overlaps(x, y int32) bool {
	for row := tileOf(y - BoxHeight); row <= tileOf(y-1); row++ {
		for col := tileOf(x - HalfWidth); col <= tileOf(x+HalfWidth-1); col++ {
			if w.Map.Solid(col, row) {
				return true
			}
		}
	}
	return false
}
