// Package period mengelola master data periode konfirmasi renewal.
package period

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

const columns = `p.id, p.name, p.start_date, p.end_date, p.created_at, p.updated_at`

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func scan(row pgx.Row, tambahan ...any) (*domain.RenewalPeriod, error) {
	var p domain.RenewalPeriod
	tujuan := append([]any{&p.ID, &p.Name, &p.StartDate, &p.EndDate, &p.CreatedAt, &p.UpdatedAt}, tambahan...)
	if err := row.Scan(tujuan...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("scan periode: %w", err)
	}
	return &p, nil
}

// List mengembalikan seluruh periode, dari yang paling awal.
//
// Daftarnya tidak dibatasi chapter: periode berlaku nasional. Yang dibatasi
// adalah jumlah permintaannya, supaya ST dan MC tidak membaca angka chapter lain.
func (r *Repository) List(ctx context.Context) ([]domain.RenewalPeriod, error) {
	batas := ""
	args := []any{}
	if klausa, arg, pakai := scope.Chapter(ctx).SQL("r.chapter_id", 1); klausa != "" {
		batas = " AND " + klausa
		if pakai {
			args = append(args, arg)
		}
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+columns+`,
		       (SELECT COUNT(*) FROM renewal_requests r WHERE r.period = p.id`+batas+`)
		FROM renewal_periods p
		ORDER BY p.start_date, p.end_date, p.id`, args...)
	if err != nil {
		return nil, fmt.Errorf("ambil periode: %w", err)
	}
	defer rows.Close()

	out := []domain.RenewalPeriod{}
	for rows.Next() {
		var jumlah int
		p, err := scan(rows, &jumlah)
		if err != nil {
			return nil, err
		}
		p.RequestCount = jumlah
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.RenewalPeriod, error) {
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+" FROM renewal_periods p WHERE p.id = $1", id))
}

// Bentrok mencari periode lain yang namanya sama atau rentangnya beririsan
// dengan p. Nama dibandingkan tanpa membedakan huruf besar dan kecil.
func (r *Repository) Bentrok(ctx context.Context, p domain.RenewalPeriod, cekRentang bool) (*domain.RenewalPeriod, error) {
	lain, err := scan(r.db.QueryRow(ctx, `
		SELECT `+columns+` FROM renewal_periods p
		WHERE p.id <> $1
		  AND (lower(p.name) = lower($2) OR ($5 AND p.start_date <= $4 AND p.end_date >= $3))
		ORDER BY p.start_date LIMIT 1`,
		p.ID, p.Name, p.StartDate.Time, p.EndDate.Time, cekRentang))
	if errors.Is(err, httpx.ErrNotFound) {
		return nil, nil
	}
	return lain, err
}

// Create menyimpan periode baru dengan id pertama dari calon yang belum terpakai.
func (r *Repository) Create(ctx context.Context, calonID []string, p domain.RenewalPeriod) (*domain.RenewalPeriod, error) {
	for _, id := range calonID {
		baru, err := scan(r.db.QueryRow(ctx, `
			INSERT INTO renewal_periods AS p (id, name, start_date, end_date)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (id) DO NOTHING
			RETURNING `+columns, id, p.Name, p.StartDate.Time, p.EndDate.Time))
		if errors.Is(err, httpx.ErrNotFound) {
			continue
		}
		return baru, err
	}
	return nil, httpx.Conflict("periode dengan rentang tanggal itu sudah ada")
}

func (r *Repository) Update(ctx context.Context, p domain.RenewalPeriod) (*domain.RenewalPeriod, error) {
	return scan(r.db.QueryRow(ctx, `
		UPDATE renewal_periods AS p
		SET name = $2, start_date = $3, end_date = $4, updated_at = now()
		WHERE p.id = $1
		RETURNING `+columns, p.ID, p.Name, p.StartDate.Time, p.EndDate.Time))
}

// JumlahPermintaan menghitung permintaan konfirmasi di satu periode, di seluruh
// chapter. Dipakai penjaga hapus, yang tidak boleh ikut lingkup pemanggil.
func (r *Repository) JumlahPermintaan(ctx context.Context, id string) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM renewal_requests WHERE period = $1", id).Scan(&n); err != nil {
		return 0, fmt.Errorf("hitung permintaan periode: %w", err)
	}
	return n, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM renewal_periods WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("hapus periode: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}
