package invoice

import (
	"context"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Halaman pembayaran dan Renewal Due mengambil invoice milik baris yang sedang
// tampil saja, lewat ?ids= dan ?memberIds=.
func TestSaringInvoicePerIdDanPerMember(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB, memA, memB := duaChapter(t, repo)
	nasional := scope.WithoutLimit(context.Background())

	buat := func(member, chapter string) string {
		t.Helper()
		inv, err := repo.Create(nasional, contohInput(member, chapter, 1_000_000), "", "IDR")
		if err != nil {
			t.Fatalf("buat invoice: %v", err)
		}
		return inv.ID
	}
	idA, idB := buat(memA, chA), buat(memB, chB)

	perID, total, err := repo.List(nasional, domain.InvoiceFilter{IDs: []string{idA}, Limit: 50})
	if err != nil {
		t.Fatalf("List per id: %v", err)
	}
	if total != 1 || len(perID) != 1 || perID[0].ID != idA {
		t.Errorf("per id mengembalikan %d baris, ingin tepat invoice A", len(perID))
	}

	perMember, total, err := repo.List(nasional, domain.InvoiceFilter{MemberIDs: []string{memB, "tidak-ada"}, Limit: 50})
	if err != nil {
		t.Fatalf("List per member: %v", err)
	}
	if total != 1 || len(perMember) != 1 || perMember[0].ID != idB {
		t.Errorf("per member mengembalikan %d baris, ingin tepat invoice B", len(perMember))
	}

	// Lingkup chapter tetap di atas saringan id: ST chapter A yang meminta
	// invoice chapter B tidak mendapat apa pun.
	stA := scope.WithChapter(context.Background(), chA)
	bocor, _, err := repo.List(stA, domain.InvoiceFilter{IDs: []string{idB}, Limit: 50})
	if err != nil {
		t.Fatalf("List ST: %v", err)
	}
	if len(bocor) != 0 {
		t.Errorf("ST chapter A melihat %d invoice chapter B lewat ?ids=", len(bocor))
	}
}
