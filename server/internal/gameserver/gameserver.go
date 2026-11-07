// Package gameserver connects the pieces: the /ws endpoint accepts
// WebSocket connections, enforces the connection limits, runs a
// session.Session per connection, and owns the zone those sessions join.
//
// Shutdown is ordered so nobody is dropped silently: stop accepting, tell
// every session SHUTTING_DOWN and wait for them to finish (each one
// leaves the zone on its way out), then stop the zone.
package gameserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/dunamismax/lorehallow/server/internal/mapdata"
	"github.com/dunamismax/lorehallow/server/internal/metrics"
	"github.com/dunamismax/lorehallow/server/internal/pb"
	"github.com/dunamismax/lorehallow/server/internal/session"
	"github.com/dunamismax/lorehallow/server/internal/wire"
	"github.com/dunamismax/lorehallow/server/internal/zone"
)

// Config configures the game server.
type Config struct {
	Map *mapdata.Map
	// MaxConnections caps open game connections. Beyond it, clients get
	// Disconnect SERVER_FULL; well beyond it, a plain HTTP 503.
	MaxConnections int
	// MaxPendingPerIP caps connections from one IP that haven't finished
	// the handshake (slowloris protection).
	MaxPendingPerIP int
	// OriginPatterns lists extra allowed browser origins (coder/websocket
	// patterns). Non-browser clients send no Origin and are always allowed.
	OriginPatterns []string
	// TrustProxy takes the client IP from X-Forwarded-For when the
	// connection comes from loopback (Caddy in front).
	TrustProxy bool
	// DrainTimeout bounds how long shutdown waits for sessions to close.
	DrainTimeout time.Duration
	Session      session.Config
	Zone         zone.Config // Map, Log and Metrics are filled in from here
	Version      string
	Log          *slog.Logger
	Metrics      *metrics.Registry
}

// Defaults.
const (
	DefaultMaxConnections  = 5000
	DefaultMaxPendingPerIP = 16
	DefaultDrainTimeout    = 15 * time.Second
	// hardCapExtra connections past MaxConnections are accepted just to be
	// told SERVER_FULL; past that, a 503 costs no WebSocket at all.
	hardCapExtra = 256
)

// Server is the game server. Create it with New, mount Handler at /ws and
// call Run.
type Server struct {
	cfg  Config
	zone *zone.Zone
	sm   *session.Metrics
	log  *slog.Logger

	nextID atomic.Uint32
	conns  atomic.Int64

	mu       sync.Mutex
	pending  map[string]int
	closed   bool
	sessions sync.WaitGroup
	ctx      context.Context // sessions' context: cancelled to shut down
	cancel   context.CancelFunc

	rejected *metrics.Counter
}

// New builds a server. It doesn't start anything.
func New(cfg Config) *Server {
	if cfg.MaxConnections == 0 {
		cfg.MaxConnections = DefaultMaxConnections
	}
	if cfg.MaxPendingPerIP == 0 {
		cfg.MaxPendingPerIP = DefaultMaxPendingPerIP
	}
	if cfg.DrainTimeout == 0 {
		cfg.DrainTimeout = DefaultDrainTimeout
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	if cfg.Metrics == nil {
		cfg.Metrics = metrics.NewRegistry()
	}
	zc := cfg.Zone
	zc.Map, zc.Log, zc.Metrics, zc.ServerVersion = cfg.Map, cfg.Log, cfg.Metrics, cfg.Version
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		cfg:      cfg,
		zone:     zone.New(zc),
		sm:       session.NewMetrics(cfg.Metrics),
		log:      cfg.Log,
		pending:  map[string]int{},
		ctx:      ctx,
		cancel:   cancel,
		rejected: cfg.Metrics.NewCounter("lorehallow_connections_rejected_total", "Connection attempts refused before a session started (over capacity, too many pending from one IP, shutting down)."),
	}
}

// Online is the number of open game connections.
func (s *Server) Online() int64 { return s.conns.Load() }

// Run runs the zone until ctx is cancelled, then shuts down in order.
func (s *Server) Run(ctx context.Context) {
	zctx, stopZone := context.WithCancel(context.Background())
	zoneDone := make(chan struct{})
	go func() { s.zone.Run(zctx); close(zoneDone) }()

	<-ctx.Done()
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.log.Info("disconnecting players", "online", s.conns.Load())
	s.cancel() // every session sends SHUTTING_DOWN and closes

	drained := make(chan struct{})
	go func() { s.sessions.Wait(); close(drained) }()
	select {
	case <-drained:
	case <-time.After(s.cfg.DrainTimeout):
		s.log.Warn("sessions still open after the drain timeout", "online", s.conns.Load())
	}
	stopZone()
	<-zoneDone
}

