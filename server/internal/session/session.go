// Package session runs one client connection: the WebSocket, the
// handshake, the read loop, the writer goroutine and the bounded send
// queue between the zone and the socket (ADR 0005, docs/design/protocol.md).
//
// Lifecycle, one way in and one way out:
//
//	handshaking ──ClientHello ok──► in world ──► closing ──► closed
//	     │                              │
//	     └── timeout, bad message,      └── peer closed, error, too slow,
//	         protocol mismatch, full        timed out, kicked, flooding,
//	                                        server shutting down
//
// Every exit goes through close(reason): it records the first reason,
// wakes the writer, which sends a Disconnect (when the reason calls for
// one) and closes the socket, which ends the read loop. Serve then
// removes the player from the zone and releases everything queued.
//
// Goroutines per connection: exactly two. The caller's goroutine runs
// the read loop; Serve starts one writer. The caller is a goroutine the
// game server starts for the session, not the HTTP handler, so net/http's
// request state can be collected (ADR 0010).
package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/coder/websocket"

	"github.com/dunamismax/lorehallow/server/internal/pb"
	"github.com/dunamismax/lorehallow/server/internal/sim"
	"github.com/dunamismax/lorehallow/server/internal/wire"
	"github.com/dunamismax/lorehallow/server/internal/zone"
)

// Config holds the per-connection limits. Zero fields take the defaults
// below, which docs/design/protocol.md documents.
type Config struct {
	HandshakeTimeout time.Duration // ClientHello must arrive within this
	PingInterval     time.Duration // server Ping cadence
	IdleTimeout      time.Duration // no message for this long: disconnect
	SendQueueLen     int           // messages
	SendQueueBytes   int           // bytes
	InputQueueLen    int           // inputs waiting for the zone
	RatePerSecond    float64       // inbound messages per second, sustained
	RateBurst        float64       // inbound burst
	MaxRateDrops     int           // messages dropped for rate before disconnect
	ProtocolVersion  uint32
}

// Defaults.
const (
	DefaultHandshakeTimeout = 5 * time.Second
	DefaultPingInterval     = 2 * time.Second
	DefaultIdleTimeout      = 15 * time.Second
	DefaultSendQueueLen     = 256
	DefaultSendQueueBytes   = 256 << 10
	DefaultInputQueueLen    = sim.MaxPending
	DefaultRatePerSecond    = 60
	DefaultRateBurst        = 120
	DefaultMaxRateDrops     = 120
	// MaxNameRunes bounds display names.
	MaxNameRunes = 16
	// CloseCodeBase + DisconnectReason is the WebSocket close code.
	CloseCodeBase = 4000
)

func (c Config) withDefaults() Config {
	set := func(v *time.Duration, d time.Duration) {
		if *v == 0 {
			*v = d
		}
	}
	set(&c.HandshakeTimeout, DefaultHandshakeTimeout)
	set(&c.PingInterval, DefaultPingInterval)
	set(&c.IdleTimeout, DefaultIdleTimeout)
	if c.SendQueueLen == 0 {
		c.SendQueueLen = DefaultSendQueueLen
	}
	if c.SendQueueBytes == 0 {
		c.SendQueueBytes = DefaultSendQueueBytes
	}
	if c.InputQueueLen == 0 {
		c.InputQueueLen = DefaultInputQueueLen
	}
	if c.RatePerSecond == 0 {
		c.RatePerSecond = DefaultRatePerSecond
	}
	if c.RateBurst == 0 {
		c.RateBurst = DefaultRateBurst
	}
	if c.MaxRateDrops == 0 {
		c.MaxRateDrops = DefaultMaxRateDrops
	}
	if c.ProtocolVersion == 0 {
		c.ProtocolVersion = pb.ProtocolVersion
	}
	return c
}

// Conn is the part of *websocket.Conn a session uses, so tests can drive a
// session with a fake connection.
type Conn interface {
	Reader(ctx context.Context) (websocket.MessageType, io.Reader, error)
	Write(ctx context.Context, typ websocket.MessageType, p []byte) error
	Close(code websocket.StatusCode, reason string) error
	CloseNow() error
	SetReadLimit(n int64)
}

// World is what a session needs from the game: a zone to join and leave,
// and player ids.
type World interface {
	Join(ctx context.Context, req zone.JoinRequest) (zone.JoinResult, error)
	Leave(id uint32)
}

// Deps are a session's collaborators.
type Deps struct {
	World   World
	NextID  func() uint32
	Log     *slog.Logger
	Metrics *Metrics
	// Joined, if set, is called once the player is in the world (tests use it).
	Joined func(id uint32)
}

