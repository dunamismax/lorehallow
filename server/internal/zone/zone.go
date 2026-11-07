// Package zone runs one zone: a goroutine that owns a sim.World and is
// the only code that reads or writes it (ADR 0005).
//
// Everything else talks to a zone through channels at its edges:
//
//   - Join and Leave go through a bounded command channel.
//   - Each player's inputs arrive on that player's own bounded channel,
//     which the tick drains without blocking.
//   - Everything the zone sends goes to each player's Outbox, whose Send
//     never blocks. A full outbox is the connection's problem (it evicts
//     itself); the tick never waits for a socket.
//
// Each tick the zone encodes every entity that changed, once, into a
// shared, reference counted wire.Body, and every player's Snapshot is
// assembled from it by that player's writer goroutine. The transport is
// reliable and ordered, so unchanged entities are simply left out
// (ADR 0008), and a player with nothing new gets no Snapshot at all,
// apart from a keepalive every KeepaliveTicks.
package zone

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/dunamismax/lorehallow/server/internal/mapdata"
	"github.com/dunamismax/lorehallow/server/internal/metrics"
	"github.com/dunamismax/lorehallow/server/internal/pb"
	"github.com/dunamismax/lorehallow/server/internal/sim"
	"github.com/dunamismax/lorehallow/server/internal/wire"
)

// Outbox is how the zone sends to one player. Send must not block: it
// returns false when the player's queue is full, and the implementation
// then closes that connection itself. Send takes ownership of the
// message: it releases it (wire.Outgoing.Release) whether or not it was
// queued, so callers never release after a Send. Kick asks the
// connection to close with a reason; it must not block either.
type Outbox interface {
	Send(wire.Outgoing) bool
	Kick(reason pb.DisconnectReason, text string)
}

// JoinRequest asks the zone to add a player.
type JoinRequest struct {
	ID     uint32
	Name   string
	Out    Outbox
	Inputs <-chan sim.Input
}

// JoinResult is the zone's answer.
type JoinResult struct {
	Spawn wire.Entity
	Tick  uint32
}

// ErrFull means the zone has MaxPlayers players.
var ErrFull = errors.New("zone: full")

// ErrStopped means the zone is not running.
var ErrStopped = errors.New("zone: stopped")

// Config configures a zone.
type Config struct {
	Map *mapdata.Map
	// MaxPlayers caps the players in the zone.
	MaxPlayers int
	// TickRate is ticks per second (sim.TickRate unless testing).
	TickRate int
	// MaxStrikes is how many invalid or flooding inputs a player may send
	// before it is kicked.
	MaxStrikes int
	// KeepaliveTicks is the longest a player goes without a Snapshot when
	// nothing it can see changes (default: one second of ticks).
	KeepaliveTicks int
	// ServerVersion is reported in ServerHello.
	ServerVersion string
	Log           *slog.Logger
	Metrics       *metrics.Registry
}

// Zone is one running zone. Create it with New and run it with Run.
type Zone struct {
	cfg     Config
	world   *sim.World
	cmds    chan command
	stopped chan struct{}
	members map[uint32]*member
	spawn   int
	// Arrivals and departures since the last tick, announced in batches.
	arrived  []*member
	departed []uint32

	m zoneMetrics
}

type member struct {
	id      uint32
	name    string
	out     Outbox
	inputs  <-chan sim.Input
	strikes int
	kicked  bool
	// What the player last got a Snapshot for.
	sentTick uint32
	sentAck  uint32
	// fresh: joined since the last tick; hasn't been told about the zone.
	fresh bool
}

type command struct {
	join  *JoinRequest
	leave uint32
	reply chan joinReply
	// inspect runs on the zone goroutine (tests only).
	inspect func(*sim.World)
}

type joinReply struct {
	res JoinResult
	err error
}

type zoneMetrics struct {
	tick       *metrics.Histogram
	overruns   *metrics.Counter
	players    *metrics.Gauge
	sendFull   *metrics.Counter
	inputs     *metrics.Counter
	dropped    *metrics.Counter
	kicks      *metrics.Counter
	snapshotSz *metrics.Gauge
	skipped    *metrics.Counter
}

