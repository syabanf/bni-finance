// Package blackbox is a flight-recorder for integration traffic: every call to
// an external API (Paper.id, Xendit, BNI VM) and every callback we receive is
// captured with its request body, the endpoint hit, the response body, and
// whether it succeeded.
//
// Rekaman ditulis ke Postgres DAN disimpan di ring buffer memori. Dulu hanya
// memori, dan itu hilang tiap restart — cukup untuk "apa yang barusan terjadi",
// tetapi tidak untuk pertanyaan yang sebenarnya diajukan orang: "invoice ini
// dikirim kapan, dan waktu itu Paper.id menjawab apa", yang hampir selalu
// ditanyakan berhari-hari kemudian.
//
// Tanpa Store yang terpasang (tes, mode Data Contoh) ia tetap berfungsi penuh
// di memori saja, jadi tidak ada jalur yang menuntut database ada.
//
// SECURITY: callers pass only JSON bodies here, never headers. Credentials
// (Paper.id client_id/secret, bearer tokens) live in headers, so they cannot
// reach this recorder by construction. The page that reads it is admin-only.
package blackbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// Store adalah penyimpanan tahan-restart untuk rekaman. Repository di paket ini
// mengimplementasikannya; antarmukanya ada supaya tes tidak menuntut Postgres.
type Store interface {
	Insert(ctx context.Context, e Entry) error
	List(ctx context.Context, limit int) ([]Entry, error)
	Clear(ctx context.Context) error
	Prune(ctx context.Context) (int64, error)
}

const (
	// Outbound = a request we made to an external API.
	// Inbound  = a callback an external system made to us.
	Outbound = "outbound"
	Inbound  = "inbound"
)

// Entry is one recorded call. Request/Response are raw JSON so the page can
// pretty-print them without a second parse.
type Entry struct {
	ID          string          `json:"id"`
	Time        time.Time       `json:"time"`
	Integration string          `json:"integration"`
	Direction   string          `json:"direction"`
	Method      string          `json:"method"`
	URL         string          `json:"url"`
	Request     json.RawMessage `json:"request,omitempty"`
	Response    json.RawMessage `json:"response,omitempty"`
	Status      int             `json:"status"`
	Success     bool            `json:"success"`
	DurationMS  int64           `json:"durationMs"`
	Error       string          `json:"error,omitempty"`
}

// Recorder is a fixed-size, newest-first ring buffer. Safe for concurrent use —
// integration calls run on many goroutines at once.
type Recorder struct {
	mu      sync.Mutex
	entries []Entry
	max     int
	seq     uint64
	now     func() time.Time

	// observe is notified of every recorded call. It exists so metrics can be
	// derived from the ONE place all four integrations already funnel through,
	// instead of adding a counter to each client and eventually forgetting one.
	observe func(Call)

	// store membuat rekaman bertahan melewati restart. nil = memori saja.
	store Store
	log   *slog.Logger
}

// WithStore memasang penyimpanan tahan-restart.
func (r *Recorder) WithStore(s Store, log *slog.Logger) *Recorder {
	if r == nil {
		return nil
	}
	r.store = s
	r.log = log
	return r
}

// WithObserver attaches a callback invoked on every Record. The callback runs
// while the recorder's lock is held, so it must not block or call back in.
func (r *Recorder) WithObserver(fn func(Call)) *Recorder {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	r.observe = fn
	r.mu.Unlock()
	return r
}

// New returns a recorder holding at most max entries.
func New(max int) *Recorder {
	if max <= 0 {
		max = 200
	}
	return &Recorder{max: max, now: time.Now}
}

// Call describes one recorded interaction. Request and Response are raw bytes
// that will be stored as-is when they are valid JSON, or wrapped as a JSON
// string when they are not (so the page always receives valid JSON).
type Call struct {
	Integration string
	Direction   string
	Method      string
	URL         string
	Request     []byte
	Response    []byte
	Status      int
	Success     bool
	Duration    time.Duration
	Err         error
}

