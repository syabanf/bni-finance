package renewal

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
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

const columns = `
	r.id, r.member_id, r.chapter_id, r.period,
	r.requested_by, r.requested_at, r.assigned_mc,
	r.answer, r.answered_by, r.answered_at, r.note,
	r.created_at, r.updated_at,
	m.name, c.display_name, m.renewal_date`

const from = `
	FROM renewal_requests r
	JOIN members m  ON m.id = r.member_id
	JOIN chapters c ON c.id = r.chapter_id`

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface{ Scan(dest ...any) error }

func scan(row scannable) (*domain.RenewalRequest, error) {
	var r domain.RenewalRequest
	// renewal_date boleh null (member berstatus pending belum punya tanggal
	// perpanjangan), jadi dibaca sebagai pointer lalu dibungkus.
	var renewalDate *time.Time
	err := row.Scan(
		&r.ID, &r.MemberID, &r.ChapterID, &r.Period,
		&r.RequestedBy, &r.RequestedAt, &r.AssignedMC,
		&r.Answer, &r.AnsweredBy, &r.AnsweredAt, &r.Note,
		&r.CreatedAt, &r.UpdatedAt,
		&r.MemberName, &r.ChapterName, &renewalDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("scan permintaan renewal: %w", err)
	}
	if renewalDate != nil {
		d := domain.NewDate(*renewalDate)
		r.RenewalDate = &d
	}
	return &r, nil
}

// List mengembalikan permintaan, dibatasi chapter pemanggil.
func (r *Repository) List(ctx context.Context, f domain.RenewalFilter) ([]domain.RenewalRequest, int, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if f.ChapterID != "" {
		add("r.chapter_id = $%d", f.ChapterID)
	}
	// Batas chapter pemanggil, di atas filter apa pun yang ia kirim.
	if klausa, arg, pakai := scope.Chapter(ctx).SQL("r.chapter_id", len(args)+1); klausa != "" {
		if pakai {
			args = append(args, arg)
		}
		where = append(where, klausa)
	}
	if f.Answer != "" {
		add("r.answer = $%d", domain.RenewalAnswer(f.Answer))
	}
	if f.Period != "" {
		add("r.period = $%d", f.Period)
	}

	clause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+from+" WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung permintaan renewal: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	q := fmt.Sprintf("SELECT %s %s WHERE %s ORDER BY r.answer = 'pending' DESC, m.renewal_date NULLS LAST, m.name LIMIT $%d OFFSET $%d",
		columns, from, clause, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ambil permintaan renewal: %w", err)
	}
	defer rows.Close()

	out := make([]domain.RenewalRequest, 0, f.Limit)
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *item)
	}
	return out, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.RenewalRequest, error) {
	lim := scope.Chapter(ctx)
	switch {
	case lim.Buntu:
		return nil, httpx.ErrNotFound
	case lim.Terbatas:
		return scan(r.db.QueryRow(ctx,
			"SELECT "+columns+from+" WHERE r.id = $1 AND r.chapter_id = $2", id, lim.ChapterID))
	}
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+from+" WHERE r.id = $1", id))
}

// Create membuat permintaan untuk sekumpulan member, dalam SATU transaksi.
//
// Mengembalikan jumlah yang benar-benar dibuat dan jumlah yang dilewati karena
// sudah ada. Membedakan keduanya penting: ST yang menekan tombolnya dua kali
// harus melihat "0 baru, 12 sudah ada", bukan "12 dibuat" yang membuatnya
// mengira permintaan pertamanya hilang.
func (r *Repository) Create(ctx context.Context, memberIDs []string, period string,
	requestedBy string, assignedMC *string) (dibuat, dilewati int, err error) {

	lim := scope.Chapter(ctx)
	if lim.Buntu {
		return 0, 0, httpx.Forbidden("tidak ada lingkup chapter pada permintaan ini")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, id := range memberIDs {
		// chapter_id diambil dari MEMBERNYA, bukan dari klien. Klien yang
		// menentukan chapter berarti ST bisa membuat permintaan atas nama
		// chapter lain hanya dengan mengirim id yang berbeda.
		var chapterID string
		switch err := tx.QueryRow(ctx,
			"SELECT chapter_id FROM members WHERE id = $1", id).Scan(&chapterID); {
		case errors.Is(err, pgx.ErrNoRows):
			return 0, 0, httpx.BadRequest(fmt.Sprintf("member %q tidak ditemukan", id))
		case err != nil:
			return 0, 0, fmt.Errorf("baca chapter member: %w", err)
		}
		if lim.Terbatas && chapterID != lim.ChapterID {
			return 0, 0, httpx.Forbidden(fmt.Sprintf(
				"member %q ada di chapter lain (%s)", id, chapterID))
		}

		// ON CONFLICT DO NOTHING mengandalkan indeks unik (member_id, period).
		// Tanpa itu, ST yang menekan tombolnya dua kali menghasilkan dua
		// permintaan untuk orang yang sama, dan MC melihat pekerjaan ganda yang
		// tidak bisa ia bedakan.
		tag, err := tx.Exec(ctx, `
			INSERT INTO renewal_requests (member_id, chapter_id, period, requested_by, assigned_mc)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (member_id, period) DO NOTHING`,
			id, chapterID, period, requestedBy, assignedMC)
		if err != nil {
			return 0, 0, fmt.Errorf("simpan permintaan renewal: %w", err)
		}
		if tag.RowsAffected() == 1 {
			dibuat++
		} else {
			dilewati++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("simpan permintaan renewal: %w", err)
	}
	return dibuat, dilewati, nil
}

// Answer mencatat jawaban MC.
func (r *Repository) Answer(ctx context.Context, id string, in domain.AnswerRenewalInput,
	answeredBy string) (*domain.RenewalRequest, error) {

	lim := scope.Chapter(ctx)
	if lim.Buntu {
		return nil, httpx.ErrNotFound
	}

	// Batas chapter masuk ke klausa WHERE pada UPDATE-nya, bukan diperiksa
	// lebih dulu lewat pembacaan terpisah. Membaca lalu menulis adalah dua
	// langkah, dan di antara keduanya barisnya bisa berubah.
	klausa, args := "id = $4", []any{in.Answer, in.Note, answeredBy, id}
	if lim.Terbatas {
		klausa = "id = $4 AND chapter_id = $5"
		args = append(args, lim.ChapterID)
	}

	var kena bool
	if err := r.db.QueryRow(ctx, `
		UPDATE renewal_requests
		SET answer = $1, note = $2, answered_by = $3, answered_at = now(), updated_at = now()
		WHERE `+klausa+`
		RETURNING true`, args...).Scan(&kena); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("simpan jawaban renewal: %w", err)
	}
	return r.GetByID(ctx, id)
}
