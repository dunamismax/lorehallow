// Package bot is a headless Lorehallow client: it speaks the real protocol
// over a real WebSocket, so tests and load tests exercise exactly what a
// player's client does.
//
// A Bot owns one goroutine, its read loop. Sending inputs is driven from
// outside (a Swarm's shared drivers call Step on every bot each tick), so
// thousands of bots cost thousands of goroutines, not tens of thousands of
// timers.
package bot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/dunamismax/lorehallow/server/internal/pb"
	"github.com/dunamismax/lorehallow/server/internal/wire"
)

// Behavior is what a bot does each tick.
type Behavior int

// Behaviors.
const (
	// Wander walks in a random direction, changing every few seconds, and
	// sometimes stops.
	Wander Behavior = iota
	// Idle sends nothing but heartbeat answers, and one step a minute.
	Idle
	// Still sends a standing-still input every tick (a player afk with the
	// client open).
	Still
	// Follow walks toward another entity it can see.
	Follow
)

// Cheat is a hostile mode.
type Cheat int

// Cheat modes. A cheating bot breaks the rules on purpose; the server must
// shrug it off.
const (
	NoCheat Cheat = iota
	// Flood sends 50 inputs per tick.
	Flood
	// ForgeSeq sends inputs with random, repeated and huge sequence numbers.
	ForgeSeq
	// Oversize sends a message over the client size limit.
	Oversize
	// Unknown sends an envelope with an unknown payload field.
	Unknown
	// BadValues sends movement values outside -1..1.
	BadValues
)

// ParseBehavior and ParseCheat turn flag values into modes.
func ParseBehavior(s string) (Behavior, error) {
	switch s {
	case "wander":
		return Wander, nil
	case "idle":
		return Idle, nil
	case "still":
		return Still, nil
	case "follow":
		return Follow, nil
	}
	return 0, fmt.Errorf("unknown behavior %q (wander, idle, still, follow)", s)
}

// ParseCheat parses a cheat mode name ("" or "none" is NoCheat).
func ParseCheat(s string) (Cheat, error) {
	switch s {
	case "", "none":
		return NoCheat, nil
	case "flood":
		return Flood, nil
	case "forge-seq":
		return ForgeSeq, nil
	case "oversize":
		return Oversize, nil
	case "unknown":
		return Unknown, nil
	case "bad-values":
		return BadValues, nil
	}
	return 0, fmt.Errorf("unknown cheat %q (flood, forge-seq, oversize, unknown, bad-values)", s)
}

// Config configures one bot.
type Config struct {
	URL      string
	Name     string
	Behavior Behavior
	Cheat    Cheat
	Seed     uint64
	// ProtocolVersion overrides the version sent in ClientHello (tests).
	ProtocolVersion uint32
	// HTTPClient dials the WebSocket (nil: a default client).
	HTTPClient *http.Client
	// Stats, if set, aggregates counters across many bots.
	Stats *Stats
}

// Entity is what a bot knows about one entity.
type Entity struct {
	ID     uint32
	X, Y   int32
	Facing pb.Facing
	Anim   pb.Anim
	Name   string
	Tick   uint32 // server tick of the last update
}

// DisconnectError is returned when the server sends a Disconnect.
type DisconnectError struct {
	Reason pb.DisconnectReason
	Text   string
}

func (e *DisconnectError) Error() string {
	return fmt.Sprintf("disconnected: %s: %s", e.Reason, e.Text)
}

// Bot is one connected client.
type Bot struct {
	cfg  Config
	conn *websocket.Conn
	rng  *rand.Rand

	ID      uint32
	Spawn   Entity
	MapID   string
	JoinRTT time.Duration // dial start to ServerHello

	rbuf    bytes.Buffer // only the read loop (and Dial before it) reads
	seq     uint32
	dx, dy  int32
	turnAt  int
	ticks   int
	writeMu sync.Mutex // Step and the reader's Pong both write

	mu         sync.Mutex
	known      map[uint32]*Entity
	lastAck    uint32
	lastTick   uint32
	snapshots  int
	leaves     map[uint32]time.Time
	joins      map[uint32]time.Time
	disconnect *DisconnectError
	done       chan struct{}
	err        error
}

