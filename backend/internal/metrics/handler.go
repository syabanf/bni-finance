package metrics

import (
	"crypto/subtle"
	"net/http"
	"runtime"
	"strings"
)

// Handler serves the exposition endpoint.
//
// Access: when token is non-empty every scrape must carry
// `Authorization: Bearer <token>`; empty leaves the endpoint open.
//
// Open is the default because a metrics endpoint nobody can scrape is not
// hardening, it is decoration — and the inventory it reveals (route patterns)
// is already public through /docs and /openapi.json by design. What it must
// never reveal is business data, and that is guaranteed by construction rather
// than by configuration: no metric here takes a member id, invoice number,
// email, or amount as a label. See TestNoBusinessDataInLabels.
//
// Request volumes and error rates are still worth protecting in production.
// Set METRICS_TOKEN, or block /metrics at the ingress.
func Handler(reg *Registry, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			got := strings.TrimSpace(r.Header.Get("Authorization"))
			want := "Bearer " + token
			// Constant-time: a plain == leaks the token byte by byte to anyone
			// willing to time the responses.
			if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="metrics"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		// Version 0.0.4 is the text exposition format every Prometheus reads.
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		reg.Write(w)
	})
}

// PoolStats is the subset of pgxpool.Stat worth exporting. Declared as an
// interface so this package does not depend on the driver — and so tests can
// supply numbers without a database.
type PoolStats interface {
	AcquiredConns() int32
	IdleConns() int32
	TotalConns() int32
	MaxConns() int32
	AcquireCount() int64
	EmptyAcquireCount() int64
}

// RegisterPool exposes connection pool health. Pool exhaustion shows up as
// rising latency everywhere at once and is otherwise invisible; empty_acquire
// counts the times a request had to WAIT for a free connection, which is the
// number that moves first.
func RegisterPool(r *Registry, stats func() PoolStats) {
	gauge := func(name, help string, read func(PoolStats) float64) {
		r.NewGauge(name, help, func() float64 {
			s := stats()
			if s == nil {
				return 0
			}
			return read(s)
		})
	}
	gauge("db_pool_conns_acquired", "Koneksi database yang sedang dipakai.",
		func(s PoolStats) float64 { return float64(s.AcquiredConns()) })
	gauge("db_pool_conns_idle", "Koneksi database menganggur.",
		func(s PoolStats) float64 { return float64(s.IdleConns()) })
	gauge("db_pool_conns_total", "Total koneksi di pool.",
		func(s PoolStats) float64 { return float64(s.TotalConns()) })
	gauge("db_pool_conns_max", "Batas maksimum koneksi pool.",
		func(s PoolStats) float64 { return float64(s.MaxConns()) })
	gauge("db_pool_acquire_total", "Jumlah pengambilan koneksi.",
		func(s PoolStats) float64 { return float64(s.AcquireCount()) })
	gauge("db_pool_acquire_empty_total",
		"Pengambilan yang harus MENUNGGU koneksi kosong — naiknya angka ini tanda pool mepet.",
		func(s PoolStats) float64 { return float64(s.EmptyAcquireCount()) })
}

// RegisterRuntime exposes the Go runtime numbers that explain an unhealthy
// process: goroutine leaks, heap growth, and GC pressure.
func RegisterRuntime(r *Registry) {
	r.NewGauge("go_goroutines", "Jumlah goroutine yang berjalan.",
		func() float64 { return float64(runtime.NumGoroutine()) })

	memstat := func(name, help string, read func(*runtime.MemStats) float64) {
		r.NewGauge(name, help, func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return read(&m)
		})
	}
	memstat("go_memstats_alloc_bytes", "Byte heap yang sedang teralokasi.",
		func(m *runtime.MemStats) float64 { return float64(m.Alloc) })
	memstat("go_memstats_sys_bytes", "Byte memori yang diminta dari OS.",
		func(m *runtime.MemStats) float64 { return float64(m.Sys) })
	memstat("go_gc_cycles_total", "Jumlah siklus GC selesai.",
		func(m *runtime.MemStats) float64 { return float64(m.NumGC) })
}

// Integrations counts outbound and inbound calls to Paper.id, Xendit, and BNI
// VM. The blackbox page shows the last N calls; this survives a restart's worth
// of aggregation and can raise an alert, which a ring buffer cannot.
type Integrations struct {
	calls    *Counter
	duration *Histogram
}

func NewIntegrations(r *Registry) *Integrations {
	return &Integrations{
		calls: r.NewCounter("integration_calls_total",
			"Panggilan integrasi, per mitra, arah, dan hasil."),
		duration: r.NewHistogram("integration_call_duration_seconds",
			"Durasi panggilan integrasi dalam detik.",
			// Upstreams are slower than our own endpoints: Paper.id routinely
			// takes seconds, and the 60s client timeout must land inside a
			// bucket or a stall looks identical to a fast success.
			[]float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}),
	}
}

// Record is safe on a nil receiver, so wiring is optional.
func (i *Integrations) Record(integration, direction string, success bool, seconds float64) {
	if i == nil {
		return
	}
	outcome := "failed"
	if success {
		outcome = "success"
	}
	labels := Labels{"integration": integration, "direction": direction, "outcome": outcome}
	i.calls.Inc(labels)
	i.duration.Observe(Labels{"integration": integration, "direction": direction}, seconds)
}
