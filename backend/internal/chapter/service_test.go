package chapter

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/cache"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

// storePalsu menjawab List sesuai lingkup, seperti Repository, dan menghitung
// berapa kali ia benar-benar ditanya.
type storePalsu struct {
	Store
	semua []domain.Chapter
	tanya int
}

func (s *storePalsu) List(ctx context.Context, _ domain.ChapterFilter) ([]domain.Chapter, int, error) {
	s.tanya++
	lim := scope.Chapter(ctx)
	if !lim.Terbatas {
		return s.semua, len(s.semua), nil
	}
	for _, c := range s.semua {
		if c.ID == lim.ChapterID {
			return []domain.Chapter{c}, 1, nil
		}
	}
	return nil, 0, nil
}

func (s *storePalsu) Create(_ context.Context, in domain.CreateChapterInput) (*domain.Chapter, error) {
	c := domain.Chapter{ID: *in.ID, Name: in.Name, DisplayName: in.Name}
	s.semua = append(s.semua, c)
	return &c, nil
}

func layananLive(t *testing.T) (*Service, *storePalsu) {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("set TEST_REDIS_URL untuk menjalankan tes ini")
	}
	r, err := cache.Buka(context.Background(), url, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	store := &storePalsu{semua: []domain.Chapter{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}}
	svc := NewService(store).PakaiCache(r)
	svc.Kedaluwarsakan(context.Background()) // sisa tes sebelumnya
	return svc, store
}

// Daftar milik admin yang tersimpan lebih dulu TIDAK boleh disajikan ke ST.
// Kunci cache yang lupa memuat lingkup membocorkan seluruh chapter ke setiap
// pengguna berlingkup, persis yang dicegah repository_scope_test.
func TestCacheDaftarDipisahPerLingkup(t *testing.T) {
	svc, _ := layananLive(t)
	admin := scope.WithoutLimit(context.Background())
	st := scope.WithChapter(context.Background(), "b")

	if items, _, _ := svc.List(admin, domain.ChapterFilter{Limit: 50}); len(items) != 2 {
		t.Fatalf("admin melihat %d chapter", len(items))
	}
	items, total, err := svc.List(st, domain.ChapterFilter{Limit: 50})
	if err != nil || total != 1 || len(items) != 1 || items[0].ID != "b" {
		t.Errorf("ST mendapat daftar admin dari cache: %v (total %d, %v)", items, total, err)
	}
}

func TestCacheDaftarGugurSetelahChapterDibuat(t *testing.T) {
	svc, store := layananLive(t)
	admin := scope.WithoutLimit(context.Background())
	f := domain.ChapterFilter{Limit: 50}

	idBaru := "c"
	svc.List(admin, f)
	svc.List(admin, f)
	if store.tanya != 1 {
		t.Fatalf("baca kedua tidak dari cache: %d kali ke database", store.tanya)
	}
	if _, err := svc.Create(admin, domain.CreateChapterInput{ID: &idBaru, Name: "C"}); err != nil {
		t.Fatal(err)
	}
	if items, _, _ := svc.List(admin, f); len(items) != 3 {
		t.Errorf("chapter baru tidak muncul, cache basi: %d chapter", len(items))
	}
}

func (s *storePalsu) Counts(ctx context.Context) ([]domain.ChapterCounts, error) {
	s.tanya++
	if lim := scope.Chapter(ctx); lim.Terbatas {
		return []domain.ChapterCounts{{ChapterID: lim.ChapterID, MemberCount: 3}}, nil
	}
	return []domain.ChapterCounts{{ChapterID: "a", MemberCount: 5}, {ChapterID: "b", MemberCount: 3}}, nil
}

// Jumlah per chapter milik admin TIDAK boleh disajikan ke ST, dan gugur saat
// invoice atau pembayaran ditulis.
func TestCacheJumlahPerChapter(t *testing.T) {
	svc, store := layananLive(t)
	svc.KedaluwarsakanAngka(context.Background()) // sisa tes sebelumnya
	admin := scope.WithoutLimit(context.Background())
	st := scope.WithChapter(context.Background(), "b")
	store.tanya = 0

	if items, err := svc.Counts(admin); err != nil || len(items) != 2 {
		t.Fatalf("admin melihat %d chapter (galat %v), ingin 2", len(items), err)
	}
	if items, err := svc.Counts(st); err != nil || len(items) != 1 || items[0].ChapterID != "b" {
		t.Fatalf("ST melihat %v, ingin hanya chapter b", items)
	}
	_, _ = svc.Counts(admin)
	_, _ = svc.Counts(st)
	if store.tanya != 2 {
		t.Errorf("repository ditanya %d kali, ingin 2", store.tanya)
	}

	svc.KedaluwarsakanAngka(admin)
	_, _ = svc.Counts(admin)
	if store.tanya != 3 {
		t.Errorf("setelah digugurkan repository ditanya %d kali, ingin 3", store.tanya)
	}
}
