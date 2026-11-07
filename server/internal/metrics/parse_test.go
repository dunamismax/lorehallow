package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestParseRoundTrip(t *testing.T) {
	r := NewRegistry()
	r.NewCounter("c_total", "c").Add(7)
	r.NewGauge("g", "g").Set(-3)
	h := r.NewHistogram("h_seconds", "h", []float64{0.001, 0.01, 0.1})
	var b strings.Builder
	r.WriteText(&b)
	before, err := ParseText(strings.NewReader(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	for range 98 {
		h.Observe(500 * time.Microsecond)
	}
	h.Observe(5 * time.Millisecond)
	h.Observe(50 * time.Millisecond)
	b.Reset()
	r.WriteText(&b)
	after, err := ParseText(strings.NewReader(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	if after.Values["c_total"] != 7 || after.Values["g"] != -3 || after.Values["h_seconds_count"] != 100 {
		t.Fatalf("values %v", after.Values)
	}
	d := after.Histograms["h_seconds"].Sub(before.Histograms["h_seconds"])
	if d.Count() != 100 {
		t.Fatalf("count %v", d.Count())
	}
	if p := d.Quantile(0.5); p != time.Millisecond {
		t.Errorf("p50 %v", p)
	}
	if p := d.Quantile(0.99); p != 10*time.Millisecond {
		t.Errorf("p99 %v", p)
	}
	if p := d.Highest(); p != 100*time.Millisecond {
		t.Errorf("highest %v", p)
	}
	if _, err := ParseText(strings.NewReader("broken")); err == nil {
		t.Fatal("bad input accepted")
	}
}
