package dashboard

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Dashboard adalah halaman PERTAMA setelah login, dan sebelum tes ini ada ia
// tidak membatasi apa pun.
//
// Sudah dibuktikan dengan pengguna ST sungguhan: daftar invoicenya benar —
// hanya chapter dia — tetapi dashboard menampilkan setiap chapter lengkap
// dengan nama dan nominalnya, plus total nasional. Batas yang ditegakkan di
// satu halaman dan bocor di halaman berikutnya bukan batas.
//
// Tes ini MEMBUAT dunianya sendiri, tidak membaca data contoh. Alasannya sudah
// terbukti mahal di paket invoice: integration test lain men-TRUNCATE tabel,
// sehingga tes yang bergantung pada data yang ada akan memanggil t.Skip dan
// `go test` melaporkannya "ok" — tes keamanan yang hijau tanpa menguji apa pun.
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

// duaChapterBerinvoice menyiapkan dua chapter, masing-masing satu member,
// satu invoice, dan — untuk chapter B — satu pembayaran.
//
// Pembayaran itu penting: tabel payments TIDAK punya chapter_id sendiri, jadi
// lingkupnya harus diwarisi lewat invoice induknya. Tanpa baris ini, sabotase
// pada jalur pembayaran tidak akan terlihat sama sekali.
func duaChapterBerinvoice(t *testing.T, db *pgxpool.Pool) (chA, chB string) {
	t.Helper()
	ctx := context.Background()
	chA, chB = "ch-dash-a", "ch-dash-b"

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, q, args...); err != nil {
			t.Fatalf("siapkan data: %v", err)
		}
	}

	for i, ch := range []string{chA, chB} {
		exec(`INSERT INTO chapters (id, name, display_name, area_name, city_name)
		      VALUES ($1,$2,$3,'Uji','Uji') ON CONFLICT (id) DO NOTHING`,
			ch, "dash"+string(rune('a'+i)), "Dash "+string(rune('A'+i)))
		exec(`INSERT INTO members (id, chapter_id, name, email, phone, status, renewal_date)
		      VALUES ($1,$2,$3,'dash@contoh.invalid','080000000000','active', CURRENT_DATE + 10)
		      ON CONFLICT (id) DO NOTHING`,
			"mem-dash-"+ch, ch, "Member "+ch)
		exec(`INSERT INTO invoices (id, number, member_id, chapter_id, type, amount, currency,
		                           due_date, period_start, period_end, status)
		      VALUES ($1,$2,$3,$4,'renewal',$5,'IDR',
		              CURRENT_DATE + 30, CURRENT_DATE, CURRENT_DATE + 365,'sent')
		      ON CONFLICT (number) DO NOTHING`,
			"11111111-1111-1111-1111-11111111111"+string(rune('0'+i)),
			"DASH-"+strings.ToUpper(ch), "mem-dash-"+ch, ch, (i+1)*1_000_000)
	}
	exec(`INSERT INTO payments (id, invoice_id, amount, paid_at, payment_method)
	      SELECT '22222222-2222-2222-2222-222222222222', id, 2000000, now(), 'transfer'
	      FROM invoices WHERE number = $1 ON CONFLICT (id) DO NOTHING`, "DASH-CH-DASH-B")

	t.Cleanup(func() {
		c := context.Background()
		_, _ = db.Exec(c, `DELETE FROM payments WHERE invoice_id IN
		                   (SELECT id FROM invoices WHERE chapter_id = ANY($1))`, []string{chA, chB})
		_, _ = db.Exec(c, `DELETE FROM invoices WHERE chapter_id = ANY($1)`, []string{chA, chB})
		_, _ = db.Exec(c, `DELETE FROM members WHERE chapter_id = ANY($1)`, []string{chA, chB})
		_, _ = db.Exec(c, `DELETE FROM chapters WHERE id = ANY($1)`, []string{chA, chB})
	})
	return chA, chB
}

