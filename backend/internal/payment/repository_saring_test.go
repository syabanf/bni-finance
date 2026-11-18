package payment

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Dua chapter, dua member, tiga invoice, tiga pembayaran:
//
//	chapter A  Sinta   renewal       1.000.000
//	chapter A  Sinta   registration  2.000.000
//	chapter B  Bagus   renewal       4.000.000
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_test
func siapkan(t *testing.T) (*Repository, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL tidak diset — integration test dilewati")
	}
	if nama := url[strings.LastIndex(url, "/")+1:]; !strings.Contains(nama, "test") && !strings.Contains(nama, "dev") {
		t.Fatalf("menolak berjalan atas basis data %q — namanya tidak mengandung test/dev", nama)
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("sambung basis data: %v", err)
	}
	t.Cleanup(pool.Close)
	testdb.Serialize(t, pool)

	ctx := scope.WithoutLimit(context.Background())
	chapter := []string{"ch-uji-bayar-a", "ch-uji-bayar-b"}
	bersih := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM payments WHERE invoice_id IN (SELECT id FROM invoices WHERE chapter_id = ANY($1))`, chapter)
		_, _ = pool.Exec(ctx, `DELETE FROM invoice_audit_log WHERE invoice_id IN (SELECT id FROM invoices WHERE chapter_id = ANY($1))`, chapter)
		_, _ = pool.Exec(ctx, `DELETE FROM invoices WHERE chapter_id = ANY($1)`, chapter)
		_, _ = pool.Exec(ctx, `DELETE FROM members WHERE chapter_id = ANY($1)`, chapter)
		_, _ = pool.Exec(ctx, `DELETE FROM chapters WHERE id = ANY($1)`, chapter)
	}
	bersih()
	t.Cleanup(bersih)

	jalankan := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("siapkan data: %v\n%s", err, sql)
		}
	}
	for _, ch := range chapter {
		jalankan(`INSERT INTO chapters (id, name, display_name) VALUES ($1, $1, $1)`, ch)
	}
	jalankan(`INSERT INTO members (id, chapter_id, name, status) VALUES
		('mem-uji-bayar-sinta', 'ch-uji-bayar-a', 'Sinta Ujibayar', 'active'),
		('mem-uji-bayar-bagus', 'ch-uji-bayar-b', 'Bagus Ujibayar', 'active')`)
	invoice := []struct {
		nomor, member, chapter, tipe string
		nominal                      int
	}{
		{"UJI-BAYAR-001", "mem-uji-bayar-sinta", "ch-uji-bayar-a", "renewal", 1_000_000},
		{"UJI-BAYAR-002", "mem-uji-bayar-sinta", "ch-uji-bayar-a", "registration", 2_000_000},
		{"UJI-BAYAR-003", "mem-uji-bayar-bagus", "ch-uji-bayar-b", "renewal", 4_000_000},
	}
	for _, i := range invoice {
		jalankan(`
			WITH baru AS (
				INSERT INTO invoices (number, member_id, chapter_id, type, amount, due_date, period_start, period_end, status)
				VALUES ($1, $2, $3, $4::invoice_type, $5, '2026-12-01', '2026-12-01', '2027-12-01', 'paid')
				RETURNING id)
			INSERT INTO payments (invoice_id, amount, payment_method, paid_at)
			SELECT id, $5, 'transfer', '2026-11-10T03:00:00Z' FROM baru`,
			i.nomor, i.member, i.chapter, i.tipe, i.nominal)
	}
	return NewRepository(pool), ctx
}

// Saringan uji selalu dibatasi ke dua chapter tes lewat pencarian "Ujibayar"
// atau nomor "UJI-BAYAR", supaya pembayaran lain di basis data tidak terhitung.
func TestSaringanPembayaran(t *testing.T) {
	repo, ctx := siapkan(t)

	kasus := []struct {
		nama        string
		f           domain.PaymentFilter
		inginBaris  int
		inginJumlah int64
	}{
		{"cari nama member", domain.PaymentFilter{Search: "sinta ujibayar"}, 2, 3_000_000},
		{"cari nomor invoice", domain.PaymentFilter{Search: "UJI-BAYAR-003"}, 1, 4_000_000},
		{"per chapter", domain.PaymentFilter{Search: "UJI-BAYAR", ChapterID: "ch-uji-bayar-a"}, 2, 3_000_000},
		{"per tipe", domain.PaymentFilter{Search: "UJI-BAYAR", Type: "renewal"}, 2, 5_000_000},
		{"chapter dan tipe", domain.PaymentFilter{Search: "UJI-BAYAR", ChapterID: "ch-uji-bayar-a", Type: "renewal"}, 1, 1_000_000},
		{"di luar rentang tanggal", domain.PaymentFilter{Search: "UJI-BAYAR", PaidFrom: "2026-12-01"}, 0, 0},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			k.f.Limit = 50
			items, total, err := repo.List(ctx, k.f)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(items) != k.inginBaris || total != k.inginBaris {
				t.Errorf("baris = %d, total = %d, ingin %d", len(items), total, k.inginBaris)
			}
			ringkas, err := repo.Summary(ctx, k.f)
			if err != nil {
				t.Fatalf("Summary: %v", err)
			}
			if ringkas.Count != k.inginBaris || ringkas.Amount != k.inginJumlah {
				t.Errorf("ringkasan = %d baris Rp%d, ingin %d baris Rp%d", ringkas.Count, ringkas.Amount, k.inginBaris, k.inginJumlah)
			}
		})
	}
}

// Ringkasan menjumlahkan seluruh hasil filter, bukan halaman yang diminta, dan
// memecahnya menurut tipe invoice.
func TestRingkasanPembayaranTidakIkutHalaman(t *testing.T) {
	repo, ctx := siapkan(t)
	f := domain.PaymentFilter{Search: "UJI-BAYAR", Limit: 1}

	items, total, err := repo.List(ctx, f)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || total != 3 {
		t.Fatalf("halaman = %d baris, total = %d; ingin 1 dan 3", len(items), total)
	}
	ringkas, err := repo.Summary(ctx, f)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if ringkas.Amount != 7_000_000 {
		t.Errorf("jumlah = %d, ingin 7.000.000", ringkas.Amount)
	}
	if ringkas.ByType["renewal"] != 5_000_000 || ringkas.ByType["registration"] != 2_000_000 {
		t.Errorf("per tipe = %v, ingin renewal 5.000.000 dan registration 2.000.000", ringkas.ByType)
	}

	// Daftar metode tidak ikut menyempit saat metodenya disaring.
	f.Method = "tidak-ada"
	ringkas, err = repo.Summary(ctx, f)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if ringkas.Count != 0 || len(ringkas.Methods) != 1 || ringkas.Methods[0] != "transfer" {
		t.Errorf("metode = %v dengan %d baris, ingin [transfer] dengan 0 baris", ringkas.Methods, ringkas.Count)
	}
}

// ST hanya melihat pembayaran chapternya, juga di ringkasan.
func TestRingkasanPembayaranMengikutiLingkupChapter(t *testing.T) {
	repo, _ := siapkan(t)
	ctx := scope.WithChapter(context.Background(), "ch-uji-bayar-b")

	ringkas, err := repo.Summary(ctx, domain.PaymentFilter{Search: "UJI-BAYAR"})
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if ringkas.Count != 1 || ringkas.Amount != 4_000_000 {
		t.Errorf("ringkasan ST chapter B = %d baris Rp%d, ingin 1 baris Rp4.000.000", ringkas.Count, ringkas.Amount)
	}
}