// Handler is the /ws endpoint.
//
// Once the WebSocket is accepted, the session runs on a goroutine of its
// own and the HTTP handler returns. That lets net/http's per-connection
// state (the request, its headers, the response, the server's 4 KiB
// buffers and the handler goroutine's deep stack) be collected, instead
// of living as long as the player stays online (ADR 0010). The goroutine
// count per connection is unchanged: the session goroutine reads, and
// session.Serve starts the writer.
func (s *Server) Handler() http.Handler {
	opts := &websocket.AcceptOptions{OriginPatterns: s.cfg.OriginPatterns, CompressionMode: websocket.CompressionDisabled}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, sl := s.admit(w, r, opts)
		if conn == nil {
			return
		}
		go func() {
			defer sl.release()
			session.Serve(s.ctx, conn, s.cfg.Session, session.Deps{
				World:   s.zone,
				NextID:  func() uint32 { return s.nextID.Add(1) },
				Log:     s.log,
				Metrics: s.sm,
				Joined:  func(uint32) { sl.joined() },
			})
		}()
	})
}

// admit applies the connection limits and accepts the WebSocket. On
// success the caller owns the returned slot and must release it when the
// session ends; on failure admit has answered the client and released
// everything itself.
func (s *Server) admit(w http.ResponseWriter, r *http.Request, opts *websocket.AcceptOptions) (*websocket.Conn, *slot) {
	if !s.begin() {
		s.rejected.Inc()
		http.Error(w, "shutting down", http.StatusServiceUnavailable)
		return nil, nil
	}
	sl := &slot{s: s}
	accepted := false
	defer func() {
		if !accepted {
			sl.release()
		}
	}()

	n := s.conns.Add(1)
	if n > int64(s.cfg.MaxConnections+hardCapExtra) {
		s.rejected.Inc()
		http.Error(w, "server full", http.StatusServiceUnavailable)
		return nil, nil
	}
	ip := s.clientIP(r)
	if !s.addPending(ip) {
		s.rejected.Inc()
		http.Error(w, "too many connections", http.StatusTooManyRequests)
		return nil, nil
	}
	sl.ip, sl.pending = ip, true

	conn, err := websocket.Accept(withSmallBuffers(w), r, opts)
	if err != nil {
		s.rejected.Inc()
		return nil, nil // Accept wrote the HTTP error (bad upgrade, bad origin)
	}
	if n > int64(s.cfg.MaxConnections) {
		s.rejected.Inc()
		Reject(conn, pb.DisconnectReason_DISCONNECT_REASON_SERVER_FULL, "The world is full. Try again soon.")
		return nil, nil
	}
	accepted = true
	return conn, sl
}

// slot is one admitted connection's share of the server's limits: a
// place in the session count, the connection count and, until the
// handshake finishes, its IP's pending count.
type slot struct {
	s       *Server
	mu      sync.Mutex
	ip      string
	pending bool
}

// joined frees the pending-handshake place. It's safe to call more than once.
func (sl *slot) joined() {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.pending {
		sl.pending = false
		sl.s.removePending(sl.ip)
	}
}

// release gives everything back. Call it exactly once.
func (sl *slot) release() {
	sl.joined()
	sl.s.conns.Add(-1)
	sl.s.sessions.Done()
}

// begin registers a session unless the server is shutting down.
func (s *Server) begin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.sessions.Add(1)
	return true
}

func (s *Server) addPending(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending[ip] >= s.cfg.MaxPendingPerIP {
		return false
	}
	s.pending[ip]++
	return true
}

func (s *Server) removePending(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending[ip]--; s.pending[ip] <= 0 {
		delete(s.pending, ip)
	}
}

func (s *Server) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if s.cfg.TrustProxy {
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				if last := strings.TrimSpace(parts[len(parts)-1]); net.ParseIP(last) != nil {
					return last
				}
			}
		}
	}
	return host
}

// Reject tells a client why it can't come in, then closes.
func Reject(conn *websocket.Conn, reason pb.DisconnectReason, text string) {
	b, _ := wire.Encode(nil, &pb.Envelope{Payload: &pb.Envelope_Disconnect{Disconnect: &pb.Disconnect{Reason: reason, Text: text}}})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageBinary, b); err != nil && !errors.Is(err, net.ErrClosed) {
		_ = conn.CloseNow()
		return
	}
	_ = conn.Close(websocket.StatusCode(session.CloseCodeBase+int(reason)), reason.String())
}
