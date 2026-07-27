package paperid

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// Sendable is everything needed to push one invoice to Paper.id: the invoice
// itself plus the member's contact details (Paper.id requires a phone).
type Sendable struct {
	ID       string
	Number   string
	Amount   int64
	Type     domain.InvoiceType
	Status   domain.InvoiceStatus
	DueDate  time.Time
	MemberID string
	Name     string
	Email    string
	Phone    string
}

// GetSendable loads the invoice + member for the send path.
func (r *Repository) GetSendable(ctx context.Context, invoiceID string) (*Sendable, error) {
	const q = `
		SELECT i.id, i.number, i.amount, i.type, i.status, i.due_date,
		       m.id, m.name, coalesce(m.email,''), coalesce(m.phone,'')
		FROM invoices i JOIN members m ON m.id = i.member_id
		WHERE i.id = $1`

	var s Sendable
	err := r.db.QueryRow(ctx, q, invoiceID).Scan(
		&s.ID, &s.Number, &s.Amount, &s.Type, &s.Status, &s.DueDate,
		&s.MemberID, &s.Name, &s.Email, &s.Phone,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("invoice tidak ditemukan")
		}
		return nil, fmt.Errorf("ambil invoice untuk Paper.id: %w", err)
	}
	return &s, nil
}

// MarkSent records the Paper.id result and moves the invoice to `sent`, all in
// one transaction with the row locked, so a status change and its audit row can
// never drift apart. It re-checks the transition under the lock: only a draft
// can be sent.
func (r *Repository) MarkSent(
	ctx context.Context, invoiceID string, res CreateResult, dueDate, sentAt time.Time, actor string,
) (*domain.Invoice, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	var status domain.InvoiceStatus
	if err := tx.QueryRow(ctx,
		"SELECT status FROM invoices WHERE id = $1 FOR UPDATE", invoiceID,
	).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("invoice tidak ditemukan")
		}
		return nil, fmt.Errorf("kunci invoice: %w", err)
	}
	if status != domain.StatusDraft {
		return nil, httpx.Conflict("hanya invoice draft yang bisa dikirim ke Paper.id")
	}

	const upd = `
		UPDATE invoices SET
		  status = 'sent',
		  due_date = $2,
		  payment_provider = 'paper_id',
		  paper_id_invoice_id = $3,
		  paper_id_invoice_url = $4,
		  paper_id_payment_url = $5,
		  paper_id_sent_at = $6,
		  updated_at = now()
		WHERE id = $1
		RETURNING ` + invoiceColumns

	inv, err := scanInvoice(tx.QueryRow(ctx, upd,
		invoiceID, dueDate, res.PaperInvoiceID, res.InvoicePDFURL, res.PaymentURL, sentAt))
	if err != nil {
		return nil, err
	}

	draft, sent := domain.StatusDraft, domain.StatusSent
	if _, err := tx.Exec(ctx, `
		INSERT INTO invoice_audit_log (invoice_id, action, old_status, new_status, actor_name, notes)
		VALUES ($1, 'sent', $2, $3, $4, 'Dikirim ke Paper.id')`,
		invoiceID, draft, sent, actor); err != nil {
		return nil, fmt.Errorf("catat audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("simpan hasil Paper.id: %w", err)
	}
	return inv, nil
}

