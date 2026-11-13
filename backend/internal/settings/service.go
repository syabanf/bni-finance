package settings

import (
	"context"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/cache"
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
	repo  Store
	cache *cache.Redis
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

// PakaiCache memasang Redis untuk biaya keanggotaan. nil berarti tanpa cache.
func (s *Service) PakaiCache(c *cache.Redis) *Service {
	s.cache = c
	return s
}

// kunciBiaya dihapus setiap UpdateFees berhasil, termasuk dari worker kurs.
// TTL-nya batas atas bila penghapusan itu gagal karena Redis sedang mati.
const (
	kunciBiaya = "fees"
	ttlBiaya   = 10 * time.Minute
)

func (s *Service) GetFees(ctx context.Context) (*domain.FeeSettings, error) {
	return cache.Ingat(ctx, s.cache, kunciBiaya, ttlBiaya, func() (*domain.FeeSettings, error) {
		return s.repo.GetFees(ctx)
	})
}

func (s *Service) UpdateFees(ctx context.Context, in domain.UpdateFeeSettingsInput) (*domain.FeeSettings, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	// UpdatedBy TIDAK ikut dihitung di sini.
	//
	// Nilainya diisi server dari token, jadi ia selalu ada — memasukkannya ke
	// pemeriksaan ini membuat penjaganya tidak pernah menyala, dan permintaan
	// dengan body kosong akan lolos lalu menulis baris audit tanpa satu pun
	// perubahan nyata.
	if in.RegistrationFee == nil && in.RenewalFee == nil &&
		!in.MenyentuhDollar() && in.Currency == nil && in.Notes == nil {
		return nil, httpx.BadRequest("tidak ada field yang diubah")
	}

	// Rupiah DITURUNKAN DI SINI, bukan dipercayakan ke klien.
	//
	// Begitu USD atau kurs berubah, kedua harga Rupiah dihitung ulang dari
	// nilai efektifnya: yang dikirim di permintaan ini, atau yang tersimpan
	// bila tidak dikirim. Klien yang mengirim Rupiah sendiri bersama USD akan
	// ditimpa, dan itu disengaja: dua sumber untuk satu angka yang tercetak di
	// invoice adalah dua angka yang suatu saat tidak sama.
	if in.MenyentuhDollar() {
		cur, err := s.repo.GetFees(ctx)
		if err != nil {
			return nil, err
		}
		usdReg, usdRen, rate := cur.RegistrationFeeUSD, cur.RenewalFeeUSD, cur.UsdRate
		if in.RegistrationFeeUSD != nil {
			usdReg = *in.RegistrationFeeUSD
		}
		if in.RenewalFeeUSD != nil {
			usdRen = *in.RenewalFeeUSD
		}
		if in.UsdRate != nil {
			rate = *in.UsdRate
		}
		reg, ren := domain.TurunkanRupiah(usdReg, rate), domain.TurunkanRupiah(usdRen, rate)
		in.RegistrationFee, in.RenewalFee = &reg, &ren
	}
	fees, err := s.repo.UpdateFees(ctx, in)
	if err == nil {
		_ = s.cache.Hapus(ctx, kunciBiaya)
	}
	return fees, err
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