// Session is one connection. It implements zone.Outbox.
type Session struct {
	cfg  Config
	deps Deps
	conn Conn
	log  *slog.Logger

	queue  *sendQueue
	inputs chan sim.Input

	id       uint32
	joined   atomic.Bool
	lastRead atomic.Int64 // unix nanos of the last message received

	closeOnce   sync.Once
	closing     chan struct{}
	reason      pb.DisconnectReason
	text        string
	writeCtx    context.Context
	writeCancel context.CancelFunc // interrupts a Write blocked on a stalled client

	rbuf []byte
	env  pb.Envelope

	tokens   float64
	tokensAt time.Time
	drops    int
}

// Serve runs conn until it closes and returns why. It blocks, and the
// goroutine that calls it becomes the reader. Cancelling ctx
// (server shutdown) disconnects the client with SHUTTING_DOWN.
func Serve(ctx context.Context, conn Conn, cfg Config, deps Deps) pb.DisconnectReason {
	cfg = cfg.withDefaults()
	if deps.Log == nil {
		deps.Log = slog.New(slog.DiscardHandler)
	}
	if deps.Metrics == nil {
		deps.Metrics = NewMetrics(nil)
	}
	s := &Session{
		cfg:      cfg,
		deps:     deps,
		conn:     conn,
		log:      deps.Log,
		queue:    newSendQueue(cfg.SendQueueLen, cfg.SendQueueBytes),
		inputs:   make(chan sim.Input, cfg.InputQueueLen),
		closing:  make(chan struct{}),
		rbuf:     make([]byte, 0, 64), // grows on demand up to the read limit; inputs are ~10 bytes
		tokens:   cfg.RateBurst,
		tokensAt: time.Now(),
	}
	s.writeCtx, s.writeCancel = context.WithCancel(context.Background())
	defer s.writeCancel()
	s.lastRead.Store(time.Now().UnixNano())
	conn.SetReadLimit(wire.MaxClientMessage)
	m := deps.Metrics
	m.Open.Add(1)
	defer m.Open.Add(-1)

	// The handshake deadline. AfterFunc runs only if it fires.
	hs := time.AfterFunc(cfg.HandshakeTimeout, func() {
		if !s.joined.Load() {
			s.close(pb.DisconnectReason_DISCONNECT_REASON_TIMED_OUT, "No handshake.")
		}
	})
	defer hs.Stop()

	// Shutdown: a closed ctx disconnects with SHUTTING_DOWN.
	stopWatch := context.AfterFunc(ctx, func() {
		s.close(pb.DisconnectReason_DISCONNECT_REASON_SHUTTING_DOWN, "The server is restarting.")
	})
	defer stopWatch()

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		s.writeLoop()
	}()

	s.readLoop()
	// The read loop ends when the socket closes. Make sure the writer knows.
	s.close(pb.DisconnectReason_DISCONNECT_REASON_UNSPECIFIED, "")
	<-writerDone

	if s.joined.Load() {
		deps.World.Leave(s.id)
	}
	s.queue.close()
	m.Closed.Inc()
	m.reasons[s.reason].Inc()
	s.log.Debug("connection closed", "player", s.id, "reason", s.reason.String())
	return s.reason
}

// close starts closing with reason. The first call wins and reports
// true; it never blocks. It also cancels the writer's context: a Write
// blocked on a client that stopped reading returns at once (and the
// library closes the socket), so eviction never waits on the very client
// being evicted.
func (s *Session) close(reason pb.DisconnectReason, text string) bool {
	won := false
	s.closeOnce.Do(func() {
		won = true
		s.reason, s.text = reason, text
		close(s.closing)
		s.writeCancel()
	})
	return won
}

// Send implements zone.Outbox. A full queue evicts the client.
func (s *Session) Send(o wire.Outgoing) bool {
	if s.queue.push(o) {
		return true
	}
	if s.close(pb.DisconnectReason_DISCONNECT_REASON_TOO_SLOW, "Your connection couldn't keep up.") {
		s.deps.Metrics.Evicted.Inc() // once per connection, not per refused message
	}
	return false
}

// Kick implements zone.Outbox.
func (s *Session) Kick(reason pb.DisconnectReason, text string) { _ = s.close(reason, text) }

// ---- Reading -----------------------------------------------------------

func (s *Session) readLoop() {
	ctx := context.Background() // closing the socket ends a blocked read
	for {
		typ, r, err := s.conn.Reader(ctx)
		if err != nil {
			if errors.Is(err, websocket.ErrMessageTooBig) {
				s.close(pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, "Message too large.")
			}
			return
		}
		b, err := readAll(r, s.rbuf[:0])
		s.rbuf = b[:0]
		if err != nil {
			if errors.Is(err, websocket.ErrMessageTooBig) {
				s.close(pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, "Message too large.")
			}
			return
		}
		s.lastRead.Store(time.Now().UnixNano())
		s.deps.Metrics.MessagesIn.Inc()
		s.deps.Metrics.BytesIn.Add(uint64(len(b)))
		if typ != websocket.MessageBinary {
			s.close(pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, "Binary messages only.")
			return
		}
		if !s.allow() {
			continue
		}
		if err := wire.DecodeClient(b, &s.env); err != nil {
			s.close(pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, "Malformed message.")
			return
		}
		if !s.handle(&s.env) {
			return
		}
	}
}

