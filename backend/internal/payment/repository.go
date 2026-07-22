package payment

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
	id, invoice_id, amount, paid_at, payment_method,
	paper_id_payment_id, paper_id_status, xendit_payment_id, xendit_status,
	proof_url, note, created_at`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface {
	Scan(dest ...any) error
}

func scan(row scannable) (*domain.Payment, error) {
	var p domain.Payment
	err := row.Scan(
		&p.ID, &p.InvoiceID, &p.Amount, &p.PaidAt, &p.PaymentMethod,
		&p.PaperIDPaymentID, &p.PaperIDStatus, &p.XenditPaymentID, &p.XenditStatus,
		&p.ProofURL, &p.Note, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("scan payment: %w", err)
	}
	return &p, nil
}

func (r *Repository) List(ctx context.Context, f domain.PaymentFilter) ([]domain.Payment, int, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if f.InvoiceID != "" {
		add("invoice_id = $%d", f.InvoiceID)
	}
	if f.Method != "" {
		add("payment_method = $%d", f.Method)
	}
	if f.PaidFrom != "" {
		add("paid_at >= $%d", f.PaidFrom)
	}
	if f.PaidTo != "" {
		add("paid_at < ($%d::date + interval '1 day')", f.PaidTo)
	}

	clause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM payments WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung payment: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(
		"SELECT %s FROM payments WHERE %s ORDER BY paid_at DESC LIMIT $%d OFFSET $%d",
		columns, clause, len(args)-1, len(args),
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ambil payment: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Payment, 0, f.Limit)
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *p)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+" FROM payments WHERE id = $1", id))
}

// CreateAndSettle inserts the payment and — when settle is true — marks the
// invoice paid in the SAME transaction, so we can never end up with a recorded
// payment against an unpaid invoice.
func (r *Repository) CreateAndSettle(
	ctx context.Context,
	in domain.CreatePaymentInput,
	paidAt time.Time,
	settle bool,
) (*domain.Payment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the invoice row so concurrent settlements can't race.
	var status domain.InvoiceStatus
	err = tx.QueryRow(ctx, "SELECT status FROM invoices WHERE id = $1 FOR UPDATE", in.InvoiceID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("invoice tidak ditemukan")
		}
		return nil, fmt.Errorf("ambil invoice: %w", err)
	}
	if status == domain.StatusCancelled {
		return nil, httpx.Conflict("invoice sudah dibatalkan — tidak bisa menerima pembayaran")
	}

	const insert = `
		INSERT INTO payments (invoice_id, amount, paid_at, payment_method, proof_url, note)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING ` + columns

	p, err := scan(tx.QueryRow(ctx, insert,
		in.InvoiceID, in.Amount, paidAt, in.PaymentMethod, in.ProofURL, in.Note))
	if err != nil {
		return nil, err
	}

	if settle && status != domain.StatusPaid {
		_, err = tx.Exec(ctx, `
			UPDATE invoices
			   SET status = 'paid', paid_at = $2, paid_amount = COALESCE(paid_amount, $3), updated_at = now()
			 WHERE id = $1`, in.InvoiceID, paidAt, in.Amount)
		if err != nil {
			return nil, fmt.Errorf("tandai invoice lunas: %w", err)
		}

		// Settling here bypasses invoice.Repository.Update, so the timeline
		// entry has to be written on this path too — same transaction.
		paid := domain.StatusPaid
		_, err = tx.Exec(ctx, `
			INSERT INTO invoice_audit_log (invoice_id, action, old_status, new_status, notes)
			VALUES ($1, 'paid', $2, $3, $4)`,
			in.InvoiceID, status, paid, in.Note)
		if err != nil {
			return nil, fmt.Errorf("catat audit log: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaksi: %w", err)
	}
	return p, nil
}

func (r *Repository) Update(ctx context.Context, id string, in domain.UpdatePaymentInput) (*domain.Payment, error) {
	sets := []string{}
	args := []any{}
	set := func(col string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if in.Amount != nil {
		set("amount", *in.Amount)
	}
	if in.PaidAt != nil {
		set("paid_at", *in.PaidAt)
	}
	if in.PaymentMethod != nil {
		set("payment_method", *in.PaymentMethod)
	}
	if in.ProofURL != nil {
		set("proof_url", *in.ProofURL)
	}
	if in.Note != nil {
		set("note", *in.Note)
	}
	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf("UPDATE payments SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), columns)
	return scan(r.db.QueryRow(ctx, q, args...))
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM payments WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("hapus payment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}
