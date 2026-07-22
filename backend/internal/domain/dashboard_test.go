package domain

import "testing"

func TestPercentChange(t *testing.T) {
	cases := []struct {
		name              string
		current, previous int
		want              float64
	}{
		{"naik dua kali lipat", 20, 10, 100},
		{"turun separuh", 5, 10, -50},
		{"stabil", 10, 10, 0},
		// Dividing by zero must not produce Inf/NaN — those break JSON encoding.
		{"tumbuh dari nol", 7, 0, 100},
		{"dua-duanya nol", 0, 0, 0},
		{"jatuh ke nol", 0, 8, -100},
		{"dibulatkan satu desimal", 1, 3, -66.7},
	}

	for _, c := range cases {
		if got := PercentChange(c.current, c.previous); got != c.want {
			t.Errorf("%s: PercentChange(%d, %d) = %v, harusnya %v",
				c.name, c.current, c.previous, got, c.want)
		}
	}
}

func TestActionForStatus(t *testing.T) {
	cases := map[InvoiceStatus]AuditAction{
		StatusSent:      AuditSent,
		StatusPaid:      AuditPaid,
		StatusCancelled: AuditCancelled,
		StatusOverdue:   AuditOverdue,
		StatusDraft:     AuditUpdated, // tidak ada aksi khusus untuk kembali ke draft
	}
	for status, want := range cases {
		if got := ActionForStatus(status); got != want {
			t.Errorf("ActionForStatus(%s) = %s, harusnya %s", status, got, want)
		}
	}
}