// Dial connects, sends ClientHello and waits for ServerHello. On return
// the bot's read loop is running.
func Dial(ctx context.Context, cfg Config) (*Bot, error) {
	start := time.Now()
	conn, _, err := websocket.Dial(ctx, cfg.URL, &websocket.DialOptions{HTTPClient: cfg.HTTPClient})
	if err != nil {
		cfg.Stats.dialFailed()
		return nil, fmt.Errorf("dial: %w", err)
	}
	conn.SetReadLimit(wire.MaxServerMessage)
	b := &Bot{
		cfg:    cfg,
		conn:   conn,
		rng:    rand.New(rand.NewPCG(cfg.Seed, cfg.Seed^0x5eed)),
		known:  map[uint32]*Entity{},
		leaves: map[uint32]time.Time{},
		joins:  map[uint32]time.Time{},
		done:   make(chan struct{}),
	}
	pv := cfg.ProtocolVersion
	if pv == 0 {
		pv = pb.ProtocolVersion
	}
	if err := b.send(ctx, &pb.Envelope{Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{
		ProtocolVersion: pv, ClientVersion: "lorehallow-bot", DisplayName: cfg.Name,
	}}}); err != nil {
		_ = conn.CloseNow()
		return nil, err
	}
	// Read until ServerHello (or a Disconnect).
	var env pb.Envelope
	for {
		if err := b.read(ctx, &env); err != nil {
			_ = conn.CloseNow()
			cfg.Stats.dialFailed()
			return nil, err
		}
		if d := env.GetDisconnect(); d != nil {
			_ = conn.CloseNow()
			cfg.Stats.dialFailed()
			return nil, &DisconnectError{Reason: d.GetReason(), Text: d.GetText()}
		}
		if h := env.GetServerHello(); h != nil {
			b.ID, b.MapID = h.GetPlayerId(), h.GetMapId()
			b.Spawn = entityFrom(h.GetSpawn())
			b.Spawn.Name = cfg.Name
			self := b.Spawn
			b.known[b.ID] = &self // snapshots only mention it once it changes
			b.JoinRTT = time.Since(start)
			break
		}
		if p := env.GetPing(); p != nil {
			b.pong(ctx, p)
		}
	}
	cfg.Stats.joined(b.JoinRTT)
	go b.readLoop()
	return b, nil
}

func entityFrom(s *pb.EntityState) Entity {
	return Entity{ID: s.GetId(), X: s.GetX(), Y: s.GetY(), Facing: s.GetFacing(), Anim: s.GetAnim()}
}

func (b *Bot) read(ctx context.Context, env *pb.Envelope) error {
	_, r, err := b.conn.Reader(ctx)
	if err != nil {
		return err
	}
	b.rbuf.Reset()
	if _, err := io.Copy(&b.rbuf, r); err != nil {
		return err
	}
	b.cfg.Stats.received(b.rbuf.Len())
	return wire.DecodeServer(b.rbuf.Bytes(), env)
}

func (b *Bot) send(ctx context.Context, env *pb.Envelope) error {
	buf, err := wire.Encode(nil, env)
	if err != nil {
		return err
	}
	return b.write(ctx, buf)
}

func (b *Bot) write(ctx context.Context, buf []byte) error {
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	if err := b.conn.Write(ctx, websocket.MessageBinary, buf); err != nil {
		return err
	}
	b.cfg.Stats.sent(len(buf))
	return nil
}

func (b *Bot) pong(ctx context.Context, p *pb.Ping) {
	_ = b.send(ctx, &pb.Envelope{Payload: &pb.Envelope_Pong{Pong: &pb.Pong{
		Nonce: p.GetNonce(), PingSentAtMs: p.GetSentAtMs(), ReceivedAtMs: time.Now().UnixMilli(),
	}}})
}

func (b *Bot) readLoop() {
	defer close(b.done)
	ctx := context.Background()
	var env pb.Envelope
	for {
		if err := b.read(ctx, &env); err != nil {
			if errors.Is(err, wire.ErrUnknownPayload) {
				continue // a newer (or older) server's message: skip it
			}
			b.mu.Lock()
			if b.disconnect == nil {
				b.err = err
			}
			b.mu.Unlock()
			b.cfg.Stats.closed(b.Disconnect())
			return
		}
		b.handle(ctx, &env)
	}
}