// SettleByRef is the webhook path: find the invoice by its Paper.id id or
// number, record the payment, and mark it paid — in one transaction.
//
// Returns false (without error) when the invoice is already paid or cancelled,
// so a duplicate callback is a no-op. Paper.id retries, so this must be
// idempotent.
func (r *Repository) SettleByRef(
	ctx context.Context, paperInvoiceID, number, method, status string, amount int64, paidAt time.Time,
) (settled bool, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	// Match on the Paper.id id first (exact), then fall back to our number.
	var invoiceID string
	var current domain.InvoiceStatus
	err = tx.QueryRow(ctx, `
		SELECT id, status FROM invoices
		WHERE ($1 <> '' AND paper_id_invoice_id = $1) OR ($2 <> '' AND number = $2)
		ORDER BY (paper_id_invoice_id = $1) DESC
		LIMIT 1
		FOR UPDATE`, paperInvoiceID, number).Scan(&invoiceID, &current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, httpx.NotFound("invoice untuk callback tersebut tidak ditemukan")
		}
		return false, fmt.Errorf("cari invoice: %w", err)
	}

	if current == domain.StatusPaid || current == domain.StatusCancelled {
		return false, nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO payments (invoice_id, amount, paid_at, payment_method, paper_id_payment_id, paper_id_status)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		invoiceID, amount, paidAt, method, paperInvoiceID, status); err != nil {
		return false, fmt.Errorf("simpan pembayaran: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE invoices SET status = 'paid', paid_at = $2,
		  paid_amount = COALESCE(paid_amount, $3), updated_at = now()
		WHERE id = $1`, invoiceID, paidAt, amount); err != nil {
		return false, fmt.Errorf("tandai invoice lunas: %w", err)
	}

	paid := domain.StatusPaid
	if _, err := tx.Exec(ctx, `
		INSERT INTO invoice_audit_log (invoice_id, action, old_status, new_status, actor_name, notes)
		VALUES ($1,'paid',$2,$3,'Paper.id','pembayaran otomatis via callback Paper.id')`,
		invoiceID, current, paid); err != nil {
		return false, fmt.Errorf("catat audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit pelunasan: %w", err)
	}
	return true, nil
}

// PaperInvoiceID returns the Paper.id id stored on one of our invoices, empty
// when it was never pushed. Used by the test console to build a callback that
// matches the way the real one arrives.
func (r *Repository) PaperInvoiceID(ctx context.Context, invoiceID string) (string, error) {
	var id *string
	err := r.db.QueryRow(ctx,
		"SELECT paper_id_invoice_id FROM invoices WHERE id = $1", invoiceID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", httpx.NotFound("invoice tidak ditemukan")
		}
		return "", fmt.Errorf("ambil paper_id_invoice_id: %w", err)
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

func (r *Repository) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := r.db.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = $1", key).Scan(&v)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("ambil setting %s: %w", key, err)
	}
	return v, nil
}

// invoiceColumns / scanInvoice mirror the invoice package so MarkSent can return
// a fully-populated domain.Invoice without importing it.
const invoiceColumns = `
	id, number, member_id, chapter_id, type, amount, currency,
	due_date, period_start, period_end, status,
	paper_id_invoice_id, paper_id_invoice_url, paper_id_payment_url, paper_id_sent_at,
	payment_provider, xendit_external_id, xendit_payment_id, xendit_payment_method,
	xendit_va_bank, xendit_va_number, xendit_qris_string, xendit_payment_status, xendit_expires_at,
	paid_at, paid_amount, notes, created_by, cancelled_by, cancelled_at, cancel_reason,
	created_at, updated_at`

type scannable interface{ Scan(dest ...any) error }

func scanInvoice(row scannable) (*domain.Invoice, error) {
	var inv domain.Invoice
	var due, periodStart, periodEnd time.Time
	err := row.Scan(
		&inv.ID, &inv.Number, &inv.MemberID, &inv.ChapterID, &inv.Type, &inv.Amount, &inv.Currency,
		&due, &periodStart, &periodEnd, &inv.Status,
		&inv.PaperIDInvoiceID, &inv.PaperIDInvoiceURL, &inv.PaperIDPaymentURL, &inv.PaperIDSentAt,
		&inv.PaymentProvider, &inv.XenditExternalID, &inv.XenditPaymentID, &inv.XenditPaymentMethod,
		&inv.XenditVaBank, &inv.XenditVaNumber, &inv.XenditQrisString, &inv.XenditPaymentStatus, &inv.XenditExpiresAt,
		&inv.PaidAt, &inv.PaidAmount, &inv.Notes, &inv.CreatedBy, &inv.CancelledBy, &inv.CancelledAt, &inv.CancelReason,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan invoice: %w", err)
	}
	inv.DueDate = domain.NewDate(due)
	inv.PeriodStart = domain.NewDate(periodStart)
	inv.PeriodEnd = domain.NewDate(periodEnd)
	return &inv, nil
}
