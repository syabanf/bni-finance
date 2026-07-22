package member

import (
	"context"
	"fmt"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Store interface {
	List(ctx context.Context, f domain.MemberFilter) ([]domain.Member, int, error)
	GetByID(ctx context.Context, id string) (*domain.Member, error)
	RenewalDue(ctx context.Context, days, limit int) ([]domain.RenewalDueMember, error)
	Create(ctx context.Context, in domain.CreateMemberInput) (*domain.Member, error)
	Update(ctx context.Context, id string, in domain.UpdateMemberInput) (*domain.Member, error)
	Delete(ctx context.Context, id string) error
	CountInvoices(ctx context.Context, id string) (int, error)
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f domain.MemberFilter) ([]domain.Member, int, error) {
	if f.Status != "" && !domain.MemberStatus(f.Status).Valid() {
		return nil, 0, httpx.BadRequest("status harus 'active', 'inactive', atau 'pending'")
	}
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Member, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) RenewalDue(ctx context.Context, days, limit int) ([]domain.RenewalDueMember, error) {
	return s.repo.RenewalDue(ctx, days, limit)
}

func (s *Service) Create(ctx context.Context, in domain.CreateMemberInput) (*domain.Member, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	return s.repo.Create(ctx, in)
}

func (s *Service) Update(ctx context.Context, id string, in domain.UpdateMemberInput) (*domain.Member, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	return s.repo.Update(ctx, id, in)
}

// Delete refuses to remove a member who still has invoices — deactivate the
// member instead so the billing history stays intact.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	n, err := s.repo.CountInvoices(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return httpx.Conflict(fmt.Sprintf(
			"member masih punya %d invoice — ubah statusnya menjadi 'inactive' alih-alih menghapus", n))
	}
	return s.repo.Delete(ctx, id)
}
