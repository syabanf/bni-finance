package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTP bundles the request metrics every endpoint feeds.
type HTTP struct {
	requests *Counter
	duration *Histogram
	inFlight *Counter // paired with a completion counter; see InFlight below
	done     *Counter
}

// NewHTTP registers the standard request metrics on r.
func NewHTTP(r *Registry) *HTTP {
	h := &HTTP{
		requests: r.NewCounter("http_requests_total",
			"Total permintaan HTTP, dikelompokkan per pola route, method, dan kelas status."),
		duration: r.NewHistogram("http_request_duration_seconds",
			"Durasi permintaan HTTP dalam detik.", DefaultBuckets),
		inFlight: r.NewCounter("http_requests_started_total",
			"Permintaan HTTP yang mulai diproses."),
		done: r.NewCounter("http_requests_completed_total",
			"Permintaan HTTP yang selesai diproses."),
	}
	r.NewGauge("http_requests_in_flight",
		"Permintaan HTTP yang sedang diproses.", func() float64 {
			return h.inFlight.total() - h.done.total()
		})
	return h
}

func (c *Counter) total() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var sum float64
	for _, s := range c.values {
		sum += s.value
	}
	return sum
}

// statusRecorder captures the status code, which http.ResponseWriter hides once
// written.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Middleware records one observation per request.
//
// THE LABEL IS THE ROUTE PATTERN, NEVER THE URL. `/api/v1/invoices/{id}` is one
// series; `/api/v1/invoices/<uuid>` would be one series PER INVOICE. Prometheus
// keeps every series it has ever seen in memory, so a path label taken from
// r.URL.Path turns normal traffic into an unbounded memory leak in the scraper
// — the classic way a metrics endpoint takes down the monitoring it was added
// to provide. TestPathLabelUsesRoutePatternNotURL pins this.
//
// The muxes are REQUIRED arguments, not something read from the request, and
// this signature is the scar tissue of two identical mistakes:
//
//  1. The first version read r.Pattern. Go 1.22 fills that during ServeHTTP on
//     a CLONE of the request, so a wrapper outside the mux never sees it —
//     every endpoint was labelled "other".
//  2. The second version asked only the root mux. The API mounts a second mux
//     at the /api/ prefix, so every endpoint under it came back "/api/".
//
// Both had safe cardinality, useless data, and no failing test. Pass every mux
// that can match, innermost first; the first specific (non-prefix) pattern wins.
//
// Cost: one extra routing-tree walk per mux per request. A map lookup against a
// few dozen patterns — cheaper than not knowing which endpoint is slow.
func (h *HTTP) Middleware(muxes ...*http.ServeMux) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := routeLabel(r, muxes...)
			labels := Labels{"method": r.Method, "route": route}
			h.inFlight.Inc(labels)

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			started := time.Now()
			next.ServeHTTP(rec, r)
			elapsed := time.Since(started).Seconds()

			h.done.Inc(labels)
			h.duration.Observe(labels, elapsed)
			h.requests.Inc(Labels{
				"method": r.Method,
				"route":  route,
				"status": strconv.Itoa(rec.status),
			})
		})
	}
}

// routeLabel returns the most specific matched pattern, method prefix stripped,
// or "other" when nothing matched.
func routeLabel(r *http.Request, muxes ...*http.ServeMux) string {
	best := ""
	for _, mux := range muxes {
		if mux == nil {
			continue
		}
		_, p := mux.Handler(r)
		p = stripMethod(p)
		if p == "" {
			continue
		}
		// A pattern ending in "/" is a prefix mount ("/api/"), not an endpoint.
		// Keep it only as a fallback; a real pattern from another mux wins.
		if !strings.HasSuffix(p, "/") {
			return p
		}
		if best == "" {
			best = p
		}
	}
	if best != "" {
		return best
	}
	if p := stripMethod(r.Pattern); p != "" {
		return p
	}
	// Nothing matched — a 404, whose path the caller chose. Using it here would
	// hand an attacker direct control of our series count.
	return "other"
}

// stripMethod turns "GET /api/v1/invoices/{id}" into "/api/v1/invoices/{id}";
// the method is already its own label.
func stripMethod(p string) string {
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[i:]
	}
	return ""
}
