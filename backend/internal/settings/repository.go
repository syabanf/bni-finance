// Package settings exposes the fee configuration (a singleton row) and the
// app_settings key/value table used by the sync and payment flows.
package settings

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// FeeSettingsID is the singleton primary key used by the schema.
const FeeSettingsID = "default"

const feeColumns = `id, registration_fee, renewal_fee, currency, notes, updated_by, updated_at, created_at`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface {
	Scan(dest ...any) error
}

func scanFees(row scannable) (*domain.FeeSettings, error) {
	var f domain.FeeSettings
	err := row.Scan(&f.ID, &f.RegistrationFee, &f.RenewalFee, &f.Currency,
		&f.Notes, &f.UpdatedBy, &f.UpdatedAt, &f.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("scan fee settings: %w", err)
	}
	return &f, nil
}

func (r *Repository) GetFees(ctx context.Context) (*domain.FeeSettings, error) {
	return scanFees(r.db.QueryRow(ctx,
		"SELECT "+feeColumns+" FROM fee_settings WHERE id = $1", FeeSettingsID))
}

func (r *Repository) UpdateFees(ctx context.Context, in domain.UpdateFeeSettingsInput) (*domain.FeeSettings, error) {
	sets := []string{"updated_at = now()"}
	args := []any{}
	set := func(col string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if in.RegistrationFee != nil {
		set("registration_fee", *in.RegistrationFee)
	}
	if in.RenewalFee != nil {
		set("renewal_fee", *in.RenewalFee)
	}
	if in.Currency != nil {
		set("currency", *in.Currency)
	}
	if in.Notes != nil {
		set("notes", *in.Notes)
	}
	if in.UpdatedBy != nil {
		set("updated_by", *in.UpdatedBy)
	}

	args = append(args, FeeSettingsID)
	q := fmt.Sprintf("UPDATE fee_settings SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), feeColumns)

	return scanFees(r.db.QueryRow(ctx, q, args...))
}

// --- app_settings -----------------------------------------------------------

func (r *Repository) ListApp(ctx context.Context) ([]domain.AppSetting, error) {
	rows, err := r.db.Query(ctx, "SELECT key, value, updated_at FROM app_settings ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("ambil app settings: %w", err)
	}
	defer rows.Close()

	items := make([]domain.AppSetting, 0, 16)
	for rows.Next() {
		var s domain.AppSetting
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan app setting: %w", err)
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *Repository) GetApp(ctx context.Context, key string) (*domain.AppSetting, error) {
	var s domain.AppSetting
	err := r.db.QueryRow(ctx,
		"SELECT key, value, updated_at FROM app_settings WHERE key = $1", key,
	).Scan(&s.Key, &s.Value, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("ambil app setting: %w", err)
	}
	return &s, nil
}

// SetApp upserts a key. The table is the source of truth for feature flags like
// self_payment_mode, so writes must not depend on the row already existing.
func (r *Repository) SetApp(ctx context.Context, key, value string) (*domain.AppSetting, error) {
	const q = `
		INSERT INTO app_settings (key, value, updated_at) VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = now()
		RETURNING key, value, updated_at`

	var s domain.AppSetting
	if err := r.db.QueryRow(ctx, q, key, value).Scan(&s.Key, &s.Value, &s.UpdatedAt); err != nil {
		return nil, fmt.Errorf("simpan app setting: %w", err)
	}
	return &s, nil
}

func (r *Repository) DeleteApp(ctx context.Context, key string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM app_settings WHERE key = $1", key)
	if err != nil {
		return fmt.Errorf("hapus app setting: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}
