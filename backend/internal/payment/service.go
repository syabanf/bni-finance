package payment

import (
	"context"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// Store is the persistence contract for payments (see invoice.Store).
type Store interface {
	List(ctx context.Context, f domain.PaymentFilter) ([]domain.Payment, int, error)
	Summary(ctx context.Context, f domain.PaymentFilter) (*domain.PaymentSummary, error)
	GetByID(ctx context.Context, id string) (*domain.Payment, error)
	CreateAndSettle(ctx context.Context, in domain.CreatePaymentInput, paidAt time.Time, settle bool) (*domain.Payment, error)
	Update(ctx context.Context, id string, in domain.UpdatePaymentInput) (*domain.Payment, error)
	Delete(ctx context.Context, id string) error
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo         Store
	setelahTulis func(context.Context)
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

// SetelahTulis mendaftarkan fungsi yang dipanggil setiap kali layanan ini
// berhasil menulis. Dipakai untuk menggugurkan cache ringkasan dashboard.
func (s *Service) SetelahTulis(fn func(context.Context)) *Service {
	s.setelahTulis = fn
	return s
}

func (s *Service) kabari(ctx context.Context, err error) {
	if err == nil && s.setelahTulis != nil {
		s.setelahTulis(ctx)
	}
}

func (s *Service) List(ctx context.Context, f domain.PaymentFilter) ([]domain.Payment, int, error) {
	if err := validasiFilter(f); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, f)
}

// Summary menjumlahkan seluruh hasil filter, untuk kartu ringkasan.
func (s *Service) Summary(ctx context.Context, f domain.PaymentFilter) (*domain.PaymentSummary, error) {
	if err := validasiFilter(f); err != nil {
		return nil, err
	}
	return s.repo.Summary(ctx, f)
}

func validasiFilter(f domain.PaymentFilter) error {
	if f.Type != "" && !domain.InvoiceType(f.Type).Valid() {
		return httpx.BadRequest("type harus 'registration' atau 'renewal'")
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Payment, error) {
	return s.repo.GetByID(ctx, id)
}

// Create records a payment. By default it also settles the invoice, matching
// the behaviour the UI expects when an admin records a manual payment.
func (s *Service) Create(ctx context.Context, in domain.CreatePaymentInput) (*domain.Payment, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}

	paidAt := time.Now().UTC()
	if in.PaidAt != nil {
		paidAt = *in.PaidAt
	}
	if paidAt.After(time.Now().Add(24 * time.Hour)) {
		return nil, httpx.BadRequest("paidAt tidak boleh di masa depan")
	}

	p, err := s.repo.CreateAndSettle(ctx, in, paidAt, in.ShouldSettle())
	s.kabari(ctx, err)
	return p, err
}

func (s *Service) Update(ctx context.Context, id string, in domain.UpdatePaymentInput) (*domain.Payment, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, err
	}
	p, err := s.repo.Update(ctx, id, in)
	s.kabari(ctx, err)
	return p, err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	err := s.repo.Delete(ctx, id)
	s.kabari(ctx, err)
	return err
}
