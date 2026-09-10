package auth

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

// --- token reset kata sandi -------------------------------------------------

func (r *Repository) SimpanTokenReset(ctx context.Context, userID, hash string, berlaku time.Time) error {
	// Token lama milik pengguna yang sama dibatalkan lebih dulu.
	//
	// Meminta reset dua kali seharusnya membuat tautan PERTAMA mati — kalau
	// tidak, seseorang yang meminta reset karena curiga akunnya diincar justru
	// meninggalkan tautan lamanya tetap hidup, dan permintaan barunya tidak
	// menutup apa pun.
	if _, err := r.db.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = now()
		 WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return fmt.Errorf("batalkan token reset lama: %w", err)
	}
	if _, err := r.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`, userID, hash, berlaku); err != nil {
		return fmt.Errorf("simpan token reset: %w", err)
	}
	return nil
}

func (r *Repository) AmbilTokenReset(ctx context.Context, hash string, sekarang time.Time) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx,
		`SELECT user_id FROM password_reset_tokens
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2`,
		hash, sekarang).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", httpx.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("ambil token reset: %w", err)
	}
	return userID, nil
}

func (r *Repository) PakaiTokenReset(ctx context.Context, hash, userID, passwordHash string, sekarang time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi reset: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Penandaan token dan penulisan kata sandi HARUS satu transaksi.
	//
	// Terpisah, kegagalan di antaranya meninggalkan salah satu dari dua keadaan
	// yang sama-sama buruk: token terpakai tapi kata sandi belum berubah
	// (pengguna terkunci, tautannya sudah mati), atau kata sandi berubah tapi
	// tokennya masih hidup (tautan lama tetap bisa mengambil alih akun).
	//
	// Kondisi used_at IS NULL diulang di sini, bukan mengandalkan pemeriksaan
	// sebelumnya: dua permintaan yang tiba bersamaan sama-sama lolos
	// pemeriksaan itu, dan hanya klausa ini yang membuat salah satunya kalah.
	ct, err := tx.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = $2
		 WHERE token_hash = $1 AND used_at IS NULL`, hash, sekarang)
	if err != nil {
		return fmt.Errorf("tandai token terpakai: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}

	if _, err := tx.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`,
		userID, passwordHash); err != nil {
		return fmt.Errorf("tulis kata sandi baru: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *Repository) HapusTokenResetLama(ctx context.Context, sebelum time.Time) (int64, error) {
	ct, err := r.db.Exec(ctx,
		`DELETE FROM password_reset_tokens WHERE expires_at < $1`, sebelum)
	if err != nil {
		return 0, fmt.Errorf("bersihkan token reset: %w", err)
	}
	return ct.RowsAffected(), nil
}

// --- OTP login ---------------------------------------------------------------

func (r *Repository) SimpanOTP(ctx context.Context, userID, hash string, berlaku time.Time) error {
	// Kode lama milik pengguna yang sama dibatalkan. Dua kode hidup sekaligus
	// berarti dua peluang menebak untuk satu akun, dan jatah percobaannya
	// terpisah — batas lima berubah jadi sepuluh tanpa ada yang menyadarinya.
	if _, err := r.db.Exec(ctx,
		`UPDATE login_otp_codes SET used_at = now()
		 WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return fmt.Errorf("batalkan OTP lama: %w", err)
	}
	if _, err := r.db.Exec(ctx,
		`INSERT INTO login_otp_codes (user_id, code_hash, expires_at) VALUES ($1,$2,$3)`,
		userID, hash, berlaku); err != nil {
		return fmt.Errorf("simpan OTP: %w", err)
	}
	return nil
}

// PakaiOTP memeriksa kode dan mencatat percobaannya dalam SATU transaksi.
//
// Pemeriksaan dan pencatatan yang terpisah adalah balapan yang bisa dimenangkan:
// sepuluh tebakan yang dikirim bersamaan semuanya membaca attempts yang sama,
// semuanya lolos pemeriksaan batas, dan batas lima tidak pernah berlaku.
// FOR UPDATE membuat mereka mengantre.
func (r *Repository) PakaiOTP(ctx context.Context, userID, hash string, sekarang time.Time, maks int) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("mulai transaksi OTP: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id, simpanan string
	var percobaan int
	err = tx.QueryRow(ctx,
		`SELECT id, code_hash, attempts FROM login_otp_codes
		 WHERE user_id = $1 AND used_at IS NULL AND expires_at > $2
		 ORDER BY created_at DESC LIMIT 1
		 FOR UPDATE`, userID, sekarang).Scan(&id, &simpanan, &percobaan)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("ambil OTP: %w", err)
	}

	if percobaan >= maks {
		// Dihanguskan, bukan dibiarkan menunggu kedaluwarsa: kode yang sudah
		// habis jatahnya tidak boleh bisa dicoba lagi setelah restart.
		if _, err := tx.Exec(ctx, `UPDATE login_otp_codes SET used_at = $2 WHERE id = $1`, id, sekarang); err != nil {
			return 0, fmt.Errorf("hanguskan OTP: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit hangus OTP: %w", err)
		}
		return 0, httpx.ErrNotFound
	}

	if !TokenCocok(simpanan, hash) {
		if _, err := tx.Exec(ctx,
			`UPDATE login_otp_codes SET attempts = attempts + 1 WHERE id = $1`, id); err != nil {
			return 0, fmt.Errorf("catat percobaan OTP: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit percobaan OTP: %w", err)
		}
		return maks - percobaan - 1, httpx.ErrNotFound
	}

	if _, err := tx.Exec(ctx, `UPDATE login_otp_codes SET used_at = $2 WHERE id = $1`, id, sekarang); err != nil {
		return 0, fmt.Errorf("tandai OTP terpakai: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit OTP: %w", err)
	}
	return maks - percobaan, nil
}

func (r *Repository) HapusOTPLama(ctx context.Context, sebelum time.Time) (int64, error) {
	ct, err := r.db.Exec(ctx, `DELETE FROM login_otp_codes WHERE expires_at < $1`, sebelum)
	if err != nil {
		return 0, fmt.Errorf("bersihkan OTP: %w", err)
	}
	return ct.RowsAffected(), nil
}

// OTPAktif membaca sakelar dari app_settings.
//
// BAWAANNYA MATI, dan galat apa pun juga dibaca mati. Pengaturan yang gagal
// terbaca tidak boleh MENGAKTIFKAN lapisan yang bisa mengunci semua orang di
// luar aplikasinya — termasuk admin yang harus mematikannya.
func (r *Repository) OTPAktif(ctx context.Context) bool {
	var v string
	err := r.db.QueryRow(ctx,
		`SELECT value FROM app_settings WHERE key = 'login_otp_enabled'`).Scan(&v)
	if err != nil {
		return false
	}
	return strings.TrimSpace(v) == "true"
}
