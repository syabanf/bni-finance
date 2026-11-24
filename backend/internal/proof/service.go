package proof

import (
	"context"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/upload"
)

type Store interface {
	List(ctx context.Context, f domain.ProofFilter) ([]domain.PaymentProof, int, error)
	GetByID(ctx context.Context, id string) (*domain.PaymentProof, error)
	Create(ctx context.Context, in domain.SubmitProofInput, submittedBy string) (*domain.PaymentProof, error)
	Approve(ctx context.Context, id, reviewer string, note *string) (*domain.PaymentProof, error)
	Reject(ctx context.Context, id, reviewer, note string) (*domain.PaymentProof, error)
	Delete(ctx context.Context, id string) error
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo         Store
	setelahTulis func(context.Context)
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

// SetelahTulis dipanggil setelah bukti disetujui, karena persetujuan mengubah
// angka dashboard: invoice menjadi lunas dan pembayarannya tercatat.
func (s *Service) SetelahTulis(fn func(context.Context)) *Service {
	s.setelahTulis = fn
	return s
}

func (s *Service) List(ctx context.Context, f domain.ProofFilter) ([]domain.PaymentProof, int, error) {
	if f.Status != "" && !domain.ProofStatus(f.Status).Valid() {
		return nil, 0, httpx.BadRequest("status harus 'pending', 'approved', atau 'rejected'")
	}
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*domain.PaymentProof, error) {
	return s.repo.GetByID(ctx, id)
}

// Kirim menyimpan bukti bayar dari ST. Berkasnya diunggah lebih dulu lewat
// POST /api/v1/uploads, sama seperti faktur pajak.
func (s *Service) Kirim(ctx context.Context, in domain.SubmitProofInput) (*domain.PaymentProof, error) {
	if err := in.Validate(time.Now()); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	in.ProofURL = strings.TrimSpace(in.ProofURL)
	if !strings.HasPrefix(in.ProofURL, upload.URLPrefix) || strings.Contains(in.ProofURL, "..") {
		return nil, httpx.BadRequest("proofUrl harus berkas yang diunggah lewat /api/v1/uploads")
	}
	return s.repo.Create(ctx, in, namaPelaku(ctx))
}

func (s *Service) Setujui(ctx context.Context, id string, in domain.ReviewProofInput) (*domain.PaymentProof, error) {
	p, err := s.repo.Approve(ctx, id, namaPelaku(ctx), in.Note)
	if err == nil && s.setelahTulis != nil {
		s.setelahTulis(ctx)
	}
	return p, err
}

func (s *Service) Tolak(ctx context.Context, id string, in domain.ReviewProofInput) (*domain.PaymentProof, error) {
	if in.Note == nil || strings.TrimSpace(*in.Note) == "" {
		return nil, httpx.BadRequest("tulis alasan penolakan, supaya ST tahu apa yang harus diperbaiki")
	}
	return s.repo.Reject(ctx, id, namaPelaku(ctx), strings.TrimSpace(*in.Note))
}

func (s *Service) Tarik(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func namaPelaku(ctx context.Context) string {
	if u, ok := auth.UserFrom(ctx); ok {
		return u.Name
	}
	return ""
}
