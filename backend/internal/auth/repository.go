package auth

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

const columns = `id, email, password_hash, name, role, created_at, updated_at`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface {
	Scan(dest ...any) error
}

func scan(row scannable) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}

// GetByEmail matches case-insensitively, backed by the lower(email) index.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scan(r.db.QueryRow(ctx,
		"SELECT "+columns+" FROM users WHERE lower(email) = $1", domain.NormalizeEmail(email)))
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+" FROM users WHERE id = $1", id))
}

func (r *Repository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.Query(ctx, "SELECT "+columns+" FROM users ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("ambil user: %w", err)
	}
	defer rows.Close()

	items := make([]domain.User, 0, 8)
	for rows.Next() {
		u, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *u)
	}
	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, email, passwordHash, name string, role domain.UserRole) (*domain.User, error) {
	const q = `
		INSERT INTO users (email, password_hash, name, role)
		VALUES ($1,$2,$3,$4)
		RETURNING ` + columns

	u, err := scan(r.db.QueryRow(ctx, q, domain.NormalizeEmail(email), passwordHash, name, role))
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			return nil, httpx.Conflict("email tersebut sudah terdaftar")
		}
		return nil, err
	}
	return u, nil
}

func (r *Repository) UpdateName(ctx context.Context, id, name string) (*domain.User, error) {
	return scan(r.db.QueryRow(ctx,
		"UPDATE users SET name = $2, updated_at = now() WHERE id = $1 RETURNING "+columns, id, name))
}

func (r *Repository) UpdateRole(ctx context.Context, id string, role domain.UserRole) (*domain.User, error) {
	return scan(r.db.QueryRow(ctx,
		"UPDATE users SET role = $2, updated_at = now() WHERE id = $1 RETURNING "+columns, id, role))
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, id, hash string) error {
	tag, err := r.db.Exec(ctx,
		"UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1", id, hash)
	if err != nil {
		return fmt.Errorf("ubah kata sandi: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("hapus user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

// CountAdmins backs the guard that stops the last administrator from being
// deleted or demoted, which would lock everyone out of the settings pages.
func (r *Repository) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&n)
	return n, err
}
