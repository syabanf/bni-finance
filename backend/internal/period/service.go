package period

import (
	"context"
	"fmt"
	"strings"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// Store adalah kontrak persistensi yang dibutuhkan service.
type Store interface {
	List(ctx context.Context) ([]domain.RenewalPeriod, error)
	GetByID(ctx context.Context, id string) (*domain.RenewalPeriod, error)
	Bentrok(ctx context.Context, p domain.RenewalPeriod, cekRentang bool) (*domain.RenewalPeriod, error)
	Create(ctx context.Context, calonID []string, p domain.RenewalPeriod) (*domain.RenewalPeriod, error)
	Update(ctx context.Context, p domain.RenewalPeriod) (*domain.RenewalPeriod, error)
	JumlahPermintaan(ctx context.Context, id string) (int, error)
	Delete(ctx context.Context, id string) error
}

var _ Store = (*Repository)(nil)

type Service struct{ repo Store }

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context) ([]domain.RenewalPeriod, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, in domain.SaveRenewalPeriodInput) (*domain.RenewalPeriod, error) {
	var p domain.RenewalPeriod
	if err := in.Terapkan(&p); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	if err := s.tolakBentrok(ctx, p, true); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, domain.KodePeriode(p.StartDate, p.EndDate), p)
}

func (s *Service) Update(ctx context.Context, id string, in domain.SaveRenewalPeriodInput) (*domain.RenewalPeriod, error) {
	lama, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p := *lama
	if err := in.Terapkan(&p); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	// Rentang hanya diperiksa bila tanggalnya berubah. Periode lama berbentuk
	// tahun beririsan dengan semesternya, dan mengganti nama salah satunya
	// tidak boleh tertolak karena irisan yang sudah ada sejak awal.
	geser := !p.StartDate.Equal(lama.StartDate.Time) || !p.EndDate.Equal(lama.EndDate.Time)
	if err := s.tolakBentrok(ctx, p, geser); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, p)
}

// tolakBentrok menolak nama yang sudah dipakai dan rentang yang beririsan.
//
// Dua periode yang beririsan membuat satu member jatuh tempo di keduanya, dan
// indeks unik (member, periode) tidak mencegah ia ditanyakan dua kali.
func (s *Service) tolakBentrok(ctx context.Context, p domain.RenewalPeriod, cekRentang bool) error {
	lain, err := s.repo.Bentrok(ctx, p, cekRentang)
	if err != nil || lain == nil {
		return err
	}
	if strings.EqualFold(lain.Name, p.Name) {
		return httpx.Conflict(fmt.Sprintf("nama %q sudah dipakai periode lain", lain.Name))
	}
	return httpx.Conflict(fmt.Sprintf("rentang ini beririsan dengan periode %q (%s sampai %s)",
		lain.Name, lain.StartDate, lain.EndDate))
}

// Delete menolak menghapus periode yang sudah punya permintaan konfirmasi.
// Jawaban MC di periode itu menjadi dasar invoice renewal.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	n, err := s.repo.JumlahPermintaan(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return httpx.Conflict(fmt.Sprintf(
			"periode ini dipakai %d permintaan konfirmasi, jadi tidak bisa dihapus", n))
	}
	return s.repo.Delete(ctx, id)
}
