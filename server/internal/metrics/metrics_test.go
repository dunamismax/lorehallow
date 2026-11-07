package metrics

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTextFormat(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("lh_things_total", "Things.")
	g := r.NewGauge("lh_level", "Level.")
	r.NewGaugeFunc("lh_answer", "Answer.", func() float64 { return 42.5 })
	h := r.NewHistogram("lh_tick_seconds", "Ticks.", []float64{0.001, 0.01})
	c.Add(3)
	g.Set(-2)
	h.Observe(500 * time.Microsecond)
	h.Observe(5 * time.Millisecond)
	h.Observe(time.Second)
	var b strings.Builder
	r.WriteText(&b)
	want := `# HELP lh_answer Answer.
# TYPE lh_answer gauge
lh_answer 42.5
# HELP lh_level Level.
# TYPE lh_level gauge
lh_level -2
# HELP lh_things_total Things.
# TYPE lh_things_total counter
lh_things_total 3
# HELP lh_tick_seconds Ticks.
# TYPE lh_tick_seconds histogram
lh_tick_seconds_bucket{le="0.001"} 1
lh_tick_seconds_bucket{le="0.01"} 2
lh_tick_seconds_bucket{le="+Inf"} 3
lh_tick_seconds_sum 1.0055
lh_tick_seconds_count 3
`
	if b.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", b.String(), want)
	}
}

func TestQuantile(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("h", "h", DurationBuckets())
	if h.Quantile(0.99) != 0 {
		t.Fatal("empty histogram")
	}
	for i := range 1000 {
		d := time.Millisecond
		if i >= 990 {
			d = 20 * time.Millisecond
		}
		h.Observe(d)
	}
	p50, p99, p999 := h.Quantile(0.5), h.Quantile(0.99), h.Quantile(0.999)
	if p50 < time.Millisecond || p50 > 1500*time.Microsecond {
		t.Errorf("p50 = %v, want about 1 ms (bucket upper bound)", p50)
	}
	if p99 < time.Millisecond || p99 > 1500*time.Microsecond {
		t.Errorf("p99 = %v, want about 1 ms: 990 of 1000 are 1 ms", p99)
	}
	if p999 != 20*time.Millisecond {
		t.Errorf("p99.9 = %v, want the max, 20 ms", p999)
	}
	if h.Max() != 20*time.Millisecond || h.Count() != 1000 {
		t.Errorf("max %v count %d", h.Max(), h.Count())
	}
}

func TestConcurrentUpdates(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("c", "c")
	h := r.NewHistogram("h", "h", DurationBuckets())
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for i := range 1000 {
				c.Inc()
				h.Observe(time.Duration(i) * time.Microsecond)
			}
		})
	}
	wg.Wait()
	if c.Value() != 8000 || h.Count() != 8000 || h.Max() != 999*time.Microsecond {
		t.Fatalf("counter %d, histogram %d, max %v", c.Value(), h.Count(), h.Max())
	}
}

func TestDuplicateNamePanics(t *testing.T) {
	r := NewRegistry()
	r.NewCounter("x", "x")
	defer func() {
		if recover() == nil {
			t.Fatal("no panic on duplicate name")
		}
	}()
	r.NewGauge("x", "x")
}

func TestObserveDoesNotAllocate(t *testing.T) {
	h := NewRegistry().NewHistogram("h", "h", DurationBuckets())
	if a := testing.AllocsPerRun(100, func() { h.Observe(3 * time.Millisecond) }); a != 0 {
		t.Fatalf("Observe allocated %.1f times", a)
	}
}
