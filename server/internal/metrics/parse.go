package metrics

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Scrape is one parsed /metrics response: plain values by name, and
// histogram buckets by name.
type Scrape struct {
	Values     map[string]float64
	Histograms map[string]Buckets
}

// Buckets are a histogram's cumulative bucket counts by upper bound.
type Buckets struct {
	Bounds []float64 // ascending, the last one +Inf
	Counts []float64 // cumulative
}

// ParseText parses the Prometheus text format as this package writes it
// (no labels except histogram "le").
func ParseText(r io.Reader) (Scrape, error) {
	s := Scrape{Values: map[string]float64{}, Histograms: map[string]Buckets{}}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, val, ok := strings.Cut(line, " ")
		if !ok {
			return s, fmt.Errorf("metrics: bad line %q", line)
		}
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return s, fmt.Errorf("metrics: bad value in %q: %w", line, err)
		}
		if base, le, ok := strings.Cut(name, `_bucket{le="`); ok {
			le = strings.TrimSuffix(le, `"}`)
			bound := math.Inf(1)
			if le != "+Inf" {
				if bound, err = strconv.ParseFloat(le, 64); err != nil {
					return s, fmt.Errorf("metrics: bad bucket in %q", line)
				}
			}
			h := s.Histograms[base]
			h.Bounds = append(h.Bounds, bound)
			h.Counts = append(h.Counts, v)
			s.Histograms[base] = h
			continue
		}
		s.Values[name] = v
	}
	return s, sc.Err()
}

// Sub returns the observations made between an earlier scrape and this one.
func (b Buckets) Sub(earlier Buckets) Buckets {
	out := Buckets{Bounds: b.Bounds, Counts: make([]float64, len(b.Counts))}
	for i := range b.Counts {
		out.Counts[i] = b.Counts[i]
		if i < len(earlier.Counts) {
			out.Counts[i] -= earlier.Counts[i]
		}
	}
	return out
}

// Count is the number of observations.
func (b Buckets) Count() float64 {
	if len(b.Counts) == 0 {
		return 0
	}
	return b.Counts[len(b.Counts)-1]
}

// Quantile returns the upper bound of the bucket holding the q-quantile:
// an upper estimate, never an under-report. +Inf means it overflowed.
func (b Buckets) Quantile(q float64) time.Duration {
	total := b.Count()
	if total == 0 {
		return 0
	}
	rank := math.Ceil(q * total)
	i := sort.Search(len(b.Counts), func(i int) bool { return b.Counts[i] >= rank })
	if i >= len(b.Bounds) || math.IsInf(b.Bounds[i], 1) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(b.Bounds[i] * float64(time.Second))
}

// Highest returns the upper bound of the highest non-empty bucket: every
// observation was at most this long.
func (b Buckets) Highest() time.Duration {
	prev := 0.0
	last := -1
	for i, c := range b.Counts {
		if c > prev {
			last = i
		}
		prev = c
	}
	if last < 0 {
		return 0
	}
	if math.IsInf(b.Bounds[last], 1) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(b.Bounds[last] * float64(time.Second))
}
