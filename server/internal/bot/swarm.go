package bot

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// SwarmConfig configures many bots.
type SwarmConfig struct {
	URL string
	N   int
	// Ramp spreads the N connects evenly over this long (0: as fast as
	// Concurrency allows: a login storm).
	Ramp time.Duration
	// Concurrency bounds dials in flight (default 64).
	Concurrency int
	Behavior    Behavior
	// Cheat applies to the first CheatN bots.
	Cheat  Cheat
	CheatN int
	// Silent bots open the WebSocket and never say hello (slowloris).
	Silent   int
	TickRate int // default 20
	Seed     uint64
	// PerDriver is how many bots one driver goroutine steps (default 256).
	PerDriver int
	// HTTPClient dials (nil: a client with no connection limits).
	HTTPClient *http.Client
	Stats      *Stats
}

// Swarm is a running group of bots.
type Swarm struct {
	cfg  SwarmConfig
	mu   sync.Mutex
	bots []*Bot
	raw  []*websocket.Conn
	Errs []error
}

// RunSwarm connects the bots, steps them every tick until ctx is
// cancelled, then hangs up. It returns once every bot has closed.
// onReady, if set, is called when every connect has finished.
func RunSwarm(ctx context.Context, cfg SwarmConfig, onReady func(*Swarm)) *Swarm {
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 64
	}
	if cfg.TickRate == 0 {
		cfg.TickRate = 20
	}
	if cfg.PerDriver == 0 {
		cfg.PerDriver = 256
	}
	if cfg.Stats == nil {
		cfg.Stats = &Stats{}
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Transport: &http.Transport{MaxConnsPerHost: 0, DisableKeepAlives: true}}
	}
	s := &Swarm{cfg: cfg}

	// Drivers step their share of the bots every tick.
	var drivers sync.WaitGroup
	adds := make([]chan *Bot, 0)
	nDrivers := max(1, (cfg.N+cfg.PerDriver-1)/cfg.PerDriver)
	for range nDrivers {
		ch := make(chan *Bot, cfg.PerDriver)
		adds = append(adds, ch)
		drivers.Go(func() { s.drive(ctx, ch) })
	}

	// Connect with bounded concurrency, paced by Ramp.
	sem := make(chan struct{}, cfg.Concurrency)
	var dials sync.WaitGroup
	var interval time.Duration
	if cfg.Ramp > 0 && cfg.N > 1 {
		interval = cfg.Ramp / time.Duration(cfg.N)
	}
	start := time.Now()
connect:
	for i := range cfg.N {
		if interval > 0 {
			if wait := time.Until(start.Add(time.Duration(i) * interval)); wait > 0 {
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					break connect
				}
			}
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break connect
		}
		dials.Go(func() {
			defer func() { <-sem }()
			if i < cfg.Silent {
				s.dialSilent(ctx)
				return
			}
			bc := Config{
				URL: cfg.URL, Name: fmt.Sprintf("bot%d", i), Behavior: cfg.Behavior,
				Seed: cfg.Seed + uint64(i), HTTPClient: cfg.HTTPClient, Stats: cfg.Stats,
			}
			if i-cfg.Silent < cfg.CheatN {
				bc.Cheat = cfg.Cheat
			}
			dctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			b, err := Dial(dctx, bc)
			if err != nil {
				s.mu.Lock()
				s.Errs = append(s.Errs, err)
				s.mu.Unlock()
				return
			}
			s.mu.Lock()
			s.bots = append(s.bots, b)
			s.mu.Unlock()
			adds[i%nDrivers] <- b
		})
	}
	dials.Wait()
	if onReady != nil {
		onReady(s)
	}
	<-ctx.Done()
	drivers.Wait()
	s.mu.Lock()
	bots, raw := s.bots, s.raw
	s.mu.Unlock()
	var closing sync.WaitGroup
	for _, b := range bots {
		closing.Go(b.CloseNow)
	}
	for _, c := range raw {
		_ = c.CloseNow()
	}
	closing.Wait()
	return s
}

// dialSilent opens a WebSocket and never says anything.
func (s *Swarm) dialSilent(ctx context.Context) {
	c, _, err := websocket.Dial(ctx, s.cfg.URL, &websocket.DialOptions{HTTPClient: s.cfg.HTTPClient})
	if err != nil {
		s.mu.Lock()
		s.Errs = append(s.Errs, err)
		s.mu.Unlock()
		return
	}
	s.mu.Lock()
	s.raw = append(s.raw, c)
	s.mu.Unlock()
}

func (s *Swarm) drive(ctx context.Context, adds <-chan *Bot) {
	ticker := time.NewTicker(time.Second / time.Duration(s.cfg.TickRate))
	defer ticker.Stop()
	var mine []*Bot
	for {
		select {
		case <-ctx.Done():
			return
		case b := <-adds:
			mine = append(mine, b)
		case <-ticker.C:
			live := mine[:0]
			for _, b := range mine {
				select {
				case <-b.Done():
					continue
				default:
				}
				_ = b.Step(ctx)
				live = append(live, b)
			}
			mine = live
		}
	}
}

// Bots returns the connected bots.
func (s *Swarm) Bots() []*Bot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*Bot(nil), s.bots...)
}
