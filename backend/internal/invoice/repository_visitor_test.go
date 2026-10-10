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

// TIPE INVOICE MENGIKUTI STATUS: VISITOR HANYA PENDAFTARAN, MEMBER HANYA RENEWAL.
//
// Renewal memperpanjang keanggotaan yang sudah ada; tamu belum punya itu.
// Jalannya ke dalam adalah invoice pendaftaran, dan justru itu yang harus
// tetap boleh: menolak keduanya berarti tamu tidak pernah bisa menjadi
// anggota lewat sistem ini. Sebaliknya, member aktif yang ditagih pendaftaran
// adalah tagihan ganda dengan nama yang salah; jalannya adalah renewal.
func TestTipeInvoiceMengikutiStatusMember(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	ctx := scope.WithoutLimit(context.Background())

	const ch, tamu, aktif = "ch-uji-tamu", "mem-uji-tamu", "mem-uji-aktif"
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
	if _, err := pool.Exec(ctx, `
		INSERT INTO members (id, chapter_id, name, status, joined_date)
		VALUES ($1,$2,'Anggota Uji','active'::member_status, CURRENT_DATE)
		ON CONFLICT (id) DO UPDATE SET status = 'active'`, aktif, ch); err != nil {
		t.Fatalf("siapkan member aktif: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, "DELETE FROM invoice_audit_log WHERE invoice_id IN (SELECT id FROM invoices WHERE member_id IN ($1,$2))", tamu, aktif)
		_, _ = pool.Exec(c, "DELETE FROM invoices WHERE member_id IN ($1,$2)", tamu, aktif)
		_, _ = pool.Exec(c, "DELETE FROM members WHERE id IN ($1,$2)", tamu, aktif)
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
	// Pendaftaran hanya sekali: yang kedua ditolak selama yang pertama belum dibatalkan.
	if _, err := repo.Create(ctx, pendaftaran, "", "IDR"); httpx.StatusOf(err) != 400 || !strings.Contains(err.Error(), "sudah punya invoice pendaftaran") {
		t.Errorf("pendaftaran kedua harus ditolak 400, dapat %v", err)
	}

	daftarUlang := contohInput(aktif, ch, 18_000_000)
	daftarUlang.Type = domain.TypeRegistration
	if _, err := repo.Create(ctx, daftarUlang, "", "IDR"); err == nil {
		t.Fatal("pendaftaran untuk member aktif diterima")
	} else {
		if httpx.StatusOf(err) != 400 {
			t.Errorf("status = %d, mau 400: %v", httpx.StatusOf(err), err)
		}
		if !strings.Contains(err.Error(), "renewal") {
			t.Errorf("pesannya harus menunjuk jalan keluarnya (invoice renewal): %v", err)
		}
	}
	konfirmasiRenewal(t, pool, aktif)
	if _, err := repo.Create(ctx, contohInput(aktif, ch, 12_700_000), "", "IDR"); err != nil {
		t.Fatalf("renewal untuk member aktif harus diterima: %v", err)
	}
}
