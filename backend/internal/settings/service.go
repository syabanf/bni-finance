package settings

import (
	"context"
	"strings"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Store interface {
	GetFees(ctx context.Context) (*domain.FeeSettings, error)
	UpdateFees(ctx context.Context, in domain.UpdateFeeSettingsInput) (*domain.FeeSettings, error)
	ListApp(ctx context.Context) ([]domain.AppSetting, error)
	GetApp(ctx context.Context, key string) (*domain.AppSetting, error)
	SetApp(ctx context.Context, key, value string) (*domain.AppSetting, error)
	DeleteApp(ctx context.Context, key string) error
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) GetFees(ctx context.Context) (*domain.FeeSettings, error) {
	return s.repo.GetFees(ctx)
}

func (s *Service) UpdateFees(ctx context.Context, in domain.UpdateFeeSettingsInput) (*domain.FeeSettings, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	if in.RegistrationFee == nil && in.RenewalFee == nil &&
		in.Currency == nil && in.Notes == nil && in.UpdatedBy == nil {
		return nil, httpx.BadRequest("tidak ada field yang diubah")
	}
	return s.repo.UpdateFees(ctx, in)
}

// ListApp returns every setting with credential-shaped values redacted.
func (s *Service) ListApp(ctx context.Context) ([]domain.AppSetting, error) {
	items, err := s.repo.ListApp(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AppSetting, len(items))
	for i, it := range items {
		out[i] = it.Redact()
	}
	return out, nil
}

func (s *Service) GetApp(ctx context.Context, key string) (*domain.AppSetting, error) {
	item, err := s.repo.GetApp(ctx, normalizeKey(key))
	if err != nil {
		return nil, err
	}
	redacted := item.Redact()
	return &redacted, nil
}

// SetApp writes a value. Secrets are write-only: the response is redacted the
// same way a read would be, so the plaintext never travels back out.
func (s *Service) SetApp(ctx context.Context, key string, in domain.SetAppSettingInput) (*domain.AppSetting, error) {
	key = normalizeKey(key)
	if key == "" {
		return nil, httpx.BadRequest("key wajib diisi")
	}
	if in.Value == domain.MaskedValue {
		return nil, httpx.BadRequest("value masih berupa nilai tersamar — kirim nilai sebenarnya")
	}
	item, err := s.repo.SetApp(ctx, key, in.Value)
	if err != nil {
		return nil, err
	}
	redacted := item.Redact()
	return &redacted, nil
}

func (s *Service) DeleteApp(ctx context.Context, key string) error {
	return s.repo.DeleteApp(ctx, normalizeKey(key))
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