func (b *Bot) handle(ctx context.Context, env *pb.Envelope) {
	now := time.Now()
	switch p := env.GetPayload().(type) {
	case *pb.Envelope_Snapshot:
		s := p.Snapshot
		b.mu.Lock()
		b.snapshots++
		b.lastAck, b.lastTick = s.GetAckInputSeq(), s.GetTick()
		for _, e := range s.GetEntities() {
			k := b.known[e.GetId()]
			if k == nil {
				k = &Entity{ID: e.GetId()}
				b.known[e.GetId()] = k
			}
			k.X, k.Y, k.Facing, k.Anim, k.Tick = e.GetX(), e.GetY(), e.GetFacing(), e.GetAnim(), s.GetTick()
		}
		b.mu.Unlock()
		b.cfg.Stats.snapshot()
	case *pb.Envelope_EntitiesJoined:
		b.mu.Lock()
		for _, in := range p.EntitiesJoined.GetEntities() {
			e := entityFrom(in.GetState())
			if e.ID == b.ID {
				continue // the whole-zone batch includes us; we know ourselves
			}
			e.Name = in.GetDisplayName()
			b.known[e.ID] = &e
			b.joins[e.ID] = now
		}
		b.mu.Unlock()
	case *pb.Envelope_EntitiesLeft:
		b.mu.Lock()
		for _, id := range p.EntitiesLeft.GetIds() {
			delete(b.known, id)
			b.leaves[id] = now
		}
		b.mu.Unlock()
	case *pb.Envelope_Ping:
		b.pong(ctx, p.Ping)
	case *pb.Envelope_Disconnect:
		b.mu.Lock()
		b.disconnect = &DisconnectError{Reason: p.Disconnect.GetReason(), Text: p.Disconnect.GetText()}
		b.mu.Unlock()
	}
}

// Step sends this tick's input according to the bot's behavior (or cheat).
// Call it at the tick rate from one goroutine at a time.
func (b *Bot) Step(ctx context.Context) error {
	b.ticks++
	switch b.cfg.Cheat {
	case Flood:
		for range 50 {
			if err := b.input(ctx, 1, 0); err != nil {
				return err
			}
		}
		return nil
	case ForgeSeq:
		switch b.rng.IntN(3) {
		case 0:
			b.seq = b.rng.Uint32()
		case 1: // replay an old one
		default:
			b.seq++
		}
		return b.send(ctx, inputEnv(b.seq, 1, 0))
	case Oversize:
		return b.write(ctx, make([]byte, wire.MaxClientMessage*4))
	case Unknown:
		return b.write(ctx, []byte{0xfa, 0x07, 0x01, 0x00}) // field 127, 1 byte
	case BadValues:
		b.seq++
		return b.send(ctx, inputEnv(b.seq, 7, -9))
	}
	switch b.cfg.Behavior {
	case Idle:
		if b.ticks%(60*20) == 0 { // one step a minute
			return b.input(ctx, 1, 0)
		}
		return nil
	case Still:
		return b.input(ctx, 0, 0)
	case Follow:
		dx, dy := b.towardNearest()
		return b.input(ctx, dx, dy)
	default: // Wander
		if b.ticks >= b.turnAt {
			b.turnAt = b.ticks + 20 + b.rng.IntN(60) // 1 to 4 seconds
			if b.rng.IntN(4) == 0 {
				b.dx, b.dy = 0, 0
			} else {
				b.dx, b.dy = int32(b.rng.IntN(3)-1), int32(b.rng.IntN(3)-1)
			}
		}
		return b.input(ctx, b.dx, b.dy)
	}
}

func (b *Bot) input(ctx context.Context, dx, dy int32) error {
	b.seq++
	return b.send(ctx, inputEnv(b.seq, dx, dy))
}

func inputEnv(seq uint32, dx, dy int32) *pb.Envelope {
	return &pb.Envelope{Payload: &pb.Envelope_Input{Input: &pb.PlayerInput{Seq: seq, MoveX: dx, MoveY: dy}}}
}

func (b *Bot) towardNearest() (int32, int32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	me := b.known[b.ID]
	if me == nil {
		return 0, 0
	}
	var best *Entity
	var bestD int64
	for id, e := range b.known {
		if id == b.ID {
			continue
		}
		d := sq(int64(e.X-me.X)) + sq(int64(e.Y-me.Y))
		if best == nil || d < bestD {
			best, bestD = e, d
		}
	}
	if best == nil || bestD < sq(64*16) { // close enough: 64 px
		return 0, 0
	}
	return sign(best.X - me.X), sign(best.Y - me.Y)
}

func sq(v int64) int64 { return v * v }

