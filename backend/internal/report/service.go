package report

import (
	"context"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Store interface {
	Summary(ctx context.Context, f domain.ReportFilter) (*domain.ReportSummary, error)
	Detail(ctx context.Context, f domain.ReportFilter, limit, offset int) ([]domain.ReportDetailRow, int, error)
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) Summary(ctx context.Context, f domain.ReportFilter) (*domain.ReportSummary, error) {
	if err := validasi(f); err != nil {
		return nil, err
	}
	return s.repo.Summary(ctx, f)
}

func (s *Service) Detail(ctx context.Context, f domain.ReportFilter, limit, offset int) ([]domain.ReportDetailRow, int, error) {
	if err := validasi(f); err != nil {
		return nil, 0, err
	}
	return s.repo.Detail(ctx, f, limit, offset)
}

// validasi menolak tanggal dan tipe yang salah bentuk dengan 400. Tanpa ini
// Postgres yang menolaknya, dan pemanggil hanya melihat 500.
func validasi(f domain.ReportFilter) error {
	for nama, v := range map[string]string{"from": f.From, "to": f.To} {
		if v == "" {
			continue
		}
		if _, err := domain.ParseDate(v); err != nil {
			return httpx.BadRequest(nama + " harus berformat YYYY-MM-DD")
		}
	}
	if f.Type != "" && !domain.InvoiceType(f.Type).Valid() {
		return httpx.BadRequest("type harus 'registration' atau 'renewal'")
	}
	return nil
}
