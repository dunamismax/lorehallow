// Command lorehallow-bot runs headless bots against a Lorehallow server:
// one for a quick check, thousands for a load test.
//
//	lorehallow-bot -url ws://127.0.0.1:7420/ws -n 100 -behavior wander -duration 1m
//
// It prints a summary when it stops, including its own CPU and memory,
// so a load test can show the generator wasn't the bottleneck. With
// -json the summary is one JSON object (used by lorehallow-loadtest).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/dunamismax/lorehallow/server/internal/bot"
	"github.com/dunamismax/lorehallow/server/internal/procstat"
)

// Summary is what one run reports.
type Summary struct {
	Bots          int            `json:"bots"`
	Joined        int64          `json:"joined"`
	DialFailed    int64          `json:"dial_failed"`
	Closed        int64          `json:"closed"`
	Reasons       map[string]int `json:"disconnect_reasons"`
	Snapshots     int64          `json:"snapshots"`
	MessagesIn    int64          `json:"messages_in"`
	BytesIn       int64          `json:"bytes_in"`
	MessagesOut   int64          `json:"messages_out"`
	BytesOut      int64          `json:"bytes_out"`
	MeanJoinMs    float64        `json:"mean_join_ms"`
	MaxJoinMs     float64        `json:"max_join_ms"`
	Seconds       float64        `json:"seconds"`
	CPUSeconds    float64        `json:"generator_cpu_seconds"`
	PeakRSSBytes  int64          `json:"generator_peak_rss_bytes"`
	Goroutines    int            `json:"generator_goroutines_at_end"`
	FirstDialErrs []string       `json:"first_dial_errors,omitempty"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "lorehallow-bot:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("lorehallow-bot", flag.ContinueOnError)
	url := fs.String("url", "ws://127.0.0.1:7420/ws", "server WebSocket URL")
	n := fs.Int("n", 1, "number of bots")
	behavior := fs.String("behavior", "wander", "wander, idle, still or follow")
	cheat := fs.String("cheat", "none", "cheat mode for -cheat-n bots: flood, forge-seq, oversize, unknown, bad-values")
	cheatN := fs.Int("cheat-n", 0, "how many bots cheat")
	silent := fs.Int("silent", 0, "how many bots connect and never say hello (slowloris)")
	ramp := fs.Duration("ramp", 0, "spread connects over this long (0: as fast as possible)")
	conc := fs.Int("concurrency", 64, "dials in flight")
	duration := fs.Duration("duration", 0, "stop after this long (0: until interrupted)")
	seed := fs.Uint64("seed", 1, "random seed")
	asJSON := fs.Bool("json", false, "print the summary as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	b, err := bot.ParseBehavior(*behavior)
	if err != nil {
		return err
	}
	c, err := bot.ParseCheat(*cheat)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

	stats := &bot.Stats{}
	cpu0 := procstat.SelfCPU()
	start := time.Now()
	sampler := procstat.NewPeakRSS(os.Getpid(), time.Second)
	sw := bot.RunSwarm(ctx, bot.SwarmConfig{
		URL: *url, N: *n, Ramp: *ramp, Concurrency: *conc, Behavior: b, Cheat: c, CheatN: *cheatN,
		Silent: *silent, Seed: *seed, Stats: stats,
	}, func(sw *bot.Swarm) {
		fmt.Fprintf(os.Stderr, "lorehallow-bot: %d connected, %d failed in %v\n", len(sw.Bots()), len(sw.Errs), time.Since(start).Round(time.Millisecond))
	})
	peak := sampler.Stop()

	s := Summary{
		Bots: *n, Joined: stats.Joined.Load(), DialFailed: stats.DialFailed.Load(), Closed: stats.Closed.Load(),
		Reasons: map[string]int{}, Snapshots: stats.Snapshots.Load(),
		MessagesIn: stats.MessagesIn.Load(), BytesIn: stats.BytesIn.Load(),
		MessagesOut: stats.MessagesOut.Load(), BytesOut: stats.BytesOut.Load(),
		Seconds: time.Since(start).Seconds(), CPUSeconds: (procstat.SelfCPU() - cpu0).Seconds(),
		PeakRSSBytes: peak, Goroutines: runtime.NumGoroutine(),
	}
	for r, k := range stats.ReasonCounts() {
		s.Reasons[r.String()] = k
	}
	if j := stats.Joined.Load(); j > 0 {
		s.MeanJoinMs = float64(stats.JoinNanos.Load()) / float64(j) / 1e6
		s.MaxJoinMs = float64(stats.MaxJoinNanos.Load()) / 1e6
	}
	for i, e := range sw.Errs {
		if i == 3 {
			break
		}
		s.FirstDialErrs = append(s.FirstDialErrs, e.Error())
	}
	if *asJSON {
		return json.NewEncoder(out).Encode(s)
	}
	_, err = fmt.Fprintf(out, "bots %d, joined %d, failed %d, snapshots %d, in %.1f KB/s, out %.1f KB/s, mean join %.1f ms, generator CPU %.1f s, peak RSS %.1f MB\n",
		s.Bots, s.Joined, s.DialFailed, s.Snapshots, float64(s.BytesIn)/s.Seconds/1e3, float64(s.BytesOut)/s.Seconds/1e3,
		s.MeanJoinMs, s.CPUSeconds, float64(s.PeakRSSBytes)/1e6)
	return err
}