// New creates a zone. Call Run to start it.
func New(cfg Config) *Zone {
	if cfg.TickRate == 0 {
		cfg.TickRate = sim.TickRate
	}
	if cfg.MaxPlayers == 0 {
		cfg.MaxPlayers = 1000
	}
	if cfg.MaxStrikes == 0 {
		cfg.MaxStrikes = 40
	}
	if cfg.KeepaliveTicks == 0 {
		cfg.KeepaliveTicks = cfg.TickRate
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	if cfg.Metrics == nil {
		cfg.Metrics = metrics.NewRegistry()
	}
	r := cfg.Metrics
	z := &Zone{
		cfg:     cfg,
		world:   sim.NewWorld(cfg.Map),
		cmds:    make(chan command, 256),
		stopped: make(chan struct{}),
		members: map[uint32]*member{},
	}
	z.m = zoneMetrics{
		tick:       r.NewHistogram("lorehallow_zone_tick_seconds", "Time to run one zone tick, from input drain to the last snapshot enqueued.", metrics.DurationBuckets()),
		overruns:   r.NewCounter("lorehallow_zone_tick_overruns_total", "Ticks that took longer than the tick interval."),
		players:    r.NewGauge("lorehallow_zone_players", "Players in the zone."),
		sendFull:   r.NewCounter("lorehallow_zone_send_full_total", "Messages the zone could not enqueue because a player's send queue was full."),
		inputs:     r.NewCounter("lorehallow_zone_inputs_total", "Inputs accepted by the simulation."),
		dropped:    r.NewCounter("lorehallow_zone_inputs_dropped_total", "Inputs rejected as stale, duplicate, flooding or invalid."),
		kicks:      r.NewCounter("lorehallow_zone_kicks_total", "Players kicked for repeated invalid or flooding input."),
		snapshotSz: r.NewGauge("lorehallow_zone_snapshot_body_bytes", "Size of the shared entity bytes (changed entities only) in the last tick."),
		skipped:    r.NewCounter("lorehallow_zone_snapshots_skipped_total", "Per-player Snapshots not sent because nothing the player can see changed."),
	}
	return z
}

// Map returns the zone's map.
func (z *Zone) Map() *mapdata.Map { return z.cfg.Map }

// Join adds a player. On success the zone has queued the ServerHello on
// req.Out; at the start of the next tick the player gets EntitiesJoined
// with everyone in the zone (and everyone hears about it), then Snapshots
// until Leave.
func (z *Zone) Join(ctx context.Context, req JoinRequest) (JoinResult, error) {
	reply := make(chan joinReply, 1)
	select {
	case z.cmds <- command{join: &req, reply: reply}:
	case <-z.stopped:
		return JoinResult{}, ErrStopped
	case <-ctx.Done():
		return JoinResult{}, ctx.Err()
	}
	select {
	case r := <-reply:
		return r.res, r.err
	case <-z.stopped:
		return JoinResult{}, ErrStopped
	case <-ctx.Done():
		// The zone may still add the player; the caller's Leave removes it.
		return JoinResult{}, ctx.Err()
	}
}

// Leave removes a player. It is safe to call for a player that never
// joined, and after the zone has stopped.
func (z *Zone) Leave(id uint32) {
	select {
	case z.cmds <- command{leave: id}:
	case <-z.stopped:
	}
}

// Run runs the tick loop until ctx is cancelled.
func (z *Zone) Run(ctx context.Context) {
	defer close(z.stopped)
	interval := time.Second / time.Duration(z.cfg.TickRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	z.cfg.Log.Info("zone started", "map", z.cfg.Map.ID, "tick_rate", z.cfg.TickRate)
	for {
		select {
		case <-ctx.Done():
			z.cfg.Log.Info("zone stopped", "map", z.cfg.Map.ID, "tick", z.world.Tick)
			return
		case c := <-z.cmds:
			z.handle(c)
		case <-ticker.C:
			start := time.Now()
			z.tick()
			d := time.Since(start)
			z.m.tick.Observe(d)
			if d > interval {
				z.m.overruns.Inc()
				z.cfg.Log.Warn("tick overrun", "tick", z.world.Tick, "took", d, "budget", interval, "players", len(z.members))
			}
		}
	}
}

func (z *Zone) handle(c command) {
	if c.inspect != nil {
		c.inspect(z.world)
		return
	}
	if c.join != nil {
		res, err := z.join(c.join)
		c.reply <- joinReply{res, err}
		return
	}
	z.leave(c.leave)
}

func (z *Zone) join(req *JoinRequest) (JoinResult, error) {
	if len(z.members) >= z.cfg.MaxPlayers {
		return JoinResult{}, ErrFull
	}
	if _, dup := z.members[req.ID]; dup {
		return JoinResult{}, fmt.Errorf("zone: player %d already here", req.ID)
	}
	spawns := z.cfg.Map.Spawns
	sp := spawns[z.spawn%len(spawns)]
	z.spawn++
	p := z.world.Add(req.ID, sp)
	if p == nil {
		return JoinResult{}, fmt.Errorf("zone: player %d already in the world", req.ID)
	}
	m := &member{id: req.ID, name: req.Name, out: req.Out, inputs: req.Inputs, sentTick: z.world.Tick, fresh: true}
	self := entity(p)
	hello := encode(&pb.Envelope{Payload: &pb.Envelope_ServerHello{ServerHello: &pb.ServerHello{
		PlayerId: req.ID, TickRate: uint32(z.cfg.TickRate), ServerTick: z.world.Tick,
		Spawn: entityPB(self), MapId: z.cfg.Map.ID, ServerVersion: z.cfg.ServerVersion,
	}}})
	if !m.out.Send(wire.Outgoing{Bytes: hello}) {
		z.world.Remove(req.ID)
		z.m.sendFull.Inc()
		return JoinResult{}, errors.New("zone: new player's queue is full")
	}
	// Everyone hears about the newcomer, and the newcomer about everyone,
	// at the start of the next tick, in batches (flushArrivals).
	z.members[req.ID] = m
	z.arrived = append(z.arrived, m)
	z.m.players.Set(int64(len(z.members)))
	return JoinResult{Spawn: self, Tick: z.world.Tick}, nil
}

func (z *Zone) leave(id uint32) {
	m, ok := z.members[id]
	if !ok {
		return
	}
	delete(z.members, id)
	z.world.Remove(id)
	z.m.players.Set(int64(len(z.members)))
	if m.fresh {
		// Left before anyone heard it arrived: nobody needs to hear it left.
		z.arrived = slices.DeleteFunc(z.arrived, func(a *member) bool { return a == m })
		return
	}
	z.departed = append(z.departed, id)
}

// Batch limits: every message stays far under wire.MaxServerMessage
// (an EntityIntro is at most about 90 bytes with a 16-rune name).
const (
	maxIntrosPerMessage = 400
	maxIdsPerMessage    = 4000
)

// flushArrivals tells everyone about the tick's arrivals and departures,
// in as few messages as possible, each encoded once and shared:
//
//   - players already here get one EntitiesJoined with the newcomers and
//     one EntitiesLeft with who left;
//   - each newcomer gets the whole zone (itself included, which clients
//     ignore), shared by every newcomer of the tick.
//
// Coalescing per tick bounds the messages a player receives per tick,
// however fast players join: a login storm is a few messages a tick, not
// one per arrival (measured: per-arrival messages evicted newcomers once
// a zone passed 256 players).
func (z *Zone) flushArrivals() {
	if len(z.departed) > 0 {
		for _, chunk := range chunks(z.departed, maxIdsPerMessage) {
			b := encode(&pb.Envelope{Payload: &pb.Envelope_EntitiesLeft{EntitiesLeft: &pb.EntitiesLeft{Ids: chunk}}})
			for _, m := range z.members {
				if !m.fresh { // newcomers never heard of them
					z.send(m, wire.Outgoing{Bytes: b})
				}
			}
		}
		z.departed = z.departed[:0]
	}
	if len(z.arrived) == 0 {
		return
	}
	newcomers := make([]*pb.EntityIntro, 0, len(z.arrived))
	for _, m := range z.arrived {
		newcomers = append(newcomers, z.intro(m))
	}
	var news [][]byte
	for _, chunk := range chunks(newcomers, maxIntrosPerMessage) {
		news = append(news, encode(&pb.Envelope{Payload: &pb.Envelope_EntitiesJoined{EntitiesJoined: &pb.EntitiesJoined{Entities: chunk}}}))
	}
	everyone := make([]*pb.EntityIntro, 0, len(z.members))
	for _, p := range z.world.Players() {
		everyone = append(everyone, z.intro(z.members[p.ID]))
	}
	var all [][]byte
	for _, chunk := range chunks(everyone, maxIntrosPerMessage) {
		all = append(all, encode(&pb.Envelope{Payload: &pb.Envelope_EntitiesJoined{EntitiesJoined: &pb.EntitiesJoined{Entities: chunk}}}))
	}
	for _, m := range z.members {
		msgs := news
		if m.fresh {
			msgs = all
		}
		for _, b := range msgs {
			z.send(m, wire.Outgoing{Bytes: b})
		}
	}
	for _, m := range z.arrived {
		m.fresh = false
	}
	z.arrived = z.arrived[:0]
}

func (z *Zone) intro(m *member) *pb.EntityIntro {
	return &pb.EntityIntro{State: entityPB(entity(z.world.Player(m.id))), DisplayName: m.name}
}

func chunks[T any](s []T, n int) [][]T {
	var out [][]T
	for len(s) > n {
		out = append(out, s[:n:n])
		s = s[n:]
	}
	return append(out, s)
}

// send enqueues without blocking; a full queue is counted, and the
// connection evicts itself.
func (z *Zone) send(m *member, o wire.Outgoing) {
	if !m.out.Send(o) {
		z.m.sendFull.Inc()
	}
}

func (z *Zone) tick() {
	// 0. Announce the arrivals and departures since the last tick, with
	// everyone's state before this tick's movement (which follows as the
	// tick's Snapshot).
	z.flushArrivals()

	// 1. Drain each player's inputs into the simulation, at most a
	// channel's worth, so a flooding reader can't keep the tick here.
	for _, m := range z.members {
		for range cap(m.inputs) {
			var in sim.Input
			select {
			case in = <-m.inputs:
			default:
				goto drained
			}
			switch z.world.Queue(m.id, in) {
			case sim.Accepted:
				z.m.inputs.Inc()
			case sim.Invalid, sim.Full:
				m.strikes++
				z.m.dropped.Inc()
			default: // stale or duplicate: harmless (late or replayed)
				z.m.dropped.Inc()
			}
		}
	drained:
		if m.strikes >= z.cfg.MaxStrikes && !m.kicked {
			m.kicked = true
			z.m.kicks.Inc()
			z.cfg.Log.Warn("kicking player for invalid or flooding input", "player", m.id, "strikes", m.strikes)
			m.out.Kick(pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, "Too many invalid inputs.")
		}
	}

	// 2. Advance the rules one tick.
	z.world.Step()

	// 3. Encode every entity that changed, once, then hand the shared
	// bytes to every player with its own acknowledged input. A player
	// with nothing new (no changes, same ack) gets nothing, apart from a
	// keepalive Snapshot every KeepaliveTicks.
	body := wire.NewBody()
	for _, p := range z.world.Players() {
		if p.Changed {
			body.B = wire.AppendEntity(body.B, entity(p))
		}
	}
	z.m.snapshotSz.Set(int64(len(body.B)))
	tick := z.world.Tick
	for _, m := range z.members {
		p := z.world.Player(m.id)
		if len(body.B) == 0 && p.LastSeq == m.sentAck && int(tick-m.sentTick) < z.cfg.KeepaliveTicks {
			z.m.skipped.Inc()
			continue
		}
		m.sentTick, m.sentAck = tick, p.LastSeq
		body.Retain() // this reference now belongs to the Outbox
		if !m.out.Send(wire.Outgoing{Body: body, Tick: tick, Ack: p.LastSeq}) {
			z.m.sendFull.Inc()
		}
	}
	body.Release()
}

func entity(p *sim.Player) wire.Entity {
	return wire.Entity{ID: p.ID, X: p.X, Y: p.Y, Facing: pb.Facing(p.Facing), Anim: pb.Anim(p.Anim)}
}

func entityPB(e wire.Entity) *pb.EntityState {
	return &pb.EntityState{Id: e.ID, X: e.X, Y: e.Y, Facing: e.Facing, Anim: e.Anim}
}

func encode(env *pb.Envelope) []byte {
	b, err := wire.Encode(nil, env)
	if err != nil {
		panic(fmt.Sprintf("zone: encoding a server message failed: %v", err)) // only on a programming error
	}
	return b
}
