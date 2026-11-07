// Command lorehallow-loadtest runs one scenario from docs/design/scaling.md
// against a real server and writes a draft result to docs/perf/.
//
//	lorehallow-loadtest -scenario idle -n 100 -duration 2m
//
// It starts bin/lorehallow-server as a separate process (so the server's
// memory and CPU are its own), connects an in-process bot swarm, waits a
// warm-up, then measures over a window: it scrapes the server's /metrics
// at the start and end of the window and samples the server's RSS every
// second. After the window it disconnects every bot and checks the server
// returns to its baseline. The draft names the hardware it ran on and
// marks each scaling.md target pass or fail; an agent checks the numbers
// and fills in the notes before committing it.
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/dunamismax/lorehallow/server/internal/bot"
	"github.com/dunamismax/lorehallow/server/internal/metrics"
	"github.com/dunamismax/lorehallow/server/internal/procstat"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "lorehallow-loadtest:", err)
		os.Exit(1)
	}
}

type options struct {
	scenario       string
	n              int
	duration       time.Duration
	warmup         time.Duration
	ramp           time.Duration
	serverBin, mp  string
	outDir, commit string
	hardware       string
	protocol       uint
}

func run(args []string) error {
	fs := flag.NewFlagSet("lorehallow-loadtest", flag.ContinueOnError)
	var o options
	fs.StringVar(&o.scenario, "scenario", "idle", "scenario: idle or spread (more in Phase 3)")
	fs.IntVar(&o.n, "n", 100, "bots")
	fs.DurationVar(&o.duration, "duration", 2*time.Minute, "measured window")
	fs.DurationVar(&o.warmup, "warmup", 20*time.Second, "time after the last connect before measuring")
	fs.DurationVar(&o.ramp, "ramp", 0, "spread connects over this long (default: 10 ms per bot)")
	fs.StringVar(&o.serverBin, "server", "bin/lorehallow-server", "server binary")
	fs.StringVar(&o.mp, "map", "content/maps/starter.tmj", "map")
	fs.StringVar(&o.outDir, "out", "docs/perf", "directory for the draft result")
	fs.StringVar(&o.commit, "commit", "unknown", "git commit being measured")
	fs.StringVar(&o.hardware, "hardware", "", "hardware name and description (default: this machine as dev-m5 if it matches)")
	fs.UintVar(&o.protocol, "protocol-version", 0, "protocol version the bots announce (0: current; set it to measure an older server build)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var behavior bot.Behavior
	switch o.scenario {
	case "idle":
		behavior = bot.Idle
	case "spread":
		behavior = bot.Wander
	default:
		return fmt.Errorf("scenario %q isn't implemented yet (idle, spread)", o.scenario)
	}
	if o.ramp == 0 {
		o.ramp = time.Duration(o.n) * 10 * time.Millisecond
	}

	// Start the server on free ports, with limits that fit the test.
	game, admin := freePort(), freePort()
	srv := exec.Command(o.serverBin, "-addr", game, "-admin-addr", admin, "-map", o.mp,
		"-max-connections", fmt.Sprint(o.n+100), "-max-pending-per-ip", fmt.Sprint(o.n+100), "-log-level", "warn")
	srv.Stdout, srv.Stderr = os.Stderr, os.Stderr
	if err := srv.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}
	defer func() { _ = srv.Process.Kill(); _ = srv.Wait() }()
	if err := waitHealthy("http://"+game+"/healthz", 10*time.Second); err != nil {
		return err
	}
	pid := srv.Process.Pid
	scrape := func() (metrics.Scrape, error) { return scrapeMetrics("http://" + admin + "/metrics") }

	settle(3 * time.Second)
	base, err := scrape()
	if err != nil {
		return err
	}
	baseRSS, err := procstat.RSS(pid)
	if err != nil {
		return err
	}

	// Connect the swarm.
	ctx, stopSwarm := context.WithCancel(context.Background())
	defer stopSwarm()
	stats := &bot.Stats{}
	ready := make(chan *bot.Swarm, 1)
	swarmDone := make(chan *bot.Swarm, 1)
	genCPU0 := procstat.SelfCPU()
	genPeak := procstat.NewPeakRSS(os.Getpid(), time.Second)
	connectStart := time.Now()
	go func() {
		swarmDone <- bot.RunSwarm(ctx, bot.SwarmConfig{URL: "ws://" + game + "/ws", N: o.n, Ramp: o.ramp, Behavior: behavior, Seed: 1, Stats: stats, ProtocolVersion: uint32(o.protocol)},
			func(s *bot.Swarm) { ready <- s })
	}()
	sw := <-ready
	connectTook := time.Since(connectStart)
	fmt.Fprintf(os.Stderr, "loadtest: %d of %d bots connected in %v; warming up %v\n", len(sw.Bots()), o.n, connectTook.Round(time.Millisecond), o.warmup)
	if len(sw.Bots()) != o.n {
		stopSwarm()
		<-swarmDone
		return fmt.Errorf("only %d of %d bots connected; first errors: %v", len(sw.Bots()), o.n, sw.Errs[:min(3, len(sw.Errs))])
	}
	settle(o.warmup)

	// Measure.
	fmt.Fprintf(os.Stderr, "loadtest: measuring for %v\n", o.duration)
	start, err := scrape()
	if err != nil {
		return err
	}
	t0 := time.Now()
	in0, out0 := stats.BytesIn.Load(), stats.BytesOut.Load()
	var rssSum, rssMax int64
	var rssN int64
	tick := time.NewTicker(time.Second)
	for time.Since(t0) < o.duration {
		<-tick.C
		if v, err := procstat.RSS(pid); err == nil {
			rssSum += v
			rssN++
			rssMax = max(rssMax, v)
		}
	}
	tick.Stop()
	end, err := scrape()
	if err != nil {
		return err
	}
	window := time.Since(t0).Seconds()
	genIn, genOut := stats.BytesIn.Load()-in0, stats.BytesOut.Load()-out0
	stillOn := 0
	for _, b := range sw.Bots() {
		select {
		case <-b.Done():
		default:
			stillOn++
		}
	}

	// Disconnect everyone and check the server returns to baseline.
	stopSwarm()
	<-swarmDone
	genCPU := (procstat.SelfCPU() - genCPU0).Seconds()
	genRSS := genPeak.Stop()
	var after metrics.Scrape
	backAt := time.Duration(-1)
	leaveStart := time.Now()
	for time.Since(leaveStart) < 15*time.Second {
		after, err = scrape()
		if err == nil && after.Values["lorehallow_connections_open"] == 0 &&
			after.Values["lorehallow_goroutines"] <= base.Values["lorehallow_goroutines"]*1.01+1 {
			backAt = time.Since(leaveStart)
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	r := result{o: o, window: window, connectTook: connectTook, n: float64(o.n), stillOn: stillOn,
		base: base, start: start, end: end, after: after, backAt: backAt,
		baseRSS: baseRSS, meanRSS: rssSum / max(rssN, 1), maxRSS: rssMax,
		genCPU: genCPU, genRSS: genRSS, genIn: genIn, genOut: genOut, stats: stats}
	path, err := r.write()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "loadtest: wrote %s\n", path)
	_ = srv.Process.Signal(syscall.SIGTERM)
	return nil
}

// settle waits without spinning. (This is a load-test driver, not a test:
// it really does need wall-clock time to pass.)
func settle(d time.Duration) { time.Sleep(d) }

func freePort() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().String()
}

