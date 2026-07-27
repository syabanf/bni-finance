// Package blackbox is a flight-recorder for integration traffic: every call to
// an external API (Paper.id, Xendit, BNI VM) and every callback we receive is
// captured with its request body, the endpoint hit, the response body, and
// whether it succeeded.
//
// It is an in-memory ring buffer, not a table: this is a diagnostic aid where
// only recent calls matter, so it avoids a migration and a write on every
// integration call. Entries are lost on restart — by design.
//
// SECURITY: callers pass only JSON bodies here, never headers. Credentials
// (Paper.id client_id/secret, bearer tokens) live in headers, so they cannot
// reach this recorder by construction. The page that reads it is admin-only.
package blackbox

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"
)

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
}

// List returns a copy of the entries, newest first.
func (r *Recorder) List() []Entry {
	if r == nil {
		return []Entry{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Entry, len(r.entries))
	copy(out, r.entries)
	return out
}

// Clear empties the buffer.
func (r *Recorder) Clear() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = nil
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
