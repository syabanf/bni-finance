// Package publicpay serves the unauthenticated payment page and the Xendit
// integration, replacing the get-public-invoice, xendit-create-payment and
// xendit-webhook edge functions.
package publicpay

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

// PublicInvoice is a deliberately narrow projection. The page is reachable by
// anyone holding the link, so it exposes the member's NAME and nothing else —
// no email, phone, or company.
type PublicInvoice struct {
	ID       string               `json:"id"`
	Number   string               `json:"number"`
	Type     domain.InvoiceType   `json:"type"`
	Amount   int64                `json:"amount"`
	Currency string               `json:"currency"`
	Status   domain.InvoiceStatus `json:"status"`

	DueDate     domain.Date `json:"dueDate"`
	PeriodStart domain.Date `json:"periodStart"`
	PeriodEnd   domain.Date `json:"periodEnd"`

	MemberName  string  `json:"memberName"`
	ChapterName *string `json:"chapterName,omitempty"`
	Notes       *string `json:"notes,omitempty"`

	PaymentProvider     *string    `json:"paymentProvider,omitempty"`
	PaperIDPaymentURL   *string    `json:"paperIdPaymentUrl,omitempty"`
	XenditPaymentMethod *string    `json:"xenditPaymentMethod,omitempty"`
	XenditVaBank        *string    `json:"xenditVaBank,omitempty"`
	XenditVaNumber      *string    `json:"xenditVaNumber,omitempty"`
	XenditQrisString    *string    `json:"xenditQrisString,omitempty"`
	XenditPaymentStatus *string    `json:"xenditPaymentStatus,omitempty"`
	XenditExpiresAt     *time.Time `json:"xenditExpiresAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

const publicColumns = `
	i.id, i.number, i.type, i.amount, i.currency, i.status,
	i.due_date, i.period_start, i.period_end,
	m.name, c.display_name, i.notes,
	i.payment_provider, i.paper_id_payment_url,
	i.xendit_payment_method, i.xendit_va_bank, i.xendit_va_number,
	i.xendit_qris_string, i.xendit_payment_status, i.xendit_expires_at,
	i.created_at`

func (r *Repository) GetPublicInvoice(ctx context.Context, id string) (*PublicInvoice, error) {
	const q = `SELECT ` + publicColumns + `
		FROM invoices i
		JOIN members m  ON m.id = i.member_id
		LEFT JOIN chapters c ON c.id = i.chapter_id
		WHERE i.id = $1`

	var p PublicInvoice
	var due, start, end time.Time
	err := r.db.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.Number, &p.Type, &p.Amount, &p.Currency, &p.Status,
		&due, &start, &end,
		&p.MemberName, &p.ChapterName, &p.Notes,
		&p.PaymentProvider, &p.PaperIDPaymentURL,
		&p.XenditPaymentMethod, &p.XenditVaBank, &p.XenditVaNumber,
		&p.XenditQrisString, &p.XenditPaymentStatus, &p.XenditExpiresAt,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("invoice tidak ditemukan")
		}
		return nil, fmt.Errorf("ambil invoice publik: %w", err)
	}
	p.DueDate = domain.NewDate(due)
	p.PeriodStart = domain.NewDate(start)
	p.PeriodEnd = domain.NewDate(end)
	return &p, nil
}

// billable is the minimum needed to open a Xendit charge.
type billable struct {
	ID         string
	Number     string
	Amount     int64
	Status     domain.InvoiceStatus
	MemberName string
}

func (r *Repository) getBillable(ctx context.Context, id string) (*billable, error) {
	const q = `
		SELECT i.id, i.number, i.amount, i.status, m.name
		FROM invoices i JOIN members m ON m.id = i.member_id
		WHERE i.id = $1`

	var b billable
	if err := r.db.QueryRow(ctx, q, id).Scan(&b.ID, &b.Number, &b.Amount, &b.Status, &b.MemberName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("invoice tidak ditemukan")
		}
		return nil, fmt.Errorf("ambil invoice: %w", err)
	}
	return &b, nil
}

func (r *Repository) saveVirtualAccount(ctx context.Context, invoiceID, externalID, paymentID, bank, number string, expires time.Time) error {
	const q = `
		UPDATE invoices SET
		  payment_provider = 'xendit', xendit_external_id = $2, xendit_payment_id = $3,
		  xendit_payment_method = 'va', xendit_va_bank = $4, xendit_va_number = $5,
		  xendit_qris_string = NULL, xendit_payment_status = 'PENDING',
		  xendit_expires_at = $6, updated_at = now()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, invoiceID, externalID, paymentID, bank, number, expires)
	if err != nil {
		return fmt.Errorf("simpan VA: %w", err)
	}
	return nil
}

func (r *Repository) saveQris(ctx context.Context, invoiceID, externalID, paymentID, qrString string, expires *time.Time) error {
	const q = `
		UPDATE invoices SET
		  payment_provider = 'xendit', xendit_external_id = $2, xendit_payment_id = $3,
		  xendit_payment_method = 'qris', xendit_qris_string = $4,
		  xendit_va_bank = NULL, xendit_va_number = NULL,
		  xendit_payment_status = 'PENDING', xendit_expires_at = $5, updated_at = now()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, invoiceID, externalID, paymentID, qrString, expires)
	if err != nil {
		return fmt.Errorf("simpan QRIS: %w", err)
	}
	return nil
}

// SettleByExternalID is the webhook path. It looks the invoice up by the
// reference we generated, then records the payment and marks the invoice paid
// in one transaction — the same guarantee the manual path gives.
//
// Returns false when the callback refers to an invoice that is already paid, so
// a duplicate delivery is a no-op rather than a second payment row. Xendit
// retries, so this has to be idempotent.
func (r *Repository) SettleByExternalID(ctx context.Context, externalID, xenditPaymentID, status string, amount int64, paidAt time.Time) (settled bool, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	var invoiceID string
	var current domain.InvoiceStatus
	err = tx.QueryRow(ctx,
		"SELECT id, status FROM invoices WHERE xendit_external_id = $1 FOR UPDATE", externalID,
	).Scan(&invoiceID, &current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, httpx.NotFound("invoice untuk referensi tersebut tidak ditemukan")
		}
		return false, fmt.Errorf("cari invoice: %w", err)
	}

	if current == domain.StatusPaid || current == domain.StatusCancelled {
		return false, nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO payments (invoice_id, amount, paid_at, payment_method, xendit_payment_id, xendit_status)
		VALUES ($1,$2,$3,'xendit',$4,$5)`,
		invoiceID, amount, paidAt, xenditPaymentID, status); err != nil {
		return false, fmt.Errorf("simpan pembayaran: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE invoices SET status = 'paid', paid_at = $2,
		  paid_amount = COALESCE(paid_amount, $3),
		  xendit_payment_status = $4, updated_at = now()
		WHERE id = $1`, invoiceID, paidAt, amount, status); err != nil {
		return false, fmt.Errorf("tandai invoice lunas: %w", err)
	}

	paid := domain.StatusPaid
	if _, err := tx.Exec(ctx, `
		INSERT INTO invoice_audit_log (invoice_id, action, old_status, new_status, actor_name, notes)
		VALUES ($1,'paid',$2,$3,'Xendit','pembayaran otomatis via webhook')`,
		invoiceID, current, paid); err != nil {
		return false, fmt.Errorf("catat audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit transaksi: %w", err)
	}
	return true, nil
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
