package api_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// In-memory stores standing in for Postgres. They are deliberately
// mutex-guarded so `go test -race` can prove the HTTP layer is safe under
// concurrency (a real DB would hide races behind its own locking).

type fakeInvoiceStore struct {
	mu    sync.RWMutex
	items map[string]domain.Invoice
	seq   int
}

func newFakeInvoiceStore() *fakeInvoiceStore {
	return &fakeInvoiceStore{items: make(map[string]domain.Invoice)}
}

func (s *fakeInvoiceStore) List(_ context.Context, f domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.Invoice, 0, len(s.items))
	for _, inv := range s.items {
		if f.Status != "" {
			if f.Status == "outstanding" {
				if inv.Status != domain.StatusSent && inv.Status != domain.StatusOverdue {
					continue
				}
			} else if string(inv.Status) != f.Status {
				continue
			}
		}
		if f.ChapterID != "" && inv.ChapterID != f.ChapterID {
			continue
		}
		if f.MemberID != "" && inv.MemberID != f.MemberID {
			continue
		}
		out = append(out, inv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })

	total := len(out)
	if f.Offset >= total {
		return []domain.Invoice{}, total, nil
	}
	end := f.Offset + f.Limit
	if end > total {
		end = total
	}
	return out[f.Offset:end], total, nil
}

func (s *fakeInvoiceStore) GetByID(_ context.Context, id string) (*domain.Invoice, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inv, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	return &inv, nil
}

func (s *fakeInvoiceStore) Create(_ context.Context, in domain.CreateInvoiceInput, number, currency string) (*domain.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	now := time.Now().UTC()
	inv := domain.Invoice{
		ID:          fmt.Sprintf("inv-%06d", s.seq),
		Number:      number,
		MemberID:    in.MemberID,
		ChapterID:   in.ChapterID,
		Type:        in.Type,
		Amount:      in.Amount,
		Currency:    currency,
		DueDate:     in.DueDate,
		PeriodStart: in.PeriodStart,
		PeriodEnd:   in.PeriodEnd,
		Status:      domain.StatusDraft,
		Notes:       in.Notes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.items[inv.ID] = inv
	return &inv, nil
}

func (s *fakeInvoiceStore) Update(_ context.Context, id string, in domain.UpdateInvoiceInput) (*domain.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	if in.Amount != nil {
		inv.Amount = *in.Amount
	}
	if in.Status != nil {
		inv.Status = *in.Status
		if *in.Status == domain.StatusPaid && inv.PaidAt == nil {
			now := time.Now().UTC()
			inv.PaidAt = &now
			amt := inv.Amount
			inv.PaidAmount = &amt
		}
	}
	if in.Notes != nil {
		inv.Notes = in.Notes
	}
	inv.UpdatedAt = time.Now().UTC()
	s.items[id] = inv
	return &inv, nil
}

func (s *fakeInvoiceStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return httpx.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *fakeInvoiceStore) CountPayments(_ context.Context, _ string) (int, error) { return 0, nil }

func (s *fakeInvoiceStore) NextNumber(_ context.Context, year int) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fmt.Sprintf("INV-%d-%03d", year, len(s.items)+1), nil
}

func (s *fakeInvoiceStore) count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// --- payments ---------------------------------------------------------------

type fakePaymentStore struct {
	mu       sync.RWMutex
	items    map[string]domain.Payment
	seq      int
	invoices *fakeInvoiceStore
}

func newFakePaymentStore(inv *fakeInvoiceStore) *fakePaymentStore {
	return &fakePaymentStore{items: make(map[string]domain.Payment), invoices: inv}
}

func (s *fakePaymentStore) List(_ context.Context, f domain.PaymentFilter) ([]domain.Payment, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Payment, 0, len(s.items))
	for _, p := range s.items {
		if f.InvoiceID != "" && p.InvoiceID != f.InvoiceID {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PaidAt.After(out[j].PaidAt) })

	total := len(out)
	if f.Offset >= total {
		return []domain.Payment{}, total, nil
	}
	end := f.Offset + f.Limit
	if end > total {
		end = total
	}
	return out[f.Offset:end], total, nil
}

func (s *fakePaymentStore) GetByID(_ context.Context, id string) (*domain.Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	return &p, nil
}

// CreateAndSettle mirrors the real repository: insert + settle the invoice
// atomically (here: under one lock).
func (s *fakePaymentStore) CreateAndSettle(
	_ context.Context, in domain.CreatePaymentInput, paidAt time.Time, settle bool,
) (*domain.Payment, error) {
	inv, err := s.invoices.GetByID(context.Background(), in.InvoiceID)
	if err != nil {
		return nil, httpx.NotFound("invoice tidak ditemukan")
	}
	if inv.Status == domain.StatusCancelled {
		return nil, httpx.Conflict("invoice sudah dibatalkan — tidak bisa menerima pembayaran")
	}

	s.mu.Lock()
	s.seq++
	p := domain.Payment{
		ID:            fmt.Sprintf("pay-%06d", s.seq),
		InvoiceID:     in.InvoiceID,
		Amount:        in.Amount,
		PaidAt:        paidAt,
		PaymentMethod: in.PaymentMethod,
		ProofURL:      in.ProofURL,
		Note:          in.Note,
		CreatedAt:     time.Now().UTC(),
	}
	s.items[p.ID] = p
	s.mu.Unlock()

	if settle && inv.Status != domain.StatusPaid {
		paid := domain.StatusPaid
		if _, err := s.invoices.Update(context.Background(), in.InvoiceID, domain.UpdateInvoiceInput{Status: &paid}); err != nil {
			return nil, err
		}
	}
	return &p, nil
}

func (s *fakePaymentStore) Update(_ context.Context, id string, in domain.UpdatePaymentInput) (*domain.Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	if in.Amount != nil {
		p.Amount = *in.Amount
	}
	if in.Note != nil {
		p.Note = in.Note
	}
	s.items[id] = p
	return &p, nil
}

func (s *fakePaymentStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return httpx.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *fakePaymentStore) count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
