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
	GetByID(ctx context.Context, id string) (*domain.Payment, error)
	CreateAndSettle(ctx context.Context, in domain.CreatePaymentInput, paidAt time.Time, settle bool) (*domain.Payment, error)
	Update(ctx context.Context, id string, in domain.UpdatePaymentInput) (*domain.Payment, error)
	Delete(ctx context.Context, id string) error
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f domain.PaymentFilter) ([]domain.Payment, int, error) {
	return s.repo.List(ctx, f)
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

	return s.repo.CreateAndSettle(ctx, in, paidAt, in.ShouldSettle())
}

func (s *Service) Update(ctx context.Context, id string, in domain.UpdatePaymentInput) (*domain.Payment, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, in)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
