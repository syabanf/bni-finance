package chapter

import (
	"context"
	"fmt"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
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
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f domain.ChapterFilter) ([]domain.Chapter, int, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Chapter, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, in domain.CreateChapterInput) (*domain.Chapter, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	return s.repo.Create(ctx, in)
}

func (s *Service) Update(ctx context.Context, id string, in domain.UpdateChapterInput) (*domain.Chapter, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	return s.repo.Update(ctx, id, in)
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
	return s.repo.Delete(ctx, id)
}

// Counts meneruskan agregat per chapter; pembatasan lingkupnya ada di repository.
func (s *Service) Counts(ctx context.Context) ([]domain.ChapterCounts, error) {
	return s.repo.Counts(ctx)
}
