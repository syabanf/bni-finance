package invoice

import (
	"context"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Hanya invoice `sent` yang lewat jatuh tempo yang berubah, dan tiap perubahan
// meninggalkan satu baris audit.
func TestTandaiOverdue(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB, memA, memB := duaChapter(t, repo)
	ctx := scope.WithoutLimit(context.Background())

	buat := func(member, chapter, status, jatuhTempo string) string {
		t.Helper()
		inv, err := repo.Create(ctx, contohInput(member, chapter, 1_000_000), "", "IDR")
		if err != nil {
			t.Fatalf("buat invoice: %v", err)
		}
		if _, err := pool.Exec(ctx,
			`UPDATE invoices SET status = $2::invoice_status, due_date = $3::date WHERE id = $1`,
			inv.ID, status, jatuhTempo); err != nil {
			t.Fatalf("atur invoice: %v", err)
		}
		return inv.ID
	}
	telat := buat(memA, chA, "sent", "2020-01-01")
	belumJatuhTempo := buat(memB, chB, "sent", "2999-01-01")

	n, err := repo.TandaiOverdue(ctx)
	if err != nil {
		t.Fatalf("TandaiOverdue: %v", err)
	}
	if n != 1 {
		t.Errorf("ditandai %d invoice, ingin 1", n)
	}

	status := func(id string) (s domain.InvoiceStatus) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT status FROM invoices WHERE id = $1`, id).Scan(&s); err != nil {
			t.Fatalf("baca status: %v", err)
		}
		return s
	}
	if status(telat) != domain.StatusOverdue {
		t.Errorf("invoice telat berstatus %s, ingin overdue", status(telat))
	}
	if status(belumJatuhTempo) != domain.StatusSent {
		t.Errorf("invoice yang belum jatuh tempo berstatus %s, ingin sent", status(belumJatuhTempo))
	}

	var jejak int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM invoice_audit_log
		WHERE invoice_id = $1 AND action = 'overdue' AND old_status = 'sent' AND new_status = 'overdue'`,
		telat).Scan(&jejak); err != nil {
		t.Fatalf("baca audit: %v", err)
	}
	if jejak != 1 {
		t.Errorf("baris audit overdue = %d, ingin 1", jejak)
	}

	// Sapuan kedua tidak menemukan apa pun dan tidak menggandakan audit.
	if n, err := repo.TandaiOverdue(ctx); err != nil || n != 0 {
		t.Errorf("sapuan kedua menandai %d invoice (galat %v), ingin 0", n, err)
	}
}

// Invoice lunas, draft, dan batal tidak pernah menjadi overdue walau tanggal
// jatuh temponya sudah lewat.
func TestTandaiOverdueTidakMenyentuhStatusLain(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, _, memA, _ := duaChapter(t, repo)
	ctx := scope.WithoutLimit(context.Background())

	inv, err := repo.Create(ctx, contohInput(memA, chA, 1_000_000), "", "IDR")
	if err != nil {
		t.Fatalf("buat invoice: %v", err)
	}
	for _, s := range []string{"draft", "paid", "cancelled"} {
		if _, err := pool.Exec(ctx,
			`UPDATE invoices SET status = $2::invoice_status, due_date = '2020-01-01' WHERE id = $1`, inv.ID, s); err != nil {
			t.Fatalf("atur status %s: %v", s, err)
		}
		if _, err := repo.TandaiOverdue(ctx); err != nil {
			t.Fatalf("TandaiOverdue: %v", err)
		}
		var kini string
		if err := pool.QueryRow(ctx, `SELECT status FROM invoices WHERE id = $1`, inv.ID).Scan(&kini); err != nil {
			t.Fatalf("baca status: %v", err)
		}
		if kini != s {
			t.Errorf("invoice %s berubah menjadi %s", s, kini)
		}
	}
}
