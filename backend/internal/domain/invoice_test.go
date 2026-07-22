package domain

import "testing"

func TestStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to InvoiceStatus
		want     bool
	}{
		{StatusDraft, StatusSent, true},
		{StatusDraft, StatusCancelled, true},
		{StatusDraft, StatusPaid, false}, // tidak boleh lompat
		{StatusSent, StatusPaid, true},
		{StatusSent, StatusOverdue, true},
		{StatusOverdue, StatusPaid, true},
		{StatusPaid, StatusCancelled, false}, // terminal
		{StatusPaid, StatusSent, false},
		{StatusCancelled, StatusDraft, false},
		{StatusPaid, StatusPaid, true}, // no-op selalu boleh
	}

	for _, c := range cases {
		if got := c.from.CanTransitionTo(c.to); got != c.want {
			t.Errorf("%s → %s: dapat %v, harusnya %v", c.from, c.to, got, c.want)
		}
	}
}

func mustDate(t *testing.T, s string) Date {
	t.Helper()
	d, err := ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", s, err)
	}
	return d
}

func TestCreateInvoiceValidate(t *testing.T) {
	valid := CreateInvoiceInput{
		MemberID:    "mem-1",
		ChapterID:   "ch-1",
		Type:        TypeRenewal,
		Amount:      1_500_000,
		DueDate:     mustDate(t, "2026-06-01"),
		PeriodStart: mustDate(t, "2026-06-01"),
		PeriodEnd:   mustDate(t, "2027-06-01"),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("input valid ditolak: %v", err)
	}

	t.Run("amount nol ditolak", func(t *testing.T) {
		in := valid
		in.Amount = 0
		if err := in.Validate(); err == nil {
			t.Error("amount 0 seharusnya ditolak")
		}
	})

	t.Run("tipe tidak dikenal ditolak", func(t *testing.T) {
		in := valid
		in.Type = "subscription"
		if err := in.Validate(); err == nil {
			t.Error("tipe tidak dikenal seharusnya ditolak")
		}
	})

	t.Run("periode terbalik ditolak", func(t *testing.T) {
		in := valid
		in.PeriodEnd = mustDate(t, "2026-01-01")
		if err := in.Validate(); err == nil {
			t.Error("periodEnd sebelum periodStart seharusnya ditolak")
		}
	})

	t.Run("memberId kosong ditolak", func(t *testing.T) {
		in := valid
		in.MemberID = ""
		if err := in.Validate(); err == nil {
			t.Error("memberId kosong seharusnya ditolak")
		}
	})
}

func TestCreatePaymentValidateAndSettleDefault(t *testing.T) {
	in := CreatePaymentInput{InvoiceID: "inv-1", Amount: 1_500_000}
	if err := in.Validate(); err != nil {
		t.Fatalf("input valid ditolak: %v", err)
	}
	if !in.ShouldSettle() {
		t.Error("default ShouldSettle harus true")
	}

	no := false
	in.SettleInvoice = &no
	if in.ShouldSettle() {
		t.Error("ShouldSettle harus menghormati settleInvoice=false")
	}

	in.Amount = -1
	if err := in.Validate(); err == nil {
		t.Error("amount negatif seharusnya ditolak")
	}
}
