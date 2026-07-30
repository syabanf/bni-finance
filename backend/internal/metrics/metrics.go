// Package metrics exposes Prometheus metrics without pulling in
// prometheus/client_golang.
//
// That library brings roughly ten transitive dependencies (protobuf, procfs,
// prometheus/common, …) into a module whose whole point is one direct
// dependency. What we actually need from it is a counter, a histogram, and the
// text exposition format — a line protocol simple enough to write correctly in
// a few hundred lines and verify against a test.
//
// The trade is real and worth stating: no exemplars, no native histograms, no
// pushgateway, and no protobuf negotiation. Every Prometheus and every
// OpenMetrics-compatible scraper reads the text format, so none of that is
// needed to be scraped. If those features ever are, swap this package for the
// real client — the call sites are `Inc`/`Observe`, which it also provides.
package metrics

import (
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Registry holds every metric the process exposes.
//
// Deliberately not a package-level singleton: tests need a fresh registry per
// case, and a global would make one test's counters visible to the next.
type Registry struct {
	mu      sync.RWMutex
	metrics []metric
}

func NewRegistry() *Registry { return &Registry{} }

type metric interface {
	write(w io.Writer)
}

// --- label sets --------------------------------------------------------------

// Labels are the dimensions of one series. Order does not matter; the key is
// canonicalised so {a,b} and {b,a} are the same series.
type Labels map[string]string

func (l Labels) key() string {
	if len(l) == 0 {
		return ""
	}
	names := slices.Sorted(maps.Keys(l))
	var b strings.Builder
	for i, n := range names {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(n)
		b.WriteByte('=')
		b.WriteString(l[n])
	}
	return b.String()
}

// render produces the `{a="1",b="2"}` suffix, escaping per the exposition spec.
func (l Labels) render(extra ...string) string {
	if len(l) == 0 && len(extra) == 0 {
		return ""
	}
	names := slices.Sorted(maps.Keys(l))
	parts := make([]string, 0, len(names)+len(extra)/2)
	for _, n := range names {
		parts = append(parts, n+`="`+escapeLabelValue(l[n])+`"`)
	}
	for i := 0; i+1 < len(extra); i += 2 {
		parts = append(parts, extra[i]+`="`+escapeLabelValue(extra[i+1])+`"`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// escapeLabelValue follows the exposition format: backslash, double quote and
// newline are escaped. Skipping this lets a stray quote in a label produce a
// payload the scraper rejects — silently losing every metric, not just one.
func escapeLabelValue(v string) string {
	if !strings.ContainsAny(v, `\"`+"\n") {
		return v
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return r.Replace(v)
}

// --- counter -----------------------------------------------------------------

// Counter is a monotonically increasing value per label set.
type Counter struct {
	name, help string
	mu         sync.RWMutex
	values     map[string]*counterSeries
}

type counterSeries struct {
	labels Labels
	value  float64
}

func (r *Registry) NewCounter(name, help string) *Counter {
	c := &Counter{name: name, help: help, values: map[string]*counterSeries{}}
	r.mu.Lock()
	r.metrics = append(r.metrics, c)
	r.mu.Unlock()
	return c
}

func (c *Counter) Add(labels Labels, delta float64) {
	k := labels.key()

	c.mu.RLock()
	s, ok := c.values[k]
	c.mu.RUnlock()
	if ok {
		c.mu.Lock()
		s.value += delta
		c.mu.Unlock()
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.values[k]; ok { // another goroutine won the race
		s.value += delta
		return
	}
	c.values[k] = &counterSeries{labels: maps.Clone(labels), value: delta}
}

func (c *Counter) Inc(labels Labels) { c.Add(labels, 1) }

// Series counts distinct label sets — what a cardinality test asserts on.
func (c *Counter) Series() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.values)
}

func (c *Counter) write(w io.Writer) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n", c.name, c.help, c.name)
	for _, k := range slices.Sorted(maps.Keys(c.values)) {
		s := c.values[k]
		fmt.Fprintf(w, "%s%s %s\n", c.name, s.labels.render(), formatFloat(s.value))
	}
}

// --- gauge -------------------------------------------------------------------

// Gauge reads a value at scrape time rather than storing one. Pool sizes and
// runtime stats are already counted somewhere else; copying them into a stored
// value would only add a way for the two to disagree.
type Gauge struct {
	name, help string
	read       func() float64
}

func (r *Registry) NewGauge(name, help string, read func() float64) *Gauge {
	g := &Gauge{name: name, help: help, read: read}
	r.mu.Lock()
	r.metrics = append(r.metrics, g)
	r.mu.Unlock()
	return g
}

func (g *Gauge) write(w io.Writer) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n%s %s\n",
		g.name, g.help, g.name, g.name, formatFloat(g.read()))
}

// --- histogram ---------------------------------------------------------------

// DefaultBuckets covers a web API: sub-millisecond to ten seconds. The upper
// bound matters — anything slower than the top bucket is invisible to a
// latency quantile, and this API calls Paper.id, which takes seconds.
var DefaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type Histogram struct {
	name, help string
	buckets    []float64
	mu         sync.RWMutex
	values     map[string]*histogramSeries
}

type histogramSeries struct {
	labels Labels
	counts []uint64 // one per bucket, cumulative at write time
	sum    float64
	total  uint64
}

func (r *Registry) NewHistogram(name, help string, buckets []float64) *Histogram {
	if len(buckets) == 0 {
		buckets = DefaultBuckets
	}
	b := slices.Clone(buckets)
	sort.Float64s(b)

	h := &Histogram{name: name, help: help, buckets: b, values: map[string]*histogramSeries{}}
	r.mu.Lock()
	r.metrics = append(r.metrics, h)
	r.mu.Unlock()
	return h
}

func (h *Histogram) Observe(labels Labels, v float64) {
	k := labels.key()

	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.values[k]
	if !ok {
		s = &histogramSeries{labels: maps.Clone(labels), counts: make([]uint64, len(h.buckets))}
		h.values[k] = s
	}
	s.sum += v
	s.total++
	// Bucket boundaries are "less than or equal"; the implicit +Inf bucket is
	// `total`, so a value above every bound needs no slot of its own.
	for i, upper := range h.buckets {
		if v <= upper {
			s.counts[i]++
			break
		}
	}
}

func (h *Histogram) Series() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.values)
}

func (h *Histogram) write(w io.Writer) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s histogram\n", h.name, h.help, h.name)
	for _, k := range slices.Sorted(maps.Keys(h.values)) {
		s := h.values[k]
		var cumulative uint64
		for i, upper := range h.buckets {
			cumulative += s.counts[i]
			fmt.Fprintf(w, "%s_bucket%s %d\n",
				h.name, s.labels.render("le", formatFloat(upper)), cumulative)
		}
		fmt.Fprintf(w, "%s_bucket%s %d\n", h.name, s.labels.render("le", "+Inf"), s.total)
		fmt.Fprintf(w, "%s_sum%s %s\n", h.name, s.labels.render(), formatFloat(s.sum))
		fmt.Fprintf(w, "%s_count%s %d\n", h.name, s.labels.render(), s.total)
	}
}

// --- exposition ---------------------------------------------------------------

// Write renders every metric in the Prometheus text exposition format.
func (r *Registry) Write(w io.Writer) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.metrics {
		m.write(w)
	}
}

// formatFloat keeps integers integral. Prometheus accepts `1.0`, but `1` is
// what every other exporter emits and what golden-file comparisons expect.
func formatFloat(v float64) string {
	if math.IsInf(v, 1) {
		return "+Inf"
	}
	if math.IsInf(v, -1) {
		return "-Inf"
	}
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}
