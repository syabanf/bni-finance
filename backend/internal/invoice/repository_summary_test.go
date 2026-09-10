package invoice

import (
	"context"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Ringkasan invoice menjawab DUA pertanyaan yang berbeda sekaligus.
//
//	ByStatus  "ada berapa di tiap status"      → filter status TIDAK berlaku,
//	                                              kalau tidak sisanya selalu nol
//	ByType    "dari yang sedang saya lihat,
//	           berapa pendaftaran & renewal"   → filter status BERLAKU
//
// Keduanya dihitung dari satu query GROUP BY status, type. Tes ini menjaga agar
// perbedaan perlakuan itu tetap ada — dan agar keduanya tidak diam-diam saling
// menular.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
func TestRingkasanPerTipeMengikutiFilterStatus(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, _, memA, _ := duaChapter(t, repo)
	nasional := scope.WithoutLimit(context.Background())

	// Dua renewal dan satu pendaftaran, semuanya draft.
	for i := 0; i < 2; i++ {
		in := contohInput(memA, chA, 1_000_000)
		in.Type = domain.TypeRenewal
		inv, err := repo.Create(nasional, in, "", "IDR")
		if err != nil {
			t.Fatalf("buat renewal: %v", err)
		}
		t.Cleanup(func() { _ = repo.Delete(nasional, inv.ID) })
	}
	in := contohInput(memA, chA, 5_000_000)
	in.Type = domain.TypeRegistration
	daftar, err := repo.Create(nasional, in, "", "IDR")
	if err != nil {
		t.Fatalf("buat pendaftaran: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(nasional, daftar.ID) })

	ambil := func(f domain.InvoiceFilter) *domain.InvoiceSummary {
		t.Helper()
		f.ChapterID = chA
		s, err := repo.Summary(nasional, f)
		if err != nil {
			t.Fatalf("ringkas: %v", err)
		}
		return s
	}

	// --- tanpa filter status --------------------------------------------------
	semua := ambil(domain.InvoiceFilter{})
	if got := semua.ByType["renewal"].Count; got != 2 {
		t.Errorf("byType[renewal] = %d, seharusnya 2", got)
	}
	if got := semua.ByType["registration"].Count; got != 1 {
		t.Errorf("byType[registration] = %d, seharusnya 1", got)
	}
	if got := semua.ByType["renewal"].Amount; got != 2_000_000 {
		t.Errorf("byType[renewal].amount = %d, seharusnya 2000000", got)
	}

	// TOTAL TIDAK BOLEH MENGHITUNG GANDA.
	//
	// Satu status kini muncul SEKALI PER TIPE di hasil query. Versi pertama
	// saya menjumlahkan total dari agregat berjalan ByStatus, sehingga status
	// yang punya dua tipe terhitung dua kali — 3 invoice dilaporkan 4.
	if semua.Total.Count != 3 {
		t.Errorf("total = %d, seharusnya 3 — status bertipe ganda terhitung dua kali?",
			semua.Total.Count)
	}
	if semua.Total.Amount != 7_000_000 {
		t.Errorf("total.amount = %d, seharusnya 7000000", semua.Total.Amount)
	}

	// --- disaring ke satu status ---------------------------------------------
	//
	// ByStatus harus TETAP menampilkan seluruh status (kalau tidak, tab lain
	// jadi nol), sedangkan ByType harus menyempit mengikuti pilihannya.
	draft := ambil(domain.InvoiceFilter{Status: "draft"})
	if got := draft.ByStatus["draft"].Count; got != 3 {
		t.Errorf("byStatus[draft] = %d, seharusnya 3", got)
	}
	if got := draft.ByType["renewal"].Count; got != 2 {
		t.Errorf("byType[renewal] saat status=draft = %d, seharusnya 2", got)
	}

	// Status yang tidak dipunyai satu pun baris: ByType harus KOSONG, bukan
	// diam-diam menampilkan angka seluruh status.
	lunas := ambil(domain.InvoiceFilter{Status: "paid"})
	if got := lunas.ByType["renewal"].Count; got != 0 {
		t.Errorf("byType[renewal] saat status=paid = %d, seharusnya 0 — "+
			"filter statusnya tidak diterapkan ke ByType", got)
	}
	// Tapi tabnya tetap harus tahu ada 3 draft.
	if got := lunas.ByStatus["draft"].Count; got != 3 {
		t.Errorf("byStatus[draft] saat status=paid = %d, seharusnya tetap 3 — "+
			"filter statusnya bocor ke ByStatus", got)
	}
}