// readAll reads r into buf (bounded by the connection's read limit).
func readAll(r io.Reader, buf []byte) ([]byte, error) {
	for {
		if len(buf) == cap(buf) {
			buf = append(buf, 0)[:len(buf)]
		}
		n, err := r.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		if errors.Is(err, io.EOF) {
			return buf, nil
		}
		if err != nil {
			return buf, err
		}
	}
}

// allow applies the inbound token bucket. Over the limit, the message is
// dropped; past MaxRateDrops dropped messages, the client is disconnected.
func (s *Session) allow() bool {
	now := time.Now()
	s.tokens = min(s.cfg.RateBurst, s.tokens+now.Sub(s.tokensAt).Seconds()*s.cfg.RatePerSecond)
	s.tokensAt = now
	if s.tokens >= 1 {
		s.tokens--
		return true
	}
	s.drops++
	s.deps.Metrics.RateDropped.Inc()
	if s.drops > s.cfg.MaxRateDrops {
		s.close(pb.DisconnectReason_DISCONNECT_REASON_RATE_LIMITED, "Too many messages.")
	}
	return false
}

// handle processes one decoded message. It returns false to stop reading.
func (s *Session) handle(env *pb.Envelope) bool {
	if !s.joined.Load() {
		switch p := env.GetPayload().(type) {
		case *pb.Envelope_ClientHello:
			return s.hello(p.ClientHello)
		case *pb.Envelope_Ping:
			s.pong(p.Ping)
			return true
		default:
			s.close(pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, "Say hello first.")
			return false
		}
	}
	switch p := env.GetPayload().(type) {
	case *pb.Envelope_Input:
		in := sim.Input{Seq: p.Input.GetSeq(), DX: p.Input.GetMoveX(), DY: p.Input.GetMoveY()}
		select {
		case s.inputs <- in:
		default:
			s.deps.Metrics.InputDropped.Inc() // the zone validates and counts floods
		}
	case *pb.Envelope_Ping:
		s.pong(p.Ping)
	case *pb.Envelope_Pong:
		if sent := p.Pong.GetPingSentAtMs(); sent > 0 {
			if rtt := time.Now().UnixMilli() - sent; rtt >= 0 && rtt < 60_000 {
				s.deps.Metrics.RTT.Observe(time.Duration(rtt) * time.Millisecond)
			}
		}
	default:
		s.close(pb.DisconnectReason_DISCONNECT_REASON_BAD_MESSAGE, "Unexpected message.")
		return false
	}
	return true
}

func (s *Session) hello(h *pb.ClientHello) bool {
	if h.GetProtocolVersion() != s.cfg.ProtocolVersion {
		s.close(pb.DisconnectReason_DISCONNECT_REASON_PROTOCOL_MISMATCH,
			fmt.Sprintf("Update required: this server speaks protocol %d, your client speaks %d.", s.cfg.ProtocolVersion, h.GetProtocolVersion()))
		return false
	}
	name, ok := ValidName(h.GetDisplayName())
	if !ok {
		s.close(pb.DisconnectReason_DISCONNECT_REASON_INVALID_HELLO, "Names are 1 to 16 letters, digits, spaces, ' and -.")
		return false
	}
	s.id = s.deps.NextID()
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.HandshakeTimeout)
	defer cancel()
	_, err := s.deps.World.Join(ctx, zone.JoinRequest{ID: s.id, Name: name, Out: s, Inputs: s.inputs})
	switch {
	case errors.Is(err, zone.ErrFull):
		s.close(pb.DisconnectReason_DISCONNECT_REASON_SERVER_FULL, "The world is full. Try again soon.")
		return false
	case errors.Is(err, zone.ErrStopped):
		s.close(pb.DisconnectReason_DISCONNECT_REASON_SHUTTING_DOWN, "The server is restarting.")
		return false
	case err != nil:
		// The join may still land in the zone; Leave below cleans it up.
		s.joined.Store(true)
		s.close(pb.DisconnectReason_DISCONNECT_REASON_TIMED_OUT, "Couldn't enter the world in time.")
		return false
	}
	s.joined.Store(true)
	s.log.Debug("player joined", "player", s.id, "name", name, "client", h.GetClientVersion())
	if s.deps.Joined != nil {
		s.deps.Joined(s.id)
	}
	return true
}

