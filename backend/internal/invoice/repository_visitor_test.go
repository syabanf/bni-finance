package invoice

import (
	"context"
	"strings"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// VISITOR TIDAK BISA DITAGIH RENEWAL, TAPI BISA DITAGIH PENDAFTARAN.
//
// Renewal memperpanjang keanggotaan yang sudah ada; tamu belum punya itu.
// Jalannya ke dalam adalah invoice pendaftaran, dan justru itu yang harus
// tetap boleh: menolak keduanya berarti tamu tidak pernah bisa menjadi
// anggota lewat sistem ini.
func TestVisitorHanyaBisaDitagihPendaftaran(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	ctx := scope.WithoutLimit(context.Background())

	const ch, tamu = "ch-uji-tamu", "mem-uji-tamu"
	if _, err := pool.Exec(ctx, `
		INSERT INTO chapters (id, name, display_name, area_name, city_name)
		VALUES ($1,'ujitamu','Uji Tamu','Uji','Uji') ON CONFLICT (id) DO NOTHING`, ch); err != nil {
		t.Fatalf("siapkan chapter: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO members (id, chapter_id, name, status, joined_date)
		VALUES ($1,$2,'Tamu Uji','visitor'::member_status, CURRENT_DATE)
		ON CONFLICT (id) DO UPDATE SET status = 'visitor'`, tamu, ch); err != nil {
		t.Fatalf("siapkan member: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, "DELETE FROM invoice_audit_log WHERE invoice_id IN (SELECT id FROM invoices WHERE member_id = $1)", tamu)
		_, _ = pool.Exec(c, "DELETE FROM invoices WHERE member_id = $1", tamu)
		_, _ = pool.Exec(c, "DELETE FROM members WHERE id = $1", tamu)
		_, _ = pool.Exec(c, "DELETE FROM chapters WHERE id = $1", ch)
	})

	renewal := contohInput(tamu, ch, 12_700_000)
	if _, err := repo.Create(ctx, renewal, "", "IDR"); err == nil {
		t.Fatal("renewal untuk visitor diterima")
	} else {
		if httpx.StatusOf(err) != 400 {
			t.Errorf("status = %d, mau 400: %v", httpx.StatusOf(err), err)
		}
		if !strings.Contains(err.Error(), "pendaftaran") {
			t.Errorf("pesannya harus menunjuk jalan keluarnya (invoice pendaftaran): %v", err)
		}
	}

	pendaftaran := contohInput(tamu, ch, 18_000_000)
	pendaftaran.Type = domain.TypeRegistration
	inv, err := repo.Create(ctx, pendaftaran, "", "IDR")
	if err != nil {
		t.Fatalf("pendaftaran untuk visitor harus diterima: %v", err)
	}
	if inv.Type != domain.TypeRegistration {
		t.Errorf("tipe tersimpan %q", inv.Type)
	}
}