// Record stores one call. A nil recorder is a no-op, so integrations can be
// wired with an always-safe call site even when recording is disabled.
func (r *Recorder) Record(c Call) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.observe != nil {
		r.observe(c)
	}

	r.seq++
	e := Entry{
		ID:          strconv.FormatUint(r.seq, 10),
		Time:        r.now().UTC(),
		Integration: c.Integration,
		Direction:   c.Direction,
		Method:      c.Method,
		URL:         c.URL,
		Request:     asJSON(c.Request),
		Response:    asJSON(c.Response),
		Status:      c.Status,
		Success:     c.Success,
		DurationMS:  c.Duration.Milliseconds(),
	}
	if c.Err != nil {
		e.Error = c.Err.Error()
	}

	// Prepend (newest first), then cap.
	r.entries = append([]Entry{e}, r.entries...)
	if len(r.entries) > r.max {
		r.entries = r.entries[:r.max]
	}

	r.persist(e)
}

// persist menulis rekaman ke penyimpanan.
//
// Gagal menyimpan TIDAK boleh menggagalkan panggilan integrasinya: ini catatan
// diagnostik, bukan bagian dari transaksi bisnis. Kegagalannya dicatat ke log
// supaya tidak hilang tanpa jejak — kotak hitam yang diam-diam berhenti merekam
// lebih buruk daripada yang tidak ada.
//
// Dipanggil dengan r.mu masih dipegang, jadi urutan penulisan mengikuti urutan
// perekaman. Insert punya batas waktunya sendiri sehingga database yang lambat
// tidak menahan lock ini tanpa batas.
func (r *Recorder) persist(e Entry) {
	if r.store == nil {
		return
	}
	ctx := context.Background()
	if err := r.store.Insert(ctx, e); err != nil {
		if r.log != nil {
			r.log.Error("blackbox gagal menyimpan rekaman", "error", err,
				"integration", e.Integration, "url", e.URL)
		}
		return
	}
	// Pemangkasan tidak perlu tiap baris; tiap 200 rekaman sudah menjaga
	// tabelnya tetap terbatas tanpa membebani jalur permintaan.
	if r.seq%200 == 0 {
		if _, err := r.store.Prune(ctx); err != nil && r.log != nil {
			r.log.Warn("blackbox gagal memangkas rekaman lama", "error", err)
		}
	}
}

// List returns a copy of the entries, newest first.
//
// Membaca dari penyimpanan bila ada, sehingga halaman blackbox menampilkan
// riwayat penuh dan bukan hanya yang tersisa sejak restart terakhir. Bila
// pembacaan gagal, isi memori dikembalikan sebagai cadangan — lebih baik
// menampilkan sebagian daripada halaman kosong saat sedang menelusuri masalah.
func (r *Recorder) List() []Entry {
	if r == nil {
		return []Entry{}
	}
	if r.store != nil {
		got, err := r.store.List(context.Background(), r.max)
		if err == nil {
			return got
		}
		if r.log != nil {
			r.log.Error("blackbox gagal membaca riwayat, memakai isi memori", "error", err)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Entry, len(r.entries))
	copy(out, r.entries)
	return out
}

// Clear empties the buffer and the stored history.
func (r *Recorder) Clear() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.entries = nil
	store, log := r.store, r.log
	r.mu.Unlock()

	if store != nil {
		if err := store.Clear(context.Background()); err != nil && log != nil {
			log.Error("blackbox gagal mengosongkan riwayat", "error", err)
		}
	}
}

// asJSON keeps valid JSON as-is and wraps anything else as a JSON string, so a
// non-JSON error page from an upstream still renders instead of breaking the
// response.
func asJSON(b []byte) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	if json.Valid(b) {
		return json.RawMessage(b)
	}
	quoted, err := json.Marshal(string(b))
	if err != nil {
		return nil
	}
	return json.RawMessage(quoted)
}
