package audit

import (
	"context"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Store interface {
	ListByInvoice(ctx context.Context, invoiceID string, limit int) ([]domain.AuditEntry, error)
	Create(ctx context.Context, invoiceID string, in domain.CreateAuditEntryInput) (*domain.AuditEntry, error)
	InvoiceExists(ctx context.Context, invoiceID string) error
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) ListByInvoice(ctx context.Context, invoiceID string, limit int) ([]domain.AuditEntry, error) {
	if err := s.repo.InvoiceExists(ctx, invoiceID); err != nil {
		return nil, err
	}
	return s.repo.ListByInvoice(ctx, invoiceID, limit)
}

func (s *Service) Create(ctx context.Context, invoiceID string, in domain.CreateAuditEntryInput) (*domain.AuditEntry, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	if err := s.repo.InvoiceExists(ctx, invoiceID); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, invoiceID, in)
}
