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
// Each tick the zone encodes every entity once into a shared, reference
// counted wire.Body, and every player's Snapshot is assembled from it by
// that player's writer goroutine.
package zone

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

	m zoneMetrics
}

type member struct {
	id      uint32
	name    string
	out     Outbox
	inputs  <-chan sim.Input
	strikes int
	kicked  bool
}

type command struct {
	join  *JoinRequest
	leave uint32
	reply chan joinReply
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
		snapshotSz: r.NewGauge("lorehallow_zone_snapshot_body_bytes", "Size of the shared entity bytes in the last snapshot."),
	}
	return z
}

// Map returns the zone's map.
func (z *Zone) Map() *mapdata.Map { return z.cfg.Map }

// Join adds a player. On success the zone has already queued the
// ServerHello and the EntityJoined messages for everyone else in the
// zone on req.Out, in that order, and the player gets a Snapshot every
// tick until Leave.
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
	m := &member{id: req.ID, name: req.Name, out: req.Out, inputs: req.Inputs}
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
	// Tell the newcomer about everyone, and everyone about the newcomer
	// (encoded once, shared by every recipient).
	joined := encode(joinedEnvelope(self, req.Name))
	for _, other := range z.members {
		if q := z.world.Player(other.id); q != nil {
			z.send(m, wire.Outgoing{Bytes: encode(joinedEnvelope(entity(q), other.name))})
		}
		z.send(other, wire.Outgoing{Bytes: joined})
	}
	z.members[req.ID] = m
	z.m.players.Set(int64(len(z.members)))
	return JoinResult{Spawn: self, Tick: z.world.Tick}, nil
}

func (z *Zone) leave(id uint32) {
	if _, ok := z.members[id]; !ok {
		return
	}
	delete(z.members, id)
	z.world.Remove(id)
	z.m.players.Set(int64(len(z.members)))
	left := encode(&pb.Envelope{Payload: &pb.Envelope_EntityLeft{EntityLeft: &pb.EntityLeft{Id: id}}})
	for _, m := range z.members {
		z.send(m, wire.Outgoing{Bytes: left})
	}
}

// send enqueues without blocking; a full queue is counted, and the
// connection evicts itself.
func (z *Zone) send(m *member, o wire.Outgoing) {
	if !m.out.Send(o) {
		z.m.sendFull.Inc()
	}
}

func (z *Zone) tick() {
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

	// 3. Encode every entity once, then hand the shared bytes to every
	// player with its own acknowledged input.
	body := wire.NewBody()
	for _, p := range z.world.Players() {
		body.B = wire.AppendEntity(body.B, entity(p))
	}
	z.m.snapshotSz.Set(int64(len(body.B)))
	for _, m := range z.members {
		p := z.world.Player(m.id)
		body.Retain() // this reference now belongs to the Outbox
		if !m.out.Send(wire.Outgoing{Body: body, Tick: z.world.Tick, Ack: p.LastSeq}) {
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

func joinedEnvelope(e wire.Entity, name string) *pb.Envelope {
	return &pb.Envelope{Payload: &pb.Envelope_EntityJoined{EntityJoined: &pb.EntityJoined{State: entityPB(e), DisplayName: name}}}
}

func encode(env *pb.Envelope) []byte {
	b, err := wire.Encode(nil, env)
	if err != nil {
		panic(fmt.Sprintf("zone: encoding a server message failed: %v", err)) // only on a programming error
	}
	return b
}