func (s *Session) pong(p *pb.Ping) {
	b, _ := wire.Encode(nil, &pb.Envelope{Payload: &pb.Envelope_Pong{Pong: &pb.Pong{
		Nonce: p.GetNonce(), PingSentAtMs: p.GetSentAtMs(), ReceivedAtMs: time.Now().UnixMilli(),
	}}})
	s.Send(wire.Outgoing{Bytes: b})
}

// ValidName checks a display name: 1 to 16 runes of letters, digits,
// spaces, apostrophes and hyphens, with no leading, trailing or doubled
// spaces. It returns the name unchanged when valid.
func ValidName(name string) (string, bool) {
	if !utf8.ValidString(name) {
		return "", false
	}
	n := utf8.RuneCountInString(name)
	if n < 1 || n > MaxNameRunes {
		return "", false
	}
	prevSpace := true // forbids a leading space
	for _, r := range name {
		switch {
		case r == ' ':
			if prevSpace {
				return "", false
			}
			prevSpace = true
			continue
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '\'', r == '-':
		default:
			return "", false
		}
		prevSpace = false
	}
	if prevSpace {
		return "", false // trailing space
	}
	return name, true
}

// ---- Writing -----------------------------------------------------------

func (s *Session) writeLoop() {
	ctx := s.writeCtx
	ticker := time.NewTicker(s.cfg.PingInterval)
	defer ticker.Stop()
	var batch []wire.Outgoing
	var buf []byte // grows to the largest message actually sent; quiet connections keep it tiny
	var nonce uint32
	for {
		select {
		case <-s.closing:
			s.finish()
			return
		case <-s.queue.notify:
			batch = s.queue.takeAll(batch[:0])
			for i, o := range batch {
				// Prebuilt messages are complete, immutable bytes: write them
				// as they are. Only Snapshots are assembled, into buf.
				msg := o.Bytes
				if o.Body != nil {
					buf = o.AppendTo(buf[:0])
					msg = buf
				}
				err := s.conn.Write(ctx, websocket.MessageBinary, msg)
				o.Release()
				batch[i] = wire.Outgoing{}
				if err != nil {
					for _, rest := range batch[i+1:] {
						rest.Release()
					}
					s.close(pb.DisconnectReason_DISCONNECT_REASON_UNSPECIFIED, "")
					s.finish()
					return
				}
				s.deps.Metrics.MessagesOut.Inc()
				s.deps.Metrics.BytesOut.Add(uint64(len(msg)))
			}
			if cap(buf) > 16<<10 {
				buf = nil // don't keep a crowd-sized buffer around
			}
			if cap(batch) > 64 {
				batch = nil
			}
		case now := <-ticker.C:
			if idle := now.Sub(time.Unix(0, s.lastRead.Load())); idle > s.cfg.IdleTimeout {
				s.close(pb.DisconnectReason_DISCONNECT_REASON_TIMED_OUT, "No response from your client.")
				continue
			}
			nonce++
			b, _ := wire.Encode(nil, &pb.Envelope{Payload: &pb.Envelope_Ping{Ping: &pb.Ping{Nonce: nonce, SentAtMs: now.UnixMilli()}}})
			s.Send(wire.Outgoing{Bytes: b})
		}
	}
}

// finish sends the Disconnect, if the reason calls for one, and closes
// the socket. Clients that are too slow, silent or abusive get a best
// effort Disconnect and an immediate close, so they can't hold this
// goroutine for the close handshake's timeouts.
func (s *Session) finish() {
	switch s.reason {
	case pb.DisconnectReason_DISCONNECT_REASON_UNSPECIFIED: // the peer closed, or the socket failed
		_ = s.conn.CloseNow()
		return
	case pb.DisconnectReason_DISCONNECT_REASON_TOO_SLOW: // it isn't reading: a Disconnect can't get through
		_ = s.conn.CloseNow()
		return
	}
	b, _ := wire.Encode(nil, &pb.Envelope{Payload: &pb.Envelope_Disconnect{Disconnect: &pb.Disconnect{Reason: s.reason, Text: s.text}}})
	wctx, cancel := context.WithTimeout(context.Background(), time.Second)
	err := s.conn.Write(wctx, websocket.MessageBinary, b)
	cancel()
	code := websocket.StatusCode(CloseCodeBase + int(s.reason))
	switch s.reason {
	case pb.DisconnectReason_DISCONNECT_REASON_SHUTTING_DOWN,
		pb.DisconnectReason_DISCONNECT_REASON_PROTOCOL_MISMATCH,
		pb.DisconnectReason_DISCONNECT_REASON_SERVER_FULL,
		pb.DisconnectReason_DISCONNECT_REASON_KICKED:
		if err == nil {
			// A graceful close for a client that is healthy and will answer.
			// The library bounds the handshake (5 s to write, 5 s to wait).
			_ = s.conn.Close(code, s.reason.String())
			return
		}
	}
	_ = s.conn.CloseNow()
}
