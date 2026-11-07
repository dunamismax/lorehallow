package procstat

import (
	"os"
	"testing"
	"time"
)

func TestSelfCPUAndRSS(t *testing.T) {
	before := SelfCPU()
	x := 0
	for i := range 20_000_000 {
		x += i
	}
	_ = x
	if SelfCPU() <= before {
		t.Fatal("CPU time didn't move after busy work")
	}
	rss, err := RSS(os.Getpid())
	if err != nil || rss < 1<<20 {
		t.Fatalf("RSS = %d, %v", rss, err)
	}
	p := NewPeakRSS(os.Getpid(), time.Hour)
	if peak := p.Stop(); peak < 1<<20 {
		t.Fatalf("peak = %d", peak)
	}
}
