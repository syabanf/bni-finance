package chapter

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

func ringkas(cs []domain.Chapter) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.ID)
	}
	return out
}
