package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateJSONRoundTrip(t *testing.T) {
	type payload struct {
		DueDate Date `json:"dueDate"`
	}

	raw := `{"dueDate":"2026-06-17"}`
	var p payload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := p.DueDate.String(); got != "2026-06-17" {
		t.Errorf("parse: dapat %q", got)
	}

	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Harus tetap YYYY-MM-DD, bukan RFC3339 — kontrak dengan klien TypeScript.
	if string(out) != raw {
		t.Errorf("round-trip berubah: dapat %s, harusnya %s", out, raw)
	}
}

func TestDateRejectsBadFormat(t *testing.T) {
	var d Date
	if err := json.Unmarshal([]byte(`"17-06-2026"`), &d); err == nil {
		t.Error("format DD-MM-YYYY seharusnya ditolak")
	}
}

func TestDateScanFromDriver(t *testing.T) {
	var d Date
	ts := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)
	if err := d.Scan(ts); err != nil {
		t.Fatalf("scan time.Time: %v", err)
	}
	if d.String() != "2026-06-17" {
		t.Errorf("scan: dapat %q", d.String())
	}

	if err := d.Scan(nil); err != nil {
		t.Fatalf("scan nil: %v", err)
	}
	if !d.IsZero() {
		t.Error("scan nil harus menghasilkan zero Date")
	}

	// Value() harus mengembalikan nil untuk zero agar kolom nullable aman.
	v, err := Date{}.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v != nil {
		t.Errorf("zero Date harus jadi NULL, dapat %v", v)
	}
}
