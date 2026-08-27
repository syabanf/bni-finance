package chapter

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Daftar chapter juga berlingkup.
//
// Invoice dan member sudah dibatasi dengan benar sejak awal, tapi daftar
// chapter tidak — sehingga seorang ST tetap mendapat nama dan keberadaan
// SELURUH chapter. Daftar itu pula yang mengisi dropdown penyaring chapter,
// sehingga tampak seolah ia boleh menelusuri chapter lain.
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

// duaChapter membuat dunianya sendiri, bukan membaca data contoh — tes yang
// bergantung pada data yang ada akan memanggil t.Skip setelah integration test
// lain men-TRUNCATE tabel, dan `go test` melaporkannya "ok".
func duaChapter(t *testing.T, db *pgxpool.Pool) (a, b string) {
	t.Helper()
	a, b = "ch-lingkup-a", "ch-lingkup-b"
	for i, ch := range []string{a, b} {
		if _, err := db.Exec(context.Background(),
			`INSERT INTO chapters (id, name, display_name, area_name, city_name)
			 VALUES ($1,$2,$3,'Uji','Uji') ON CONFLICT (id) DO NOTHING`,
			ch, "lingkup"+string(rune('a'+i)), "Lingkup "+string(rune('A'+i))); err != nil {
			t.Fatalf("siapkan chapter %s: %v", ch, err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.Exec(context.Background(),
			`DELETE FROM chapters WHERE id = ANY($1)`, []string{a, b})
	})
	return a, b
}

func TestDaftarChapterBerlingkup(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB := duaChapter(t, pool)

	// --- ST hanya melihat chapternya sendiri --------------------------------
	items, total, err := repo.List(scope.WithChapter(context.Background(), chA),
		domain.ChapterFilter{Limit: 500})
	if err != nil {
		t.Fatalf("list sebagai ST: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != chA {
		t.Errorf("ST chapter %s melihat %d chapter: %v", chA, total, ringkas(items))
	}

	// --- admin melihat keduanya ---------------------------------------------
	semua, _, err := repo.List(scope.WithoutLimit(context.Background()),
		domain.ChapterFilter{Limit: 500})
	if err != nil {
		t.Fatalf("list sebagai admin: %v", err)
	}
	ada := map[string]bool{}
	for _, c := range semua {
		ada[c.ID] = true
	}
	if !ada[chA] || !ada[chB] {
		t.Errorf("admin tidak melihat kedua chapter — pembatasannya kelewat jauh: %v", ringkas(semua))
	}

	// --- tanpa scope: gagal TERTUTUP ----------------------------------------
	kosong, jml, err := repo.List(context.Background(), domain.ChapterFilter{Limit: 500})
	if err != nil {
		t.Fatalf("list tanpa scope: %v", err)
	}
	if jml != 0 || len(kosong) != 0 {
		t.Errorf("konteks tanpa scope mengembalikan %d chapter — seharusnya nol", jml)
	}
}

// Counts tidak boleh menggandakan angka, dan harus tetap berlingkup.
//
// Bahaya yang dijaga di sini bukan hipotesis: meng-JOIN members dan invoices
// sekaligus menghasilkan hasil kali kartesian di dalam tiap chapter. Diuji
// langsung ke basis data dengan 3 member dan 4 invoice, versi JOIN memberi
// member=12 dan tunggakan Rp12 juta — dua-duanya terkali, dan dua-duanya masih
// terlihat seperti angka yang wajar.
func TestHitunganChapterTidakMenggandaDanBerlingkup(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB := duaChapter(t, pool)
	ctx := context.Background()

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("siapkan data: %v", err)
		}
	}
	// Chapter A: 3 member, 4 invoice sent @1 juta → 3 dan 4.000.000.
	for i := 1; i <= 3; i++ {
		exec(`INSERT INTO members (id, chapter_id, name, email, phone, status)
		      VALUES ($1,$2,$3,'h@contoh.invalid','08','active') ON CONFLICT (id) DO NOTHING`,
			fmt.Sprintf("mem-hitung-%d", i), chA, fmt.Sprintf("Member %d", i))
	}
	for i := 1; i <= 4; i++ {
		exec(`INSERT INTO invoices (id, number, member_id, chapter_id, type, amount, currency,
		                          due_date, period_start, period_end, status)
		      VALUES ($1,$2,'mem-hitung-1',$3,'renewal',1000000,'IDR',
		              CURRENT_DATE+30, CURRENT_DATE, CURRENT_DATE+365,'sent')
		      ON CONFLICT (number) DO NOTHING`,
			fmt.Sprintf("33333333-3333-3333-3333-33333333333%d", i),
			fmt.Sprintf("HITUNG-%d", i), chA)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM invoices WHERE number LIKE 'HITUNG-%'`)
		_, _ = pool.Exec(c, `DELETE FROM members WHERE id LIKE 'mem-hitung-%'`)
	})

	cari := func(items []domain.ChapterCounts, id string) *domain.ChapterCounts {
		for i := range items {
			if items[i].ChapterID == id {
				return &items[i]
			}
		}
		return nil
	}

	// --- admin: angkanya harus persis, bukan kelipatannya --------------------
	semua, err := repo.Counts(scope.WithoutLimit(ctx))
	if err != nil {
		t.Fatalf("counts sebagai admin: %v", err)
	}
	a := cari(semua, chA)
	if a == nil {
		t.Fatalf("chapter %s tidak ada di hasil", chA)
	}
	if a.MemberCount != 3 {
		t.Errorf("memberCount = %d, seharusnya 3 — JOIN menggandakan baris?", a.MemberCount)
	}
	if a.Outstanding != 4_000_000 {
		t.Errorf("outstanding = %d, seharusnya 4000000 — JOIN menggandakan baris?", a.Outstanding)
	}

	// --- ST hanya melihat chapternya ----------------------------------------
	milikST, err := repo.Counts(scope.WithChapter(ctx, chA))
	if err != nil {
		t.Fatalf("counts sebagai ST: %v", err)
	}
	if len(milikST) != 1 || milikST[0].ChapterID != chA {
		t.Errorf("ST melihat %d chapter, seharusnya hanya %s", len(milikST), chA)
	}
	if cari(milikST, chB) != nil {
		t.Errorf("ST melihat hitungan chapter %s", chB)
	}

	// --- tanpa scope: gagal TERTUTUP ----------------------------------------
	kosong, err := repo.Counts(ctx)
	if err != nil {
		t.Fatalf("counts tanpa scope: %v", err)
	}
	if len(kosong) != 0 {
		t.Errorf("konteks tanpa scope mengembalikan %d chapter — seharusnya nol", len(kosong))
	}
}

func ringkas(cs []domain.Chapter) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.ID)
	}
	return out
}