func TestDashboardTidakBocorAntarChapter(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB := duaChapterBerinvoice(t, pool)
	stA := scope.WithChapter(context.Background(), chA)

	ringkasan, err := repo.Summary(stA, 6)
	if err != nil {
		t.Fatalf("ringkasan sebagai ST: %v", err)
	}

	// --- chapterStats: chapter lain tidak boleh disebut sama sekali ----------
	for _, c := range ringkasan.ChapterStats {
		if c.ChapterID == chB {
			t.Errorf("ST chapter %s melihat statistik chapter %s (nominal %d)",
				chA, c.ChapterID, c.TotalAmount)
		}
	}

	// --- total: nominal chapter B tidak boleh ikut terjumlah ----------------
	//
	// Chapter A bernominal 1 juta, chapter B 2 juta. Kalau totalnya menyentuh
	// 3 juta, batasnya tidak menggigit.
	if ringkasan.Total.Amount >= 3_000_000 {
		t.Errorf("total = %d, ikut menghitung chapter lain", ringkasan.Total.Amount)
	}

	// --- monthly: jalur invoice DAN jalur pembayaran ------------------------
	var terbit, terkumpul int64
	for _, m := range ringkasan.Monthly {
		terbit += m.Issued
		terkumpul += m.Paid
	}
	if terbit >= 3_000_000 {
		t.Errorf("monthly.issued = %d, ikut menghitung chapter lain", terbit)
	}
	// Satu-satunya pembayaran ada di chapter B; ST chapter A harus melihat nol.
	if terkumpul != 0 {
		t.Errorf("monthly.paid = %d, membocorkan pembayaran chapter lain "+
			"(payments tidak punya chapter_id — lingkupnya harus lewat invoice induk)", terkumpul)
	}

	// --- statusBreakdown ----------------------------------------------------
	var jumlahStatus int
	for _, s := range ringkasan.StatusBreakdown {
		jumlahStatus += s.Count
	}
	if jumlahStatus > 1 {
		t.Errorf("statusBreakdown menghitung %d invoice, seharusnya 1", jumlahStatus)
	}
}

// Admin tidak boleh ikut terpotong. Pembatasan yang kelewat rajin memutus
// laporan nasional, dan itu rusak dengan cara yang sama diamnya.
func TestDashboardAdminTetapMelihatSemua(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB := duaChapterBerinvoice(t, pool)
	nasional := scope.WithoutLimit(context.Background())

	ringkasan, err := repo.Summary(nasional, 6)
	if err != nil {
		t.Fatalf("ringkasan sebagai admin: %v", err)
	}

	terlihat := map[string]bool{}
	for _, c := range ringkasan.ChapterStats {
		terlihat[c.ChapterID] = true
	}
	for _, ch := range []string{chA, chB} {
		if !terlihat[ch] {
			t.Errorf("admin tidak melihat chapter %s — pembatasannya kelewat jauh", ch)
		}
	}
	if ringkasan.Total.Amount < 3_000_000 {
		t.Errorf("total admin = %d, seharusnya mencakup kedua chapter", ringkasan.Total.Amount)
	}
}

// Konteks tanpa scope sama sekali harus GAGAL TERTUTUP, bukan terbuka.
//
// Inilah yang membedakan kelalaian dari kebocoran: handler baru yang lupa
// memasang scope harus menghasilkan layar kosong, bukan data seluruh negeri.
func TestDashboardTanpaScopeGagalTertutup(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	duaChapterBerinvoice(t, pool)

	ringkasan, err := repo.Summary(context.Background(), 6)
	if err != nil {
		t.Fatalf("ringkasan tanpa scope: %v", err)
	}
	if len(ringkasan.ChapterStats) != 0 {
		t.Errorf("konteks tanpa scope mengembalikan %d chapter — seharusnya nol",
			len(ringkasan.ChapterStats))
	}
	if ringkasan.Total.Count != 0 {
		t.Errorf("konteks tanpa scope mengembalikan %d invoice — seharusnya nol",
			ringkasan.Total.Count)
	}
}
