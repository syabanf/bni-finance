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

const columns = `id, email, password_hash, name, role, chapter_id, created_at, updated_at`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface {
	Scan(dest ...any) error
}

func scan(row scannable) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.ChapterID, &u.CreatedAt, &u.UpdatedAt)
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

func (r *Repository) Create(ctx context.Context, email, passwordHash, name string, role domain.UserRole, chapterID *string) (*domain.User, error) {
	const q = `
		INSERT INTO users (email, password_hash, name, role, chapter_id)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING ` + columns

	u, err := scan(r.db.QueryRow(ctx, q, domain.NormalizeEmail(email), passwordHash, name, role, chapterID))
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "23505"):
			return nil, httpx.Conflict("email tersebut sudah terdaftar")
		case strings.Contains(err.Error(), "23503"):
			return nil, httpx.BadRequest("chapterId tidak ditemukan")
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

// lockAdmins mengunci SELURUH baris admin dan mengembalikan jumlahnya.
//
// Menghitung admin lalu menurunkan peran lewat dua pernyataan terpisah adalah
// balapan: dua penurunan yang berjalan bersamaan sama-sama membaca "masih ada
// 2", dua-duanya lolos, dan sistem berakhir tanpa admin sama sekali — keadaan
// yang tidak bisa dipulihkan lewat API, karena tidak ada lagi yang berwenang
// mengangkat admin baru. Terukur: 3 dari 3 percobaan menyisakan nol admin.
//
// FOR UPDATE membuat transaksi kedua menunggu sampai yang pertama selesai,
// sehingga ia menghitung ulang setelah perubahan pertama terlihat.
func lockAdmins(ctx context.Context, tx pgx.Tx) (int, error) {
	rows, err := tx.Query(ctx, "SELECT id FROM users WHERE role = 'admin' FOR UPDATE")
	if err != nil {
		return 0, fmt.Errorf("kunci daftar admin: %w", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
	}
	return n, rows.Err()
}

// ErrLastAdmin ditolak oleh service menjadi 409. Sentinel, bukan httpx.Error,
// supaya repository tidak perlu tahu soal HTTP.
var ErrLastAdmin = errors.New("admin terakhir tidak boleh diturunkan atau dihapus")

// UpdateRoleGuarded menurunkan peran dengan penjaga admin terakhir DI DALAM
// transaksi yang sama, sehingga pemeriksaan dan penulisannya tak terpisahkan.
func (r *Repository) UpdateRoleGuarded(ctx context.Context, id string, role domain.UserRole) (*domain.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	current, err := scan(tx.QueryRow(ctx, "SELECT "+columns+" FROM users WHERE id = $1", id))
	if err != nil {
		return nil, err
	}
	if current.Role == domain.RoleAdmin && role != domain.RoleAdmin {
		n, err := lockAdmins(ctx, tx)
		if err != nil {
			return nil, err
		}
		if n <= 1 {
			return nil, ErrLastAdmin
		}
	}

	user, err := scan(tx.QueryRow(ctx,
		"UPDATE users SET role = $2, updated_at = now() WHERE id = $1 RETURNING "+columns, id, role))
	if err != nil {
		return nil, err
	}
	return user, tx.Commit(ctx)
}

// DeleteGuarded menghapus pengguna dengan penjaga yang sama, dalam satu
// transaksi.
func (r *Repository) DeleteGuarded(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	current, err := scan(tx.QueryRow(ctx, "SELECT "+columns+" FROM users WHERE id = $1", id))
	if err != nil {
		return err
	}
	if current.Role == domain.RoleAdmin {
		n, err := lockAdmins(ctx, tx)
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastAdmin
		}
	}

	tag, err := tx.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("hapus user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return tx.Commit(ctx)
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
