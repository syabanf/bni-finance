package report

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

const (
	chA = "ch-uji-laporan-a"
	chB = "ch-uji-laporan-b"
	chC = "ch-uji-laporan-c" // tanpa invoice sama sekali
)

var januari = domain.ReportFilter{From: "2026-01-01", To: "2026-01-31"}

// Data uji. Periode yang diuji adalah Januari 2026.
//
//	chapter A
//	  I1 renewal       1.000.000  terbit 10 Jan  lunas      dibayar 15 Jan, qris
//	  I2 pendaftaran   2.000.000  terbit 20 Jan  terkirim
//	  I3 renewal         500.000  terbit 25 Jan  draft      tidak ditagih
//	  I4 renewal         700.000  terbit 20 Des  lunas      dibayar 5 Jan, metode kosong
//	  I5 renewal         900.000  terbit 28 Jan  batal      dibayar 100.000 pada 29 Jan
//	  I6 renewal         300.000  terbit 31 Jan  overdue    dibayar 2 Feb
//	chapter B
//	  I7 renewal       4.000.000  terbit 12 Jan  lunas      dibayar 12 Jan
//
// Semua jam ditaruh di tengah hari supaya tanggalnya sama di zona waktu mana
// pun basis data uji dijalankan.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_test
func siapkan(t *testing.T) *Repository {
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

	ctx := context.Background()
	chapter := []string{chA, chB, chC}
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
		jalankan(`INSERT INTO chapters (id, name, display_name) VALUES ($1, $1, 'Uji ' || $1)`, ch)
	}
	jalankan(`INSERT INTO members (id, chapter_id, name, status) VALUES
		('mem-uji-laporan-a', $1, 'Sinta Ujilaporan', 'active'),
		('mem-uji-laporan-b', $2, 'Bagus Ujilaporan', 'active')`, chA, chB)

	invoice := []struct {
		nomor, member, chapter, tipe, status, terbit string
		nominal                                      int
		bayar                                        int
		tanggalBayar, metode                         string
	}{
		{"UJI-LAP-1", "mem-uji-laporan-a", chA, "renewal", "paid", "2026-01-10", 1_000_000, 1_000_000, "2026-01-15", "qris"},
		{"UJI-LAP-2", "mem-uji-laporan-a", chA, "registration", "sent", "2026-01-20", 2_000_000, 0, "", ""},
		{"UJI-LAP-3", "mem-uji-laporan-a", chA, "renewal", "draft", "2026-01-25", 500_000, 0, "", ""},
		{"UJI-LAP-4", "mem-uji-laporan-a", chA, "renewal", "paid", "2025-12-20", 700_000, 700_000, "2026-01-05", ""},
		{"UJI-LAP-5", "mem-uji-laporan-a", chA, "renewal", "cancelled", "2026-01-28", 900_000, 100_000, "2026-01-29", "bank_transfer"},
		{"UJI-LAP-6", "mem-uji-laporan-a", chA, "renewal", "overdue", "2026-01-31", 300_000, 300_000, "2026-02-02", "qris"},
		{"UJI-LAP-7", "mem-uji-laporan-b", chB, "renewal", "paid", "2026-01-12", 4_000_000, 4_000_000, "2026-01-12", "bank_transfer"},
	}
	for _, i := range invoice {
		jalankan(`
			INSERT INTO invoices (number, member_id, chapter_id, type, amount, due_date, period_start, period_end, status, created_at)
			VALUES ($1, $2, $3, $4::invoice_type, $5, '2026-02-15', '2026-02-15', '2027-02-15', $6::invoice_status,
			        ($7 || ' 12:00:00')::timestamp)`,
			i.nomor, i.member, i.chapter, i.tipe, i.nominal, i.status, i.terbit)
		if i.bayar > 0 {
			jalankan(`
				INSERT INTO payments (invoice_id, amount, payment_method, paid_at)
				SELECT id, $2, $3, ($4 || ' 12:00:00')::timestamp FROM invoices WHERE number = $1`,
				i.nomor, i.bayar, i.metode, i.tanggalBayar)
		}
	}
	return NewRepository(pool)
}

