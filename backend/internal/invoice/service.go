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
	LateFeeRule(ctx context.Context) (domain.LateFeeRule, error)
}

// compile-time check that the Postgres repository satisfies the contract.
var _ Store = (*Repository)(nil)

type Service struct {
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	items, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	s.tempelkanDenda(ctx, items)
	return items, total, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Invoice, error) {
	inv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	satu := []domain.Invoice{*inv}
	s.tempelkanDenda(ctx, satu)
	return &satu[0], nil
}

// tempelkanDenda mengisi field LateFee yang terhitung.
//
// Aturannya dibaca SEKALI untuk seluruh halaman, bukan per invoice — daftar 200
// baris tidak boleh berarti 200 kali membaca app_settings.
//
// Kegagalan membacanya TIDAK menggagalkan permintaan. Denda hanya keterangan
// tambahan; menolak menampilkan seluruh daftar invoice karena satu baris
// pengaturan tidak terbaca akan menukar informasi tambahan dengan halaman yang
// sama sekali kosong.
func (s *Service) tempelkanDenda(ctx context.Context, items []domain.Invoice) {
	rule, err := s.repo.LateFeeRule(ctx)
	if err != nil || !rule.Aktif {
		return
	}
	now := time.Now()
	for i := range items {
		if f := rule.Hitung(items[i], now); f.Nominal > 0 {
			items[i].LateFee = &f
		}
	}
}

func (s *Service) Create(ctx context.Context, in domain.CreateInvoiceInput) (*domain.Invoice, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}

	currency := "IDR"
	if in.Currency != nil && *in.Currency != "" {
		currency = *in.Currency
	}

	// Nomor kosong berarti "bangkitkan". Sengaja TIDAK dibangkitkan di sini:
	// membaca nomor berikutnya lalu menulisnya lewat panggilan terpisah adalah
	// balapan — dua pembuatan serentak membaca hitungan yang sama. Repository
	// membangkitkannya di dalam transaksi Create, di bawah advisory lock.
	number := ""
	if in.Number != nil {
		number = *in.Number
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

	// Penolakan cepat dengan pesan yang jelas. Pemeriksaan yang MENGIKAT ada di
	// dalam transaksi repository, setelah barisnya dikunci — status di sini
	// sudah bisa basi sebelum penulisan terjadi.
	if err := current.Status.ValidateUpdateFrom(in); err != nil {
		return nil, httpx.Conflict(err.Error())
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
