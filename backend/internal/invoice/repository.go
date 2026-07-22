package invoice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

const columns = `
	id, number, member_id, chapter_id, type, amount, currency,
	due_date, period_start, period_end, status,
	paper_id_invoice_id, paper_id_invoice_url, paper_id_payment_url, paper_id_sent_at,
	payment_provider, xendit_external_id, xendit_payment_id, xendit_payment_method,
	xendit_va_bank, xendit_va_number, xendit_qris_string, xendit_payment_status, xendit_expires_at,
	paid_at, paid_amount, notes, created_by, cancelled_by, cancelled_at, cancel_reason,
	created_at, updated_at`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface {
	Scan(dest ...any) error
}

// scan maps a row onto the domain model. Date columns come back as time.Time
// and are wrapped so the JSON stays YYYY-MM-DD.
func scan(row scannable) (*domain.Invoice, error) {
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("scan invoice: %w", err)
	}

	inv.DueDate = domain.NewDate(due)
	inv.PeriodStart = domain.NewDate(periodStart)
	inv.PeriodEnd = domain.NewDate(periodEnd)
	return &inv, nil
}

// List returns a filtered page plus the total row count for that filter.
func (r *Repository) List(ctx context.Context, f domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if f.Status != "" {
		if f.Status == "outstanding" {
			// Sama seperti UI: outstanding = sent + overdue.
			where = append(where, "status IN ('sent','overdue')")
		} else {
			add("status = $%d", domain.InvoiceStatus(f.Status))
		}
	}
	if f.Type != "" {
		add("type = $%d", domain.InvoiceType(f.Type))
	}
	if f.ChapterID != "" {
		add("chapter_id = $%d", f.ChapterID)
	}
	if f.MemberID != "" {
		add("member_id = $%d", f.MemberID)
	}
	if f.Search != "" {
		add("number ILIKE $%d", "%"+f.Search+"%")
	}
	if f.DueFrom != "" {
		add("due_date >= $%d", f.DueFrom)
	}
	if f.DueTo != "" {
		add("due_date <= $%d", f.DueTo)
	}
	if f.IssuedFrom != "" {
		add("created_at >= $%d", f.IssuedFrom)
	}
	if f.IssuedTo != "" {
		// inclusive: cover the whole end day
		add("created_at < ($%d::date + interval '1 day')", f.IssuedTo)
	}

	clause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM invoices WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung invoice: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(
		"SELECT %s FROM invoices WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		columns, clause, len(args)-1, len(args),
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ambil invoice: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Invoice, 0, f.Limit)
	for rows.Next() {
		inv, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *inv)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Invoice, error) {
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+" FROM invoices WHERE id = $1", id))
}

func (r *Repository) Create(ctx context.Context, in domain.CreateInvoiceInput, number, currency string) (*domain.Invoice, error) {
	const q = `
		INSERT INTO invoices (number, member_id, chapter_id, type, amount, currency,
		                      due_date, period_start, period_end, status, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'draft',$10,$11)
		RETURNING ` + columns
	return scan(r.db.QueryRow(ctx, q,
		number, in.MemberID, in.ChapterID, in.Type, in.Amount, currency,
		in.DueDate.Time, in.PeriodStart.Time, in.PeriodEnd.Time, in.Notes, in.CreatedBy,
	))
}

// Update writes only the fields present in the patch.
func (r *Repository) Update(ctx context.Context, id string, in domain.UpdateInvoiceInput) (*domain.Invoice, error) {
	sets := []string{"updated_at = now()"}
	args := []any{}
	set := func(col string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if in.Amount != nil {
		set("amount", *in.Amount)
	}
	if in.DueDate != nil {
		set("due_date", in.DueDate.Time)
	}
	if in.PeriodStart != nil {
		set("period_start", in.PeriodStart.Time)
	}
	if in.PeriodEnd != nil {
		set("period_end", in.PeriodEnd.Time)
	}
	if in.Notes != nil {
		set("notes", *in.Notes)
	}
	if in.Status != nil {
		set("status", *in.Status)
		switch *in.Status {
		case domain.StatusPaid:
			sets = append(sets, "paid_at = COALESCE(paid_at, now())", "paid_amount = COALESCE(paid_amount, amount)")
		case domain.StatusCancelled:
			sets = append(sets, "cancelled_at = COALESCE(cancelled_at, now())")
		}
	}
	if in.CancelReason != nil {
		set("cancel_reason", *in.CancelReason)
	}
	if in.CancelledBy != nil {
		set("cancelled_by", *in.CancelledBy)
	}

	args = append(args, id)
	q := fmt.Sprintf("UPDATE invoices SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), columns)

	return scan(r.db.QueryRow(ctx, q, args...))
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM invoices WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("hapus invoice: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

// CountPayments reports how many payments reference an invoice — used to block
// deletes that would orphan payment rows.
func (r *Repository) CountPayments(ctx context.Context, invoiceID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM payments WHERE invoice_id = $1", invoiceID).Scan(&n)
	return n, err
}

// NextNumber builds the next sequential number for the year, e.g. INV-2026-013.
func (r *Repository) NextNumber(ctx context.Context, year int) (string, error) {
	prefix := fmt.Sprintf("INV-%d-", year)
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM invoices WHERE number LIKE $1", prefix+"%").Scan(&count)
	if err != nil {
		return "", fmt.Errorf("hitung nomor invoice: %w", err)
	}
	return fmt.Sprintf("%s%03d", prefix, count+1), nil
}
