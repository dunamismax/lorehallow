// Package metrics is a small, dependency-free metrics registry that writes
// the Prometheus text exposition format. Every metric is lock-free to
// update, so the zone tick and connection goroutines can record without
// contending.
//
// It covers what the server needs (counters, gauges, histograms and
// callback gauges) and nothing else; a full client library would be a
// dependency for features we don't use.
package metrics

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Registry holds named metrics and writes them in Prometheus text format.
type Registry struct {
	mu      sync.Mutex
	metrics map[string]metric
}

type metric interface {
	write(w io.Writer, name string)
	help() string
	kind() string
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{metrics: map[string]metric{}} }

func (r *Registry) add(name string, m metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.metrics[name]; dup {
		panic("metrics: duplicate metric " + name)
	}
	r.metrics[name] = m
}

// WriteText writes every metric, sorted by name.
func (r *Registry) WriteText(w io.Writer) {
	r.mu.Lock()
	names := make([]string, 0, len(r.metrics))
	for n := range r.metrics {
		names = append(names, n)
	}
	ms := make(map[string]metric, len(r.metrics))
	for n, m := range r.metrics {
		ms[n] = m
	}
	r.mu.Unlock()
	sort.Strings(names)
	for _, n := range names {
		m := ms[n]
		_, _ = fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", n, m.help(), n, m.kind())
		m.write(w, n)
	}
}

// Counter only goes up.
type Counter struct {
	v atomic.Uint64
	h string
}

// NewCounter registers a counter.
func (r *Registry) NewCounter(name, help string) *Counter {
	c := &Counter{h: help}
	r.add(name, c)
	return c
}

// Inc adds one.
func (c *Counter) Inc() { c.v.Add(1) }

// Add adds n.
func (c *Counter) Add(n uint64) { c.v.Add(n) }

// Value returns the current count.
func (c *Counter) Value() uint64 { return c.v.Load() }

func (c *Counter) help() string { return c.h }
func (c *Counter) kind() string { return "counter" }
func (c *Counter) write(w io.Writer, n string) {
	_, _ = fmt.Fprintf(w, "%s %d\n", n, c.v.Load())
}

// Gauge goes up and down.
type Gauge struct {
	v atomic.Int64
	h string
}

// NewGauge registers a gauge.
func (r *Registry) NewGauge(name, help string) *Gauge {
	g := &Gauge{h: help}
	r.add(name, g)
	return g
}

// Add changes the gauge by d.
func (g *Gauge) Add(d int64) { g.v.Add(d) }

// Set sets the gauge.
func (g *Gauge) Set(v int64) { g.v.Store(v) }

// Value returns the current value.
func (g *Gauge) Value() int64 { return g.v.Load() }

func (g *Gauge) help() string { return g.h }
func (g *Gauge) kind() string { return "gauge" }
func (g *Gauge) write(w io.Writer, n string) {
	_, _ = fmt.Fprintf(w, "%s %d\n", n, g.v.Load())
}

// GaugeFunc reports a value computed when scraped.
type GaugeFunc struct {
	f func() float64
	h string
}

// NewGaugeFunc registers a gauge whose value comes from f at scrape time.
func (r *Registry) NewGaugeFunc(name, help string, f func() float64) {
	r.add(name, &GaugeFunc{f: f, h: help})
}

func (g *GaugeFunc) help() string { return g.h }
func (g *GaugeFunc) kind() string { return "gauge" }
func (g *GaugeFunc) write(w io.Writer, n string) {
	_, _ = fmt.Fprintf(w, "%s %s\n", n, formatFloat(g.f()))
}

// Histogram counts observations into fixed buckets.
type Histogram struct {
	bounds []float64 // upper bounds, ascending; +Inf is implicit
	counts []atomic.Uint64
	count  atomic.Uint64
	sumNs  atomic.Int64 // sum of observed durations in nanoseconds
	maxNs  atomic.Int64
	h      string
}

// DurationBuckets are exponential bucket bounds in seconds from 50 µs to
// about 0.8 s, fine enough to tell a 12.5 ms p99 from a 25 ms one.
func DurationBuckets() []float64 {
	b := []float64{}
	for v := 50e-6; v < 1; v *= math.Sqrt2 {
		b = append(b, v)
	}
	return b
}

// NewHistogram registers a histogram with the given bucket upper bounds
// in seconds.
func (r *Registry) NewHistogram(name, help string, bounds []float64) *Histogram {
	h := &Histogram{bounds: bounds, counts: make([]atomic.Uint64, len(bounds)+1), h: help}
	r.add(name, h)
	return h
}

// Observe records one duration.
func (h *Histogram) Observe(d time.Duration) {
	s := d.Seconds()
	i := sort.SearchFloat64s(h.bounds, s)
	h.counts[i].Add(1)
	h.count.Add(1)
	h.sumNs.Add(int64(d))
	for {
		m := h.maxNs.Load()
		if int64(d) <= m || h.maxNs.CompareAndSwap(m, int64(d)) {
			break
		}
	}
}

// Count returns how many durations were observed.
func (h *Histogram) Count() uint64 { return h.count.Load() }

// Max returns the largest observed duration.
func (h *Histogram) Max() time.Duration { return time.Duration(h.maxNs.Load()) }

// Quantile estimates the q-quantile (0 < q < 1) as the upper bound of the
// bucket it falls in, so it never under-reports. It returns 0 with no
// observations, and Max when q lands in the overflow bucket.
func (h *Histogram) Quantile(q float64) time.Duration {
	total := h.count.Load()
	if total == 0 {
		return 0
	}
	rank := uint64(math.Ceil(q * float64(total)))
	var seen uint64
	for i := range h.counts {
		seen += h.counts[i].Load()
		if seen >= rank {
			if i == len(h.bounds) {
				return h.Max()
			}
			return min(time.Duration(h.bounds[i]*float64(time.Second)), h.Max())
		}
	}
	return h.Max()
}

func (h *Histogram) help() string { return h.h }
func (h *Histogram) kind() string { return "histogram" }
func (h *Histogram) write(w io.Writer, n string) {
	var cum uint64
	for i, b := range h.bounds {
		cum += h.counts[i].Load()
		_, _ = fmt.Fprintf(w, "%s_bucket{le=\"%s\"} %d\n", n, formatFloat(b), cum)
	}
	cum += h.counts[len(h.bounds)].Load()
	_, _ = fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n", n, cum)
	_, _ = fmt.Fprintf(w, "%s_sum %s\n%s_count %d\n", n, formatFloat(time.Duration(h.sumNs.Load()).Seconds()), n, cum)
}

func formatFloat(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
