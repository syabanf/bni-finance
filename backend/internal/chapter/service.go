package chapter

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/cache"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

// Store is the persistence contract, kept an interface so handlers can be
// tested without Postgres.
type Store interface {
	List(ctx context.Context, f domain.ChapterFilter) ([]domain.Chapter, int, error)
	GetByID(ctx context.Context, id string) (*domain.Chapter, error)
	Create(ctx context.Context, in domain.CreateChapterInput) (*domain.Chapter, error)
	Update(ctx context.Context, id string, in domain.UpdateChapterInput) (*domain.Chapter, error)
	Delete(ctx context.Context, id string) error
	CountDependents(ctx context.Context, id string) (members, invoices int, err error)
	Counts(ctx context.Context) ([]domain.ChapterCounts, error)
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo  Store
	cache *cache.Redis
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

// PakaiCache memasang Redis untuk daftar chapter. nil berarti tanpa cache.
func (s *Service) PakaiCache(c *cache.Redis) *Service {
	s.cache = c
	return s
}

// kunciGenerasi menamai penghitung versi daftar chapter. Menaikkannya
// menggugurkan seluruh variasi daftar (per lingkup dan per penyaring) sekaligus.
const kunciGenerasi = "chapters:gen"

// ttlDaftar menjadi batas atas data basi bila ada penulis chapter yang lupa
// memanggil Kedaluwarsakan.
const ttlDaftar = 5 * time.Minute

type halaman struct {
	Items []domain.Chapter `json:"items"`
	Total int              `json:"total"`
}

// List membaca daftar chapter, lewat cache bila ada.
//
// Kuncinya memuat lingkup pengguna. Tanpa itu daftar milik admin yang
// tersimpan lebih dulu akan disajikan ke ST, dan ST melihat seluruh chapter.
func (s *Service) List(ctx context.Context, f domain.ChapterFilter) ([]domain.Chapter, int, error) {
	lim := scope.Chapter(ctx)
	if s.cache == nil || lim.Buntu {
		return s.repo.List(ctx, f)
	}
	lingkup := "semua"
	if lim.Terbatas {
		lingkup = "chapter=" + lim.ChapterID
	}
	key := fmt.Sprintf("chapters:%d:%s:%q:%q:%q:%d:%d", s.cache.Generasi(ctx, kunciGenerasi),
		lingkup, f.Search, f.CityName, f.AreaName, f.Limit, f.Offset)
	h, err := cache.Ingat(ctx, s.cache, key, ttlDaftar, func() (halaman, error) {
		items, total, err := s.repo.List(ctx, f)
		return halaman{items, total}, err
	})
	return h.Items, h.Total, err
}

// Kedaluwarsakan menggugurkan cache daftar chapter. Dipanggil setiap penulis
// tabel chapters: layanan ini, impor laporan, dan sinkronisasi BNI VM.
func (s *Service) Kedaluwarsakan(ctx context.Context) {
	s.cache.NaikkanGenerasi(ctx, kunciGenerasi)
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Chapter, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, in domain.CreateChapterInput) (*domain.Chapter, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	c, err := s.repo.Create(ctx, in)
	if err == nil {
		s.Kedaluwarsakan(ctx)
	}
	return c, err
}

func (s *Service) Update(ctx context.Context, id string, in domain.UpdateChapterInput) (*domain.Chapter, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	c, err := s.repo.Update(ctx, id, in)
	if err == nil {
		s.Kedaluwarsakan(ctx)
	}
	return c, err
}

// Delete refuses to remove a chapter that members or invoices still reference —
// the foreign key would reject it anyway, but a 409 with counts is far more
// useful than a raw constraint error.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	members, invoices, err := s.repo.CountDependents(ctx, id)
	if err != nil {
		return err
	}
	if members > 0 || invoices > 0 {
		return httpx.Conflict(fmt.Sprintf(
			"chapter masih dipakai oleh %d member dan %d invoice — pindahkan dulu sebelum menghapus",
			members, invoices))
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.Kedaluwarsakan(ctx)
	return nil
}

// Counts meneruskan agregat per chapter; pembatasan lingkupnya ada di repository.
func (s *Service) Counts(ctx context.Context) ([]domain.ChapterCounts, error) {
	return s.repo.Counts(ctx)
}
