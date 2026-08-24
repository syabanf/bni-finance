package invoice

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Ketiga cacat di berkas ini lolos dari seluruh unit test karena stub store
// tidak punya foreign key, tidak punya tabel members, dan tidak punya batas
// lebar kolom. Semuanya hanya terlihat terhadap Postgres sungguhan:
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
func livePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL tidak diset — integration test dilewati")
	}
	name := url[strings.LastIndex(url, "/")+1:]
	if !strings.Contains(name, "test") && !strings.Contains(name, "dev") {
		t.Fatalf("menolak berjalan atas basis data %q — namanya tidak mengandung test/dev", name)
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("sambung basis data: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func tgl(s string) domain.Date {
	d, err := domain.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func contohInput(memberID, chapterID string, amount int64) domain.CreateInvoiceInput {
	return domain.CreateInvoiceInput{
		MemberID:    memberID,
		ChapterID:   chapterID,
		Type:        domain.TypeRenewal,
		Amount:      amount,
		DueDate:     tgl("2026-12-01"),
		PeriodStart: tgl("2026-12-01"),
		PeriodEnd:   tgl("2027-12-01"),
	}
}

// Penghapusan invoice TIDAK PERNAH berhasil sekali pun.
//
// Setiap invoice punya baris invoice_audit_log sejak detik ia dibuat, dan FK-nya
// tidak punya klausa on delete — jadi setiap DELETE menabrak
// invoice_audit_log_invoice_id_fkey dan keluar sebagai 500. Termasuk seluruh
// data contoh. Stub store tidak punya foreign key, jadi unit test-nya hijau.
func TestHapusInvoiceIkutMembuangJejakAudit(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	// Lingkup dinyatakan EKSPLISIT. scope.Chapter gagal tertutup, jadi context
	// polos berarti "tidak boleh melihat apa pun" — dan tes ini memang menguji
	// perilaku tanpa batas chapter, bukan perilaku ST.
	ctx := scope.WithoutLimit(context.Background())

	var memberID, chapterID string
	if err := pool.QueryRow(ctx,
		"SELECT id, chapter_id FROM members LIMIT 1").Scan(&memberID, &chapterID); err != nil {
		t.Skipf("tidak ada member contoh: %v", err)
	}

	inv, err := repo.Create(ctx, contohInput(memberID, chapterID, 1_500_000), "", "IDR")
	if err != nil {
		t.Fatalf("buat invoice: %v", err)
	}

	// Prasyarat tesnya: jejak auditnya memang ada. Tanpa ini tes tetap hijau
	// meski FK-nya tidak pernah tersentuh, dan bug aslinya bisa kembali diam-diam.
	var jejak int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM invoice_audit_log WHERE invoice_id = $1", inv.ID).Scan(&jejak); err != nil {
		t.Fatalf("hitung jejak audit: %v", err)
	}
	if jejak == 0 {
		t.Fatal("invoice baru tidak punya jejak audit — tes ini tidak menguji apa pun")
	}

	if err := repo.Delete(ctx, inv.ID); err != nil {
		t.Fatalf("hapus invoice: %v", err)
	}
	if _, err := repo.GetByID(ctx, inv.ID); err == nil {
		t.Error("invoice masih ada setelah dihapus")
	}
	var sisa int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM invoice_audit_log WHERE invoice_id = $1", inv.ID).Scan(&sisa); err != nil {
		t.Fatalf("hitung sisa jejak: %v", err)
	}
	if sisa != 0 {
		t.Errorf("%d jejak audit tertinggal menunjuk invoice yang sudah tidak ada", sisa)
	}
}

// Chapter invoice harus chapter membernya.
//
// Keduanya datang terpisah dari klien. Tanpa pemeriksaan, invoice untuk member
// BNI Nusantara bisa dicatat atas nama BNI Garuda dan dijawab 201 — pendapatan
// per chapter salah hitung selamanya, tanpa satu pun tanda pada invoicenya.
func TestChapterHarusCocokDenganMember(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	// Lingkup dinyatakan EKSPLISIT. scope.Chapter gagal tertutup, jadi context
	// polos berarti "tidak boleh melihat apa pun" — dan tes ini memang menguji
	// perilaku tanpa batas chapter, bukan perilaku ST.
	ctx := scope.WithoutLimit(context.Background())

	var memberID, chapterMember, chapterLain string
	if err := pool.QueryRow(ctx,
		"SELECT id, chapter_id FROM members LIMIT 1").Scan(&memberID, &chapterMember); err != nil {
		t.Skipf("tidak ada member contoh: %v", err)
	}
	if err := pool.QueryRow(ctx,
		"SELECT id FROM chapters WHERE id <> $1 LIMIT 1", chapterMember).Scan(&chapterLain); err != nil {
		t.Skipf("butuh minimal dua chapter: %v", err)
	}

	inv, err := repo.Create(ctx, contohInput(memberID, chapterLain, 1_500_000), "", "IDR")
	if err == nil {
		_ = repo.Delete(ctx, inv.ID)
		t.Fatalf("invoice diterima dengan chapter %q padahal membernya di %q", chapterLain, chapterMember)
	}
	if httpx.StatusOf(err) != 400 {
		t.Errorf("status = %d, mau 400 — ini kesalahan pemanggil, bukan server: %v",
			httpx.StatusOf(err), err)
	}
}

// memberId yang tidak ada adalah kesalahan PEMANGGIL, dan harus terbaca begitu.
//
// Dulu pelanggaran foreign key lolos sebagai 500 "terjadi kesalahan pada
// server" — pesan yang menyuruh orang mencari kerusakan server yang tidak
// pernah ada. Kelas yang sama pernah menipu pada blackbox.
func TestMemberTidakAdaAdalah400BukanS500(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	var chapterID string
	if err := pool.QueryRow(context.Background(),
		"SELECT id FROM chapters LIMIT 1").Scan(&chapterID); err != nil {
		t.Skipf("tidak ada chapter contoh: %v", err)
	}

	_, err := repo.Create(scope.WithoutLimit(context.Background()),
		contohInput("member-yang-tidak-pernah-ada", chapterID, 1_500_000), "", "IDR")
	if err == nil {
		t.Fatal("invoice dibuat untuk member yang tidak ada")
	}
	if got := httpx.StatusOf(err); got != 400 {
		t.Errorf("status = %d, mau 400: %v", got, err)
	}
}

// Nominal di atas 2,1 miliar dulu menabrak batas int4 dan keluar sebagai 500.
// Kolomnya kini bigint, jadi angka sebesar itu harus tersimpan biasa saja.
func TestNominalDiAtasBatasInt4Tersimpan(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	// Lingkup dinyatakan EKSPLISIT. scope.Chapter gagal tertutup, jadi context
	// polos berarti "tidak boleh melihat apa pun" — dan tes ini memang menguji
	// perilaku tanpa batas chapter, bukan perilaku ST.
	ctx := scope.WithoutLimit(context.Background())

	var memberID, chapterID string
	if err := pool.QueryRow(ctx,
		"SELECT id, chapter_id FROM members LIMIT 1").Scan(&memberID, &chapterID); err != nil {
		t.Skipf("tidak ada member contoh: %v", err)
	}

	const besar int64 = 5_000_000_000 // di atas int4, di bawah MaxInvoiceAmount
	inv, err := repo.Create(ctx, contohInput(memberID, chapterID, besar), "", "IDR")
	if err != nil {
		t.Fatalf("nominal %d ditolak: %v", besar, err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, inv.ID) })
	if inv.Amount != besar {
		t.Errorf("amount tersimpan = %d, mau %d", inv.Amount, besar)
	}
}
