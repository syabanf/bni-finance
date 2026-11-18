package dashboard

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

// storePalsu menjawab menurut lingkup, seperti Repository: angka nasional
// untuk admin, angka satu chapter untuk ST. Ia juga menghitung berapa kali
// benar-benar ditanya.
type storePalsu struct {
	tanya int
}

func (s *storePalsu) Summary(ctx context.Context, _ int, _ domain.InvoiceType) (*domain.DashboardSummary, error) {
	s.tanya++
	jumlah := 100
	if scope.Chapter(ctx).Terbatas {
		jumlah = 7
	}
	return &domain.DashboardSummary{Total: domain.AmountBucket{Count: jumlah}}, nil
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
	store := &storePalsu{}
	svc := NewService(store).PakaiCache(r)
	svc.Kedaluwarsakan(context.Background()) // sisa tes sebelumnya
	return svc, store
}

// Angka nasional milik admin yang tersimpan lebih dulu TIDAK boleh disajikan
// ke ST. Kunci cache yang lupa memuat lingkup membocorkan jumlah uang seluruh
// chapter ke setiap pengguna berlingkup.
func TestCacheRingkasanDipisahPerLingkup(t *testing.T) {
	svc, store := layananLive(t)
	admin := scope.WithoutLimit(context.Background())
	st := scope.WithChapter(context.Background(), "ch-b")

	if s, err := svc.Summary(admin, 6, ""); err != nil || s.Total.Count != 100 {
		t.Fatalf("admin: %v, jumlah %d", err, s.Total.Count)
	}
	if s, err := svc.Summary(st, 6, ""); err != nil || s.Total.Count != 7 {
		t.Fatalf("ST mendapat jumlah %d, ingin 7 (angka chapternya sendiri)", s.Total.Count)
	}
	// Panggilan kedua dari masing-masing dijawab cache.
	_, _ = svc.Summary(admin, 6, "")
	_, _ = svc.Summary(st, 6, "")
	if store.tanya != 2 {
		t.Errorf("repository ditanya %d kali, ingin 2", store.tanya)
	}
}

// Tipe dan rentang bulan ikut membedakan kunci, dan pengguguran berlaku untuk
// semua variasinya.
func TestCacheRingkasanGugurSetelahTulis(t *testing.T) {
	svc, store := layananLive(t)
	admin := scope.WithoutLimit(context.Background())

	_, _ = svc.Summary(admin, 6, "")
	_, _ = svc.Summary(admin, 6, domain.TypeRenewal)
	_, _ = svc.Summary(admin, 12, "")
	if store.tanya != 3 {
		t.Fatalf("tiga variasi seharusnya tiga kali bertanya, dapat %d", store.tanya)
	}

	svc.Kedaluwarsakan(admin)
	_, _ = svc.Summary(admin, 6, "")
	_, _ = svc.Summary(admin, 6, domain.TypeRenewal)
	if store.tanya != 5 {
		t.Errorf("setelah digugurkan repository ditanya %d kali, ingin 5", store.tanya)
	}
}

// Tanpa Redis layanan tetap menjawab, langsung dari repository.
func TestRingkasanTanpaCache(t *testing.T) {
	store := &storePalsu{}
	svc := NewService(store)
	svc.Kedaluwarsakan(context.Background())
	for i := 0; i < 2; i++ {
		if _, err := svc.Summary(scope.WithoutLimit(context.Background()), 6, ""); err != nil {
			t.Fatal(err)
		}
	}
	if store.tanya != 2 {
		t.Errorf("tanpa cache repository ditanya %d kali, ingin 2", store.tanya)
	}
}