func baris(t *testing.T, s *domain.ReportSummary, id string) domain.ReportChapterRow {
	t.Helper()
	for _, c := range s.Chapters {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("chapter %s tidak ada di laporan", id)
	return domain.ReportChapterRow{}
}

// Angka per chapter untuk Januari, dihitung tangan dari tabel di atas.
func TestLaporanPerChapter(t *testing.T) {
	repo := siapkan(t)
	s, err := repo.Summary(scope.WithoutLimit(context.Background()), januari)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}

	a := baris(t, s, chA)
	ingin := domain.ReportChapterRow{
		ID: chA, Name: "Uji " + chA,
		Count:       3,         // I1, I2, I6. Draft, batal, dan yang terbit Desember tidak dihitung
		Ditagih:     3_300_000, // 1.000.000 + 2.000.000 + 300.000
		Renewal:     1_300_000,
		Pendaftaran: 2_000_000,
		Diterima:    1_800_000, // 1.000.000 + 700.000 (invoice Desember) + 100.000 (invoice batal)
		Outstanding: 2_300_000, // I2 terkirim + I6 overdue
		Rate:        55,        // 1,8 / 3,3 = 54,5%
	}
	if a != ingin {
		t.Errorf("chapter A:\n dapat %+v\n ingin %+v", a, ingin)
	}

	b := baris(t, s, chB)
	if b.Count != 1 || b.Ditagih != 4_000_000 || b.Diterima != 4_000_000 || b.Outstanding != 0 || b.Rate != 100 {
		t.Errorf("chapter B: %+v", b)
	}

	// Chapter tanpa invoice tetap tampil, dengan nol.
	c := baris(t, s, chC)
	if c.Count != 0 || c.Ditagih != 0 || c.Diterima != 0 || c.Rate != 0 {
		t.Errorf("chapter C seharusnya nol semua: %+v", c)
	}
}

// Total, tipe, bulan, dan metode. Dibatasi ke chapter A lewat lingkup ST supaya
// data lain di basis data uji tidak ikut terhitung, dan sekaligus menguji bahwa
// lingkup itu berlaku pada setiap bagian laporan.
func TestLaporanTotalMengikutiLingkup(t *testing.T) {
	repo := siapkan(t)
	s, err := repo.Summary(scope.WithChapter(context.Background(), chA), januari)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}

	if len(s.Chapters) != 1 || s.Chapters[0].ID != chA {
		t.Fatalf("ST chapter A melihat %d chapter", len(s.Chapters))
	}
	if s.Count != 3 || s.Ditagih != 3_300_000 || s.Diterima != 1_800_000 || s.Outstanding != 2_300_000 || s.Rate != 55 {
		t.Errorf("total: %d invoice, ditagih %d, diterima %d, outstanding %d, rate %d",
			s.Count, s.Ditagih, s.Diterima, s.Outstanding, s.Rate)
	}
	// Dari 1.800.000 yang diterima, hanya 1.000.000 milik invoice terbitan
	// Januari yang tidak batal. Sisanya untuk invoice Desember dan invoice batal.
	if s.DetailDiterima != 1_000_000 {
		t.Errorf("diterima untuk baris detail %d, ingin 1.000.000", s.DetailDiterima)
	}
	if s.Types[domain.TypeRenewal] != 2 || s.Types[domain.TypeRegistration] != 1 {
		t.Errorf("per tipe: %v, ingin renewal 2 dan registration 1", s.Types)
	}

	// Pembayaran Februari di luar periode, invoice Desember bukan terbitan
	// periode ini: hanya Januari yang muncul.
	if len(s.Monthly) != 1 || s.Monthly[0] != (domain.ReportMonthly{Month: "2026-01", Issued: 3_300_000, Paid: 1_800_000}) {
		t.Errorf("bulanan: %+v", s.Monthly)
	}

	inginMetode := []domain.ReportMethod{
		{Method: "qris", Count: 1, Amount: 1_000_000},
		{Method: "other", Count: 1, Amount: 700_000}, // metode kosong
		{Method: "bank_transfer", Count: 1, Amount: 100_000},
	}
	if len(s.Methods) != len(inginMetode) {
		t.Fatalf("metode: %+v", s.Methods)
	}
	for i, m := range inginMetode {
		if s.Methods[i] != m {
			t.Errorf("metode[%d] = %+v, ingin %+v", i, s.Methods[i], m)
		}
	}
}

