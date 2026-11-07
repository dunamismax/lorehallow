// Package procstat reads process CPU time and memory for load tests and
// metrics, on Linux and macOS.
package procstat

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// SelfCPU returns this process's user plus system CPU time.
func SelfCPU() time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

// RSS returns the resident set size of process pid in bytes, from ps(1),
// which reports it the same way on Linux and macOS (KiB).
func RSS(pid int) (int64, error) {
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, fmt.Errorf("ps: %w", err)
	}
	kb, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("ps rss %q: %w", out, err)
	}
	return kb * 1024, nil
}

// PeakRSS samples a process's RSS on an interval and keeps the peak.
type PeakRSS struct {
	stop chan struct{}
	done chan struct{}
	mu   sync.Mutex
	peak int64
}

// NewPeakRSS starts sampling pid every interval.
func NewPeakRSS(pid int, interval time.Duration) *PeakRSS {
	p := &PeakRSS{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			p.sample(pid)
			select {
			case <-p.stop:
				p.sample(pid)
				return
			case <-t.C:
			}
		}
	}()
	return p
}

func (p *PeakRSS) sample(pid int) {
	if v, err := RSS(pid); err == nil {
		p.mu.Lock()
		p.peak = max(p.peak, v)
		p.mu.Unlock()
	}
}

// Stop stops sampling and returns the peak in bytes.
func (p *PeakRSS) Stop() int64 {
	close(p.stop)
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.peak
}
