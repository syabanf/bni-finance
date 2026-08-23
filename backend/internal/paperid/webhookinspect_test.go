package paperid

import (
	"context"
	"strings"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/blackbox"
)

// Payload yang bentuknya berbeda TIDAK menghasilkan error dari encoding/json —
// field tak dikenal diabaikan, field yang dikenal dibiarkan kosong. Callback
// dijawab 200, tidak ada invoice yang dilunasi, dan tidak ada yang merah.
//
// Inilah yang membuat ketidakcocokan format bisa berjalan berminggu-minggu
// tanpa ketahuan. Catatan inspeksi ada supaya selisihnya terlihat pada callback
// PERTAMA.
func TestInspectPayloadMenemukanSelisih(t *testing.T) {
	kasus := []struct {
		nama   string
		body   string
		memuat []string
	}{
		{
			nama: "bentuk yang kita harapkan — bersih",
			body: `{"ref_id":"r1","payment_date":"2026-08-23",
			        "payment_info":{"status":"PAID","amount":1500000},
			        "additional_info":{"invoices":[{"uuid":"u1","number":"INV-1"}]}}`,
			memuat: nil,
		},
		{
			nama:   "nama field berbeda — inilah kegagalan senyapnya",
			body:   `{"invoice_number":"INV-1","status":"PAID","paid_amount":1500000}`,
			memuat: []string{"invoice_number", "payment_info TIDAK ADA", "TIDAK ADA identitas invoice"},
		},
		{
			nama:   "status hilang dari payment_info",
			body:   `{"ref_id":"r1","payment_info":{"amount":1500000}}`,
			memuat: []string{"payment_info.status TIDAK ADA"},
		},
		{
			nama:   "field tambahan yang kita buang",
			body:   `{"ref_id":"r1","signature":"abc","payment_info":{"status":"PAID","fee":2500}}`,
			memuat: []string{"signature", "payment_info.fee"},
		},
		{
			nama:   "tanpa identitas invoice sama sekali",
			body:   `{"payment_info":{"status":"PAID"}}`,
			memuat: []string{"TIDAK ADA identitas invoice"},
		},
		{
			nama:   "bukan objek JSON",
			body:   `["bukan","objek"]`,
			memuat: []string{"bukan objek JSON"},
		},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			notes := inspectPayload([]byte(k.body))
			gab := strings.Join(notes, " | ")
			if len(k.memuat) == 0 {
				if len(notes) != 0 {
					t.Fatalf("bentuk yang benar tidak boleh menghasilkan catatan, dapat: %s", gab)
				}
				return
			}
			for _, w := range k.memuat {
				if !strings.Contains(gab, w) {
					t.Errorf("catatan tidak menyebut %q\n  dapat: %s", w, gab)
				}
			}
		})
	}
}

// Blackbox harus menyimpan apa yang PAPER.ID KIRIM, bukan apa yang kita pahami.
// Dulu yang direkam adalah struct hasil parse — sehingga pada format yang tidak
// cocok, halaman blackbox menampilkan struct nyaris kosong dan justru
// menyesatkan orang yang sedang mendiagnosis.
func TestBlackboxMenyimpanBodyMentah(t *testing.T) {
	rec := blackbox.New(20)
	svc := NewService(&stubStore{}, "https://contoh.invalid", "id", "secret", "rahasia", rec)

	asing := `{"invoice_number":"INV-1","status":"PAID","signature":"xyz"}`
	svc.HandleWebhook(context.Background(), "rahasia", []byte(asing))

	entries := rec.List()
	if len(entries) != 1 {
		t.Fatalf("harus 1 entri, dapat %d", len(entries))
	}
	e := entries[0]

	// Body mentah utuh, termasuk field yang tidak kita pahami.
	for _, w := range []string{"invoice_number", "signature"} {
		if !strings.Contains(string(e.Request), w) {
			t.Errorf("body mentah harus memuat %q, dapat: %s", w, e.Request)
		}
	}
	// Dan catatan selisihnya ikut tersimpan.
	if !strings.Contains(string(e.Response), "catatanFormat") {
		t.Errorf("catatan format harus ikut tersimpan, dapat: %s", e.Response)
	}
	if !strings.Contains(string(e.Response), "invoice_number") {
		t.Errorf("catatan harus menyebut field asing, dapat: %s", e.Response)
	}
}

// Token salah tetap merekam body mentah — justru saat callback ditolak,
// orang paling ingin tahu apa yang sebenarnya dikirim.
func TestTokenSalahTetapMerekamBodyMentah(t *testing.T) {
	rec := blackbox.New(20)
	svc := NewService(&stubStore{}, "https://contoh.invalid", "id", "secret", "rahasia", rec)

	svc.HandleWebhook(context.Background(), "salah", []byte(`{"ref_id":"r-9"}`))

	entries := rec.List()
	if len(entries) != 1 || !strings.Contains(string(entries[0].Request), "r-9") {
		t.Fatalf("body mentah harus tersimpan meski token ditolak: %+v", entries)
	}
	if entries[0].Status != 401 || entries[0].Success {
		t.Errorf("entri harus 401 dan tidak sukses: %+v", entries[0])
	}
}