// Saringan tipe berlaku pada invoice DAN pada pembayaran, lewat tipe invoice
// yang dibayar. Kalau hanya invoicenya yang tersaring, "Diterima" menghitung
// pembayaran renewal di laporan pendaftaran.
func TestLaporanSaringanTipe(t *testing.T) {
	repo := siapkan(t)
	f := januari
	f.Type = "registration"
	s, err := repo.Summary(scope.WithChapter(context.Background(), chA), f)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if s.Count != 1 || s.Ditagih != 2_000_000 || s.Outstanding != 2_000_000 || s.Diterima != 0 || s.Rate != 0 {
		t.Errorf("pendaftaran: %d invoice, ditagih %d, outstanding %d, diterima %d, rate %d",
			s.Count, s.Ditagih, s.Outstanding, s.Diterima, s.Rate)
	}
	if len(s.Methods) != 0 {
		t.Errorf("tidak ada pembayaran pendaftaran, tetapi metode berisi %+v", s.Methods)
	}
}

// Tanpa periode, seluruh riwayat terhitung, dan 52,5% dibulatkan ke 53.
func TestLaporanTanpaPeriode(t *testing.T) {
	repo := siapkan(t)
	s, err := repo.Summary(scope.WithChapter(context.Background(), chA), domain.ReportFilter{})
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	// I1, I2, I4, I6 ditagih. Diterima: 1.000.000 + 700.000 + 100.000 + 300.000.
	if s.Count != 4 || s.Ditagih != 4_000_000 || s.Diterima != 2_100_000 || s.Rate != 53 {
		t.Errorf("semua periode: %d invoice, ditagih %d, diterima %d, rate %d", s.Count, s.Ditagih, s.Diterima, s.Rate)
	}
}

// Detail: invoice yang terbit pada periode, terbaru dulu. Kolom diterima hanya
// menghitung pembayaran yang masuk pada periode, dan jumlahnya harus kembali
// ke angka chapternya.
func TestLaporanDetail(t *testing.T) {
	repo := siapkan(t)
	ctx := scope.WithChapter(context.Background(), chA)

	semua, total, err := repo.Detail(ctx, januari, 50, 0)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if total != 3 || len(semua) != 3 {
		t.Fatalf("detail: %d baris, total %d; ingin 3", len(semua), total)
	}
	urutan := semua[0].Number + "," + semua[1].Number + "," + semua[2].Number
	if urutan != "UJI-LAP-6,UJI-LAP-2,UJI-LAP-1" {
		t.Errorf("urutan %s, ingin terbaru dulu", urutan)
	}
	// I6 dibayar Februari, jadi nol pada periode Januari.
	if semua[0].Diterima != 0 || semua[1].Diterima != 0 || semua[2].Diterima != 1_000_000 {
		t.Errorf("diterima per baris: %d, %d, %d", semua[0].Diterima, semua[1].Diterima, semua[2].Diterima)
	}
	if semua[2].Member == nil || *semua[2].Member != "Sinta Ujilaporan" || semua[2].Chapter == nil {
		t.Errorf("nama member atau chapter kosong: %+v", semua[2])
	}

	var ditagih int64
	for _, d := range semua {
		ditagih += d.Ditagih
	}
	if ditagih != 3_300_000 {
		t.Errorf("jumlah ditagih di detail %d, ingin 3.300.000 seperti baris chapternya", ditagih)
	}

	halaman, total, err := repo.Detail(ctx, januari, 2, 2)
	if err != nil {
		t.Fatalf("Detail halaman kedua: %v", err)
	}
	if total != 3 || len(halaman) != 1 || halaman[0].Number != "UJI-LAP-1" {
		t.Errorf("halaman kedua: %d baris, total %d", len(halaman), total)
	}
}

// Pengguna tanpa lingkup sama sekali tidak mendapat apa pun.
func TestLaporanTanpaLingkup(t *testing.T) {
	repo := siapkan(t)
	s, err := repo.Summary(context.Background(), januari)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if len(s.Chapters) != 0 || s.Ditagih != 0 || s.Diterima != 0 {
		t.Errorf("context tanpa lingkup melihat %d chapter, ditagih %d", len(s.Chapters), s.Ditagih)
	}
}
