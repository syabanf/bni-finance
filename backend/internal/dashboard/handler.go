package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/cache"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

type Store interface {
	Summary(ctx context.Context, months int, tipe domain.InvoiceType) (*domain.DashboardSummary, error)
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo  Store
	cache *cache.Redis
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

// PakaiCache memasang Redis untuk ringkasan dashboard. nil berarti tanpa cache.
func (s *Service) PakaiCache(c *cache.Redis) *Service {
	s.cache = c
	return s
}

// kunciGenerasi menamai penghitung versi ringkasan. Menaikkannya menggugurkan
// seluruh variasinya (per lingkup, tipe, dan rentang bulan) sekaligus.
const kunciGenerasi = "dashboard:gen"

// ttlRingkasan adalah batas atas angka basi.
//
// Pendek dengan sengaja. Invoice dan pembayaran yang ditulis lewat layanannya
// menggugurkan cache saat itu juga, tetapi beberapa penulis lain tidak:
// callback Paper.id, worker pengingat yang menandai overdue, dan impor. Untuk
// mereka, angka di dashboard tertinggal paling lama selama ini.
const ttlRingkasan = 15 * time.Second

// Summary menghitung ringkasan dashboard, lewat cache bila ada.
//
// Lima query agregat di baliknya adalah permintaan terberat di API ini: diukur
// dengan 30 pengguna bersamaan atas 15.000 invoice, p95-nya 1,5 detik dan ia
// yang paling lama menahan koneksi pool.
//
// Kuncinya memuat lingkup pengguna. Tanpa itu angka nasional milik admin yang
// tersimpan lebih dulu akan disajikan ke ST.
func (s *Service) Summary(ctx context.Context, months int, tipe domain.InvoiceType) (*domain.DashboardSummary, error) {
	lim := scope.Chapter(ctx)
	if s.cache == nil || lim.Buntu {
		return s.repo.Summary(ctx, months, tipe)
	}
	lingkup := "semua"
	if lim.Terbatas {
		lingkup = "chapter=" + lim.ChapterID
	}
	key := fmt.Sprintf("dashboard:%d:%s:%d:%q", s.cache.Generasi(ctx, kunciGenerasi), lingkup, months, tipe)
	return cache.Ingat(ctx, s.cache, key, ttlRingkasan, func() (*domain.DashboardSummary, error) {
		return s.repo.Summary(ctx, months, tipe)
	})
}

// Kedaluwarsakan menggugurkan cache ringkasan. Dipanggil setelah invoice atau
// pembayaran ditulis.
func (s *Service) Kedaluwarsakan(ctx context.Context) {
	s.cache.NaikkanGenerasi(ctx, kunciGenerasi)
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/dashboard/summary", h.summary)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	months := httpx.QueryInt(r, "months", 6, 1, 24)
	// ?type=renewal|registration menyaring seluruh angka invoice; kosong = semua.
	tipe := domain.InvoiceType(httpx.Query(r, "type"))
	if tipe != "" && !tipe.Valid() {
		httpx.Fail(w, httpx.BadRequest("type harus renewal atau registration"))
		return
	}

	sum, err := h.svc.Summary(r.Context(), months, tipe)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}