func waitHealthy(url string, within time.Duration) error {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(url); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server not healthy at %s within %v", url, within)
}

func scrapeMetrics(url string) (metrics.Scrape, error) {
	resp, err := http.Get(url)
	if err != nil {
		return metrics.Scrape{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	return metrics.ParseText(resp.Body)
}

type result struct {
	o                        options
	window                   float64
	connectTook              time.Duration
	n                        float64
	stillOn                  int
	base, start, end, after  metrics.Scrape
	backAt                   time.Duration
	baseRSS, meanRSS, maxRSS int64
	genCPU                   float64
	genRSS                   int64
	genIn, genOut            int64
	stats                    *bot.Stats
}

func (r *result) delta(name string) float64 { return r.end.Values[name] - r.start.Values[name] }

func tierOf(n int) int {
	switch {
	case n <= 100:
		return 1
	case n <= 1000:
		return 2
	case n <= 5000:
		return 3
	}
	return 4
}

type row struct {
	metric, target, measured string
	pass                     string // yes, no, or "n/a" (recorded, not held)
}

func check(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}

func ms(d time.Duration) string {
	if d == time.Duration(math.MaxInt64) {
		return "over the largest bucket"
	}
	return fmt.Sprintf("≤ %.2f ms", float64(d)/1e6)
}

func (r *result) rows() []row {
	tier := tierOf(r.o.n)
	tick := r.end.Histograms["lorehallow_zone_tick_seconds"].Sub(r.start.Histograms["lorehallow_zone_tick_seconds"])
	rtt := r.end.Histograms["lorehallow_heartbeat_rtt_seconds"].Sub(r.start.Histograms["lorehallow_heartbeat_rtt_seconds"])
	cpuCores := r.delta("lorehallow_process_cpu_seconds_total") / r.window
	outPerPlayer := r.delta("lorehallow_bytes_out_total") / r.window / r.n
	msgsOutPerPlayer := r.delta("lorehallow_messages_out_total") / r.window / r.n
	inPerPlayer := r.delta("lorehallow_bytes_in_total") / r.window / r.n
	memPerConn := float64(r.meanRSS-r.baseRSS) / r.n
	heapPerConn := ((r.end.Values["lorehallow_heap_inuse_bytes"] + r.end.Values["lorehallow_stack_inuse_bytes"]) -
		(r.base.Values["lorehallow_heap_inuse_bytes"] + r.base.Values["lorehallow_stack_inuse_bytes"])) / r.n
	gor := r.end.Values["lorehallow_goroutines"] - r.base.Values["lorehallow_goroutines"]
	mallocsPerTick := r.delta("lorehallow_mallocs_total") / max(tick.Count(), 1)
	gcFrac := r.end.Values["lorehallow_gc_cpu_fraction"]
	overruns := r.delta("lorehallow_zone_tick_overruns_total")
	evicted := r.delta("lorehallow_connections_evicted_total")
	back := "never (within 15 s)"
	if r.backAt >= 0 {
		back = fmt.Sprintf("%.1f s", r.backAt.Seconds())
	}
	common := []row{
		{"Players connected at the end of the window", fmt.Sprintf("%d", r.o.n), fmt.Sprintf("%d", r.stillOn), check(r.stillOn == r.o.n)},
		{"Zone tick p50", "(recorded)", ms(tick.Quantile(0.5)), "n/a"},
		{"Ticks over the 50 ms budget", "(recorded)", fmt.Sprintf("%.0f of %.0f", overruns, tick.Count()), "n/a"},
		{"Evicted as too slow", "0", fmt.Sprintf("%.0f", evicted), check(evicted == 0)},
		{"Server goroutines above baseline", fmt.Sprintf("≤ 2N = %d", 2*r.o.n), fmt.Sprintf("%.0f (%.2f per connection)", gor, gor/r.n), check(gor <= 2*r.n)},
		{"Server memory per connection, RSS ((mean RSS − baseline) / N)", "(see below)", fmt.Sprintf("%.1f KiB", memPerConn/1024), "n/a"},
		{"Server heap + stacks per connection", "(recorded)", fmt.Sprintf("%.1f KiB", heapPerConn/1024), "n/a"},
		{"Server allocations per tick (whole process)", "(recorded)", fmt.Sprintf("%.0f", mallocsPerTick), "n/a"},
		{"Server CPU", "(see below)", fmt.Sprintf("%.3f cores", cpuCores), "n/a"},
		{"GC CPU fraction (since start)", "≤ 5%", fmt.Sprintf("%.2f%%", gcFrac*100), check(gcFrac <= 0.05)},
		{"Outbound per player (payload)", "(see below)", fmt.Sprintf("%.0f B/s (%.1f messages/s)", outPerPlayer, msgsOutPerPlayer), "n/a"},
		{"Inbound per player (payload)", "(see below)", fmt.Sprintf("%.0f B/s", inPerPlayer), "n/a"},
		{"Back to baseline after everyone left", "≤ 10 s", back, check(r.backAt >= 0 && r.backAt <= 10*time.Second)},
	}
	var out []row
	switch r.o.scenario {
	case "idle":
		memT := 48.0
		cpuT := map[int]float64{1: 0.02, 2: 0.10, 3: 0.50, 4: 1}[tier]
		tickT := map[int]time.Duration{1: 2 * time.Millisecond, 2: 5 * time.Millisecond, 3: 12500 * time.Microsecond, 4: 12500 * time.Microsecond}[tier]
		rttT := map[int]time.Duration{1: 50 * time.Millisecond, 2: 50 * time.Millisecond, 3: 100 * time.Millisecond, 4: 200 * time.Millisecond}[tier]
		if tier == 1 {
			memT = 64
		}
		both := outPerPlayer + inPerPlayer
		out = []row{
			{"Server memory per idle connection (RSS)", fmt.Sprintf("≤ %.0f KiB", memT), fmt.Sprintf("%.1f KiB", memPerConn/1024), check(memPerConn/1024 <= memT)},
			{"Server CPU at steady state", fmt.Sprintf("≤ %.0f%% of one core", cpuT*100), fmt.Sprintf("%.1f%% of one core", cpuCores*100), check(cpuCores <= cpuT)},
			{"Idle bandwidth per connection, both directions", "≤ 200 B/s", fmt.Sprintf("%.0f B/s", both), check(both <= 200)},
			{"Heartbeat round trip p99, loopback", fmt.Sprintf("≤ %v", rttT), ms(rtt.Quantile(0.99)), check(rtt.Count() > 0 && rtt.Quantile(0.99) <= rttT)},
			{"Zone tick p99 with everyone idle", fmt.Sprintf("≤ %v", tickT), ms(tick.Quantile(0.99)), check(tick.Quantile(0.99) <= tickT)},
		}
	case "spread":
		cpuT := map[int]float64{1: 1, 2: 4}[tier]
		out = []row{
			{"Zone tick p99", "≤ 12.5 ms", ms(tick.Quantile(0.99)), check(tick.Quantile(0.99) <= 12500*time.Microsecond)},
			{"Zone tick max over the window", "≤ 50 ms", ms(tick.Highest()), check(tick.Highest() <= 50*time.Millisecond)},
			{"Outbound per player (not held until Phase 3's delta snapshots)", "≤ 8 KB/s", fmt.Sprintf("%.0f B/s", outPerPlayer), "n/a"},
			{"Inbound per player", "≤ 1 KB/s", fmt.Sprintf("%.0f B/s", inPerPlayer), check(inPerPlayer <= 1000)},
		}
		if cpuT > 0 {
			out = append(out, row{"Total server CPU", fmt.Sprintf("≤ %.0f cores", cpuT), fmt.Sprintf("%.2f cores", cpuCores), check(cpuCores <= cpuT)})
		}
	}
	return append(out, common...)
}

func (r *result) write() (string, error) {
	tier := tierOf(r.o.n)
	date := time.Now().Format("2006-01-02")
	path := filepath.Join(r.o.outDir, fmt.Sprintf("%s-%s-t%d.md", date, r.o.scenario, tier))
	for i := 2; fileExists(path); i++ {
		path = filepath.Join(r.o.outDir, fmt.Sprintf("%s-%s-t%d-%d.md", date, r.o.scenario, tier, i))
	}
	hw := r.o.hardware
	if hw == "" {
		hw = "`dev-m5` (MacBook Pro Mac17,2, Apple M5, 10 cores (4 performance, 6 efficiency), 32 GB, macOS 27.2): check this matches the machine"
	}
	rows := r.rows()
	pass := true
	for _, x := range rows {
		if x.pass == "no" {
			pass = false
		}
	}
	verdict := "every held target passed"
	if !pass {
		verdict = "at least one target failed"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s t%d: %d bots, %s\n\n", r.o.scenario, tier, r.o.n, verdict)
	fmt.Fprintf(&b, "DRAFT written by `make loadtest`. Check the numbers, fill in the notes, then commit.\n\n")
	fmt.Fprintf(&b, "- **Date:** %s\n- **Commit:** %s\n- **Hardware:** %s\n", date, r.o.commit, hw)
	fmt.Fprintf(&b, "- **Go:** %s, default GOGC and GOMEMLIMIT, GOMAXPROCS=%d\n", runtime.Version(), runtime.GOMAXPROCS(0))
	fmt.Fprintf(&b, "- **Generator:** `lorehallow-loadtest` in-process swarm on the same machine as the server; generator CPU %.1f s over the whole run (%.2f cores averaged over connect, warm-up and window), peak RSS %.1f MB\n",
		r.genCPU, r.genCPU/(r.connectTook.Seconds()+r.o.warmup.Seconds()+r.window), float64(r.genRSS)/1e6)
	fmt.Fprintf(&b, "- **Command:** `make loadtest SCENARIO=%s N=%d DURATION=%s`\n", r.o.scenario, r.o.n, r.o.duration)
	fmt.Fprintf(&b, "- **Duration:** connect %.1f s (ramp %s), warm-up %s, measured window %.1f s\n- **Map:** `%s` (one zone; Phase 2 sends every entity to every player)\n\n",
		r.connectTook.Seconds(), r.o.ramp, r.o.warmup, r.window, r.o.mp)
	fmt.Fprintf(&b, "## Results\n\nTargets from [scaling.md](../design/scaling.md) for this case and tier. \"n/a\" rows are recorded, not held. Tick times are bucket upper bounds (an over-estimate by at most ×1.41).\n\n")
	fmt.Fprintf(&b, "| Metric | Target | Measured | Pass |\n|---|---|---|---|\n")
	for _, x := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", x.metric, x.target, x.measured, x.pass)
	}
	fmt.Fprintf(&b, "\nServer RSS: baseline %.1f MB, mean %.1f MB and peak %.1f MB during the window. Generator saw %.0f B/s in and %.0f B/s out per bot.\n\n",
		float64(r.baseRSS)/1e6, float64(r.meanRSS)/1e6, float64(r.maxRSS)/1e6, float64(r.genIn)/r.window/r.n, float64(r.genOut)/r.window/r.n)
	fmt.Fprintf(&b, "## Notes\n\n(Fill in: what limited the run, anything surprising, what to try next.)\n")
	if err := os.MkdirAll(r.o.outDir, 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte(b.String()), 0o644)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