func sign(v int32) int32 {
	switch {
	case v > 64:
		return 1
	case v < -64:
		return -1
	}
	return 0
}

// Ping sends a Ping (heartbeat from the client side).
func (b *Bot) Ping(ctx context.Context) error {
	return b.send(ctx, &pb.Envelope{Payload: &pb.Envelope_Ping{Ping: &pb.Ping{SentAtMs: time.Now().UnixMilli()}}})
}

// Self returns what the bot last heard about itself.
func (b *Bot) Self() (Entity, bool) { return b.Known(b.ID) }

// Known returns what the bot last heard about entity id.
func (b *Bot) Known(id uint32) (Entity, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e, ok := b.known[id]
	if !ok {
		return Entity{}, false
	}
	return *e, true
}

// Visible returns how many entities the bot knows about, itself included.
func (b *Bot) Visible() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.known)
}

// LeftAt returns when the bot heard EntityLeft for id.
func (b *Bot) LeftAt(id uint32) (time.Time, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.leaves[id]
	return t, ok
}

// Snapshots returns how many snapshots the bot has received and the last
// acknowledged input seq.
func (b *Bot) Snapshots() (n int, ack uint32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.snapshots, b.lastAck
}

// Done is closed when the connection ends.
func (b *Bot) Done() <-chan struct{} { return b.done }

// Disconnect returns the server's Disconnect, if one arrived.
func (b *Bot) Disconnect() *DisconnectError {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.disconnect
}

// Err returns why the read loop ended, when it wasn't a Disconnect.
func (b *Bot) Err() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.err
}

// Close hangs up and waits for the read loop to end.
func (b *Bot) Close() {
	_ = b.conn.Close(websocket.StatusNormalClosure, "bye")
	<-b.done
}

// CloseNow drops the connection without a close handshake (a crash or a
// pulled cable, from the server's point of view).
func (b *Bot) CloseNow() {
	_ = b.conn.CloseNow()
	<-b.done
}

// Stats aggregates counters across many bots. All methods accept nil.
type Stats struct {
	Joined, DialFailed, Closed atomic.Int64
	Snapshots                  atomic.Int64
	MessagesIn, BytesIn        atomic.Int64
	MessagesOut, BytesOut      atomic.Int64
	JoinNanos                  atomic.Int64 // sum of JoinRTT
	MaxJoinNanos               atomic.Int64
	mu                         sync.Mutex
	Reasons                    map[pb.DisconnectReason]int
}

func (s *Stats) joined(rtt time.Duration) {
	if s == nil {
		return
	}
	s.Joined.Add(1)
	s.JoinNanos.Add(int64(rtt))
	for {
		m := s.MaxJoinNanos.Load()
		if int64(rtt) <= m || s.MaxJoinNanos.CompareAndSwap(m, int64(rtt)) {
			break
		}
	}
}

func (s *Stats) dialFailed() {
	if s != nil {
		s.DialFailed.Add(1)
	}
}

func (s *Stats) received(n int) {
	if s != nil {
		s.MessagesIn.Add(1)
		s.BytesIn.Add(int64(n))
	}
}

func (s *Stats) sent(n int) {
	if s != nil {
		s.MessagesOut.Add(1)
		s.BytesOut.Add(int64(n))
	}
}

func (s *Stats) snapshot() {
	if s != nil {
		s.Snapshots.Add(1)
	}
}

func (s *Stats) closed(d *DisconnectError) {
	if s == nil {
		return
	}
	s.Closed.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Reasons == nil {
		s.Reasons = map[pb.DisconnectReason]int{}
	}
	r := pb.DisconnectReason_DISCONNECT_REASON_UNSPECIFIED
	if d != nil {
		r = d.Reason
	}
	s.Reasons[r]++
}

// ReasonCounts returns a copy of the disconnect reasons seen.
func (s *Stats) ReasonCounts() map[pb.DisconnectReason]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[pb.DisconnectReason]int{}
	for k, v := range s.Reasons {
		out[k] = v
	}
	return out
}

// ErrClosed is returned by Step on a bot whose connection has ended.
var ErrClosed = errors.New("bot: closed")

// Move sends one input with the given intent (tests and scripted bots).
func (b *Bot) Move(ctx context.Context, dx, dy int32) error { return b.input(ctx, dx, dy) }

// LastTick returns the server tick of the latest snapshot.
func (b *Bot) LastTick() uint32 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastTick
}
