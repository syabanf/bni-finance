package invoice

import (
	"context"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// Store is the persistence contract the service depends on. Keeping it an
// interface (rather than *Repository) lets the HTTP layer be unit- and
// load-tested without a live Postgres.
type Store interface {
	List(ctx context.Context, f domain.InvoiceFilter) ([]domain.Invoice, int, error)
	GetByID(ctx context.Context, id string) (*domain.Invoice, error)
	Create(ctx context.Context, in domain.CreateInvoiceInput, number, currency string) (*domain.Invoice, error)
	Update(ctx context.Context, id string, in domain.UpdateInvoiceInput) (*domain.Invoice, error)
	Delete(ctx context.Context, id string) error
	CountPayments(ctx context.Context, invoiceID string) (int, error)
	NextNumber(ctx context.Context, year int) (string, error)
}

// compile-time check that the Postgres repository satisfies the contract.
var _ Store = (*Repository)(nil)

type Service struct {
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Invoice, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, in domain.CreateInvoiceInput) (*domain.Invoice, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}

	currency := "IDR"
	if in.Currency != nil && *in.Currency != "" {
		currency = *in.Currency
	}

	number := ""
	if in.Number != nil && *in.Number != "" {
		number = *in.Number
	} else {
		generated, err := s.repo.NextNumber(ctx, in.DueDate.Year())
		if err != nil {
			return nil, err
		}
		number = generated
	}

	return s.repo.Create(ctx, in, number, currency)
}

// Update applies a patch, rejecting illegal status transitions so an invoice
// can't jump straight from draft to paid or move off a terminal state.
func (s *Service) Update(ctx context.Context, id string, in domain.UpdateInvoiceInput) (*domain.Invoice, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Status != nil && !current.Status.CanTransitionTo(*in.Status) {
		return nil, httpx.Conflict(
			"transisi status tidak diizinkan: " + string(current.Status) + " → " + string(*in.Status),
		)
	}

	// A paid or cancelled invoice is a closed record — don't let amounts or
	// periods be rewritten after the fact.
	if current.Status == domain.StatusPaid || current.Status == domain.StatusCancelled {
		if in.Amount != nil || in.DueDate != nil || in.PeriodStart != nil || in.PeriodEnd != nil {
			return nil, httpx.Conflict("invoice berstatus " + string(current.Status) + " tidak bisa diubah nominal/periodenya")
		}
	}

	return s.repo.Update(ctx, id, in)
}

// Delete refuses to remove an invoice that still has payments attached.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	n, err := s.repo.CountPayments(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return httpx.Conflict("invoice masih punya data pembayaran — batalkan invoice alih-alih menghapusnya")
	}
	return s.repo.Delete(ctx, id)
}

// MarkOverdue is a helper the caller can schedule: any sent invoice past its
// due date becomes overdue.
func (s *Service) NextNumberFor(ctx context.Context, t time.Time) (string, error) {
	return s.repo.NextNumber(ctx, t.Year())
}
