package invoice

import (
	"context"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Laporan Pengiriman membaca invoice menurut tanggal kirimnya. Mengisi
// saringan itu membuang invoice yang belum pernah dikirim, dan urutan "sent"
// menaruh kiriman terbaru di atas.
func TestSaringDanUrutMenurutTanggalKirim(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB, memA, memB := duaChapter(t, repo)
	ctx := scope.WithoutLimit(context.Background())

	buat := func(member, chapter string, dikirim *string) string {
		t.Helper()
		inv, err := repo.Create(ctx, contohInput(member, chapter, 1_000_000), "", "IDR")
		if err != nil {
			t.Fatalf("buat invoice: %v", err)
		}
		if _, err := pool.Exec(ctx,
			`UPDATE invoices SET paper_id_sent_at = ($2 || ' 12:00:00')::timestamp WHERE id = $1`, inv.ID, dikirim); err != nil {
			t.Fatalf("atur tanggal kirim: %v", err)
		}
		return inv.ID
	}
	maret, mei := "2031-03-10", "2031-05-20"
	lama := buat(memA, chA, &maret)
	baru := buat(memB, chB, &mei)

	// Yang belum pernah dikirim tidak ikut, walau tidak ada batas akhir.
	f := domain.InvoiceFilter{MemberIDs: []string{memA, memB}, SentFrom: "2031-01-01", Sort: "sent", Limit: 50}
	items, total, err := repo.List(ctx, f)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 || len(items) != 2 || items[0].ID != baru || items[1].ID != lama {
		t.Fatalf("dapat %d baris, ingin kiriman Mei lalu Maret", len(items))
	}

	// Batas akhir inklusif: tanggal kirim 10 Maret masuk saat sentTo 10 Maret.
	f.SentTo = "2031-03-10"
	items, _, err = repo.List(ctx, f)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].ID != lama {
		t.Errorf("sentTo 10 Maret mengembalikan %d baris, ingin hanya kiriman Maret", len(items))
	}
}
