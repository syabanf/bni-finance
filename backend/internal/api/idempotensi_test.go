package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/mailer"
)

// suratMati memenuhi renewal.Surat tanpa pernah mengirim apa pun.
type suratMati struct{}

func (suratMati) Siap() bool                                { return false }
func (suratMati) Kirim(context.Context, mailer.Pesan) error { return nil }

func decode[T any](t *testing.T, b []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	return v
}

func jumlahAudit(t *testing.T, s *liveStack, invoiceID string) int {
	t.Helper()
	var n int
	if err := s.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM invoice_audit_log WHERE invoice_id = $1", invoiceID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// SETIAP AKSI CRUD AMAN DIULANG.
//
// Klik ganda, jaringan yang mengulang permintaan, dan tombol yang ditekan lagi
// setelah halaman lambat menjawab semuanya mengirim permintaan yang sama dua
// kali. Permintaan kedua harus berakhir di keadaan yang sama dengan yang
// pertama, atau ditolak dengan 404/409 tanpa mengubah apa pun.
func TestIdempotensiCRUD(t *testing.T) {
	s := newLiveServer(t)
	ctx := context.Background()

	t.Run("chapter", func(t *testing.T) {
		body := `{"id":"ch-idem","name":"Idem","displayName":"BNI Idem"}`
		s.req(t, http.MethodPost, "/api/v1/chapters", body, http.StatusCreated)
		s.req(t, http.MethodPost, "/api/v1/chapters", body, http.StatusConflict)

		ubah := `{"displayName":"BNI Idem Baru","areaName":"Jakarta Barat","cityName":"Jakarta"}`
		a := decode[domain.Chapter](t, s.req(t, http.MethodPatch, "/api/v1/chapters/ch-idem", ubah, http.StatusOK))
		b := decode[domain.Chapter](t, s.req(t, http.MethodPatch, "/api/v1/chapters/ch-idem", ubah, http.StatusOK))
		if a.DisplayName != b.DisplayName || *a.AreaName != *b.AreaName || *b.CityName != "Jakarta" {
			t.Errorf("ubah chapter dua kali harus sama: %+v / %+v", a, b)
		}

		s.req(t, http.MethodPost, "/api/v1/chapters", `{"id":"ch-idem-kosong","name":"Kosong","displayName":"BNI Kosong"}`, http.StatusCreated)
		s.req(t, http.MethodDelete, "/api/v1/chapters/ch-idem-kosong", "", http.StatusNoContent)
		s.req(t, http.MethodDelete, "/api/v1/chapters/ch-idem-kosong", "", http.StatusNotFound)
	})

	m := decode[domain.Member](t, s.req(t, http.MethodPost, "/api/v1/members",
		`{"chapterId":"ch-idem","name":"Idem Satu","email":"satu@contoh.id","phone":"081200000001"}`, http.StatusCreated))

	t.Run("member", func(t *testing.T) {
		ubah := `{"email":"baru@contoh.id","phone":"081200000002","status":"inactive","renewalDate":"2027-01-01","joinedDate":"2025-01-01"}`
		a := decode[domain.Member](t, s.req(t, http.MethodPatch, "/api/v1/members/"+m.ID, ubah, http.StatusOK))
		b := decode[domain.Member](t, s.req(t, http.MethodPatch, "/api/v1/members/"+m.ID, ubah, http.StatusOK))
		if *a.Email != *b.Email || a.Status != b.Status || a.RenewalDate.String() != "2027-01-01" ||
			b.RenewalDate.String() != a.RenewalDate.String() || b.JoinedDate.String() != "2025-01-01" {
			t.Errorf("ubah member dua kali harus sama: %+v / %+v", a, b)
		}
		for range 2 {
			kosong := decode[domain.Member](t, s.req(t, http.MethodPatch, "/api/v1/members/"+m.ID, `{"email":""}`, http.StatusOK))
			if kosong.Email != nil {
				t.Errorf("email yang dikosongkan harus null, dapat %q", *kosong.Email)
			}
		}
	})

	t.Run("permintaan konfirmasi", func(t *testing.T) {
		minta := `{"memberIds":["` + m.ID + `"],"period":"2099-S1"}`
		pertama := decode[struct{ Dibuat, Dilewati int }](t, s.req(t, http.MethodPost, "/api/v1/renewal-requests", minta, http.StatusCreated))
		kedua := decode[struct{ Dibuat, Dilewati int }](t, s.req(t, http.MethodPost, "/api/v1/renewal-requests", minta, http.StatusCreated))
		if pertama.Dibuat != 1 || kedua.Dibuat != 0 || kedua.Dilewati != 1 {
			t.Errorf("minta dua kali harus membuat satu baris: %+v / %+v", pertama, kedua)
		}
		var id string
		if err := s.pool.QueryRow(ctx, "SELECT id FROM renewal_requests WHERE member_id = $1 AND period = '2099-S1'", m.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		s.req(t, http.MethodDelete, "/api/v1/renewal-requests/"+id, "", http.StatusNoContent)
		s.req(t, http.MethodDelete, "/api/v1/renewal-requests/"+id, "", http.StatusNotFound)

		// Jawaban MC adalah dasar invoice renewal, jadi tidak bisa dibatalkan.
		s.req(t, http.MethodPost, "/api/v1/renewal-requests", `{"memberIds":["`+m.ID+`"],"period":"2099-S2"}`, http.StatusCreated)
		if err := s.pool.QueryRow(ctx, `UPDATE renewal_requests SET answer = 'will_renew'
			WHERE member_id = $1 AND period = '2099-S2' RETURNING id`, m.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		for range 2 {
			s.req(t, http.MethodDelete, "/api/v1/renewal-requests/"+id, "", http.StatusConflict)
		}
	})

	t.Run("invoice draft", func(t *testing.T) {
		konfirmasiRenewal(t, s.pool, m.ID)
		inv := decode[domain.Invoice](t, s.req(t, http.MethodPost, "/api/v1/invoices",
			`{"memberId":"`+m.ID+`","chapterId":"ch-idem","type":"renewal","amount":1500000,`+
				`"dueDate":"2026-12-01","periodStart":"2026-12-01","periodEnd":"2027-12-01"}`, http.StatusCreated))
		ubah := `{"amount":1750000,"periodStart":"2027-01-01","periodEnd":"2028-01-01","notes":"koreksi"}`
		a := decode[domain.Invoice](t, s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID, ubah, http.StatusOK))
		b := decode[domain.Invoice](t, s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID, ubah, http.StatusOK))
		if a.Amount != 1_750_000 || b.Amount != a.Amount || b.PeriodEnd.String() != "2028-01-01" || b.Status != domain.StatusDraft {
			t.Errorf("ubah draft dua kali harus sama: %+v / %+v", a, b)
		}
		if n := jumlahAudit(t, s, inv.ID); n != 2 {
			t.Errorf("ubah draft dua kali harus menulis satu entri riwayat, total %d", n)
		}
		s.req(t, http.MethodDelete, "/api/v1/invoices/"+inv.ID, "", http.StatusNoContent)
		s.req(t, http.MethodDelete, "/api/v1/invoices/"+inv.ID, "", http.StatusNotFound)
	})

	t.Run("invoice terkirim", func(t *testing.T) {
		konfirmasiRenewal(t, s.pool, m.ID)
		inv := decode[domain.Invoice](t, s.req(t, http.MethodPost, "/api/v1/invoices",
			`{"memberId":"`+m.ID+`","chapterId":"ch-idem","type":"renewal","amount":1500000,`+
				`"dueDate":"2026-12-01","periodStart":"2026-12-01","periodEnd":"2027-12-01"}`, http.StatusCreated))
		s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID, `{"status":"sent"}`, http.StatusOK)
		// Tautan Paper.id invoice terkirim tetap hidup, jadi tidak boleh dihapus.
		s.req(t, http.MethodDelete, "/api/v1/invoices/"+inv.ID, "", http.StatusConflict)

		batal := `{"status":"cancelled","cancelReason":"uji batal ganda"}`
		s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID, batal, http.StatusOK)
		s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID, batal, http.StatusOK)
		akhir := decode[domain.Invoice](t, s.req(t, http.MethodGet, "/api/v1/invoices/"+inv.ID, "", http.StatusOK))
		if akhir.Status != domain.StatusCancelled {
			t.Errorf("status akhir %s, harus cancelled", akhir.Status)
		}
		var nBatal int
		if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM invoice_audit_log WHERE invoice_id = $1 AND action = 'cancelled'",
			inv.ID).Scan(&nBatal); err != nil {
			t.Fatal(err)
		}
		if n := jumlahAudit(t, s, inv.ID); nBatal != 1 || n != 3 {
			t.Errorf("pembatalan ganda harus tercatat sekali (dibuat, dikirim, dibatalkan), dapat %d batal dari %d", nBatal, n)
		}
		// Invoice batal tetap tertutup untuk nominal dan periodenya.
		s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID, `{"amount":1}`, http.StatusConflict)
	})

	t.Run("pembayaran", func(t *testing.T) {
		konfirmasiRenewal(t, s.pool, m.ID)
		inv := decode[domain.Invoice](t, s.req(t, http.MethodPost, "/api/v1/invoices",
			`{"memberId":"`+m.ID+`","chapterId":"ch-idem","type":"renewal","amount":1500000,`+
				`"dueDate":"2026-12-01","periodStart":"2027-12-01","periodEnd":"2028-12-01"}`, http.StatusCreated))
		s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID, `{"status":"sent"}`, http.StatusOK)

		// Pembayaran yang belum melunasi invoice boleh dihapus, sekali.
		cicil := decode[domain.Payment](t, s.req(t, http.MethodPost, "/api/v1/payments",
			`{"invoiceId":"`+inv.ID+`","amount":500000,"paymentMethod":"cash","settleInvoice":false}`, http.StatusCreated))
		s.req(t, http.MethodDelete, "/api/v1/payments/"+cicil.ID, "", http.StatusNoContent)
		s.req(t, http.MethodDelete, "/api/v1/payments/"+cicil.ID, "", http.StatusNotFound)

		lunas := decode[domain.Payment](t, s.req(t, http.MethodPost, "/api/v1/payments",
			`{"invoiceId":"`+inv.ID+`","amount":1500000,"paymentMethod":"bank_transfer"}`, http.StatusCreated))
		ubah := `{"paymentMethod":"cash","note":"dibayar di kantor"}`
		a := decode[domain.Payment](t, s.req(t, http.MethodPatch, "/api/v1/payments/"+lunas.ID, ubah, http.StatusOK))
		b := decode[domain.Payment](t, s.req(t, http.MethodPatch, "/api/v1/payments/"+lunas.ID, ubah, http.StatusOK))
		if *a.PaymentMethod != "cash" || *b.Note != *a.Note || b.Amount != 1_500_000 {
			t.Errorf("ubah pembayaran dua kali harus sama: %+v / %+v", a, b)
		}
		for range 2 {
			s.req(t, http.MethodDelete, "/api/v1/payments/"+lunas.ID, "", http.StatusConflict)
		}
		akhir := decode[domain.Invoice](t, s.req(t, http.MethodGet, "/api/v1/invoices/"+inv.ID, "", http.StatusOK))
		if akhir.Status != domain.StatusPaid {
			t.Errorf("invoice harus tetap lunas, dapat %s", akhir.Status)
		}
	})

	t.Run("hapus member", func(t *testing.T) {
		// Permintaan konfirmasi ikut terhapus, alih-alih menggagalkan penghapusan.
		dua := decode[domain.Member](t, s.req(t, http.MethodPost, "/api/v1/members",
			`{"chapterId":"ch-idem","name":"Idem Dua","status":"active"}`, http.StatusCreated))
		s.req(t, http.MethodPost, "/api/v1/renewal-requests", `{"memberIds":["`+dua.ID+`"],"period":"2099-S1"}`, http.StatusCreated)
		s.req(t, http.MethodDelete, "/api/v1/members/"+dua.ID, "", http.StatusNoContent)
		s.req(t, http.MethodDelete, "/api/v1/members/"+dua.ID, "", http.StatusNotFound)
		var sisa int
		if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM renewal_requests WHERE member_id = $1", dua.ID).Scan(&sisa); err != nil {
			t.Fatal(err)
		}
		if sisa != 0 {
			t.Errorf("permintaan konfirmasi member terhapus masih tersisa %d", sisa)
		}
		// Member yang punya invoice tetap ditolak.
		for range 2 {
			s.req(t, http.MethodDelete, "/api/v1/members/"+m.ID, "", http.StatusConflict)
		}
		s.req(t, http.MethodDelete, "/api/v1/chapters/ch-idem", "", http.StatusConflict)
	})
}
