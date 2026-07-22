// Package audit exposes the invoice timeline. Status changes are written
// automatically inside the invoice/payment transactions; this package only
// reads them back and accepts manual annotations.
package audit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

const columns = `id, invoice_id, action, old_status, new_status, actor_id, actor_name, notes, created_at`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// ListByInvoice returns the timeline newest-first.
func (r *Repository) ListByInvoice(ctx context.Context, invoiceID string, limit int) ([]domain.AuditEntry, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+columns+" FROM invoice_audit_log WHERE invoice_id = $1 ORDER BY created_at DESC LIMIT $2",
		invoiceID, limit)
	if err != nil {
		return nil, fmt.Errorf("ambil audit log: %w", err)
	}
	defer rows.Close()

	items := make([]domain.AuditEntry, 0, limit)
	for rows.Next() {
		var e domain.AuditEntry
		if err := rows.Scan(&e.ID, &e.InvoiceID, &e.Action, &e.OldStatus, &e.NewStatus,
			&e.ActorID, &e.ActorName, &e.Notes, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, invoiceID string, in domain.CreateAuditEntryInput) (*domain.AuditEntry, error) {
	action := domain.AuditUpdated
	if in.Action != nil {
		action = *in.Action
	}

	const q = `
		INSERT INTO invoice_audit_log (invoice_id, action, actor_id, actor_name, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + columns

	var e domain.AuditEntry
	err := r.db.QueryRow(ctx, q, invoiceID, action, in.ActorID, in.ActorName, in.Notes).
		Scan(&e.ID, &e.InvoiceID, &e.Action, &e.OldStatus, &e.NewStatus,
			&e.ActorID, &e.ActorName, &e.Notes, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("simpan audit log: %w", err)
	}
	return &e, nil
}

// InvoiceExists guards the write path so a note can't be attached to a missing
// invoice (the FK would reject it with a far less helpful message).
func (r *Repository) InvoiceExists(ctx context.Context, invoiceID string) error {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM invoices WHERE id = $1)", invoiceID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("cek invoice: %w", err)
	}
	if !exists {
		return httpx.ErrNotFound
	}
	return nil
}
