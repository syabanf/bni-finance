// Package chapter implements CRUD over the `chapters` table.
package chapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

const columns = `id, name, display_name, area_name, city_name, synced_at`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface {
	Scan(dest ...any) error
}

func scan(row scannable) (*domain.Chapter, error) {
	var c domain.Chapter
	err := row.Scan(&c.ID, &c.Name, &c.DisplayName, &c.AreaName, &c.CityName, &c.SyncedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("scan chapter: %w", err)
	}
	return &c, nil
}

func (r *Repository) List(ctx context.Context, f domain.ChapterFilter) ([]domain.Chapter, int, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	// Pengguna berlingkup hanya melihat chapternya sendiri.
	//
	// Tanpa ini seorang ST mendapat daftar SELURUH chapter — nama dan
	// keberadaannya — meski invoice dan membernya sudah dibatasi dengan benar.
	// Daftar itu juga yang mengisi dropdown penyaring chapter, sehingga
	// tampak seolah ia boleh menelusuri chapter lain.
	if lim := scope.Chapter(ctx); lim.Buntu {
		where = append(where, "1=0")
	} else if lim.Terbatas {
		add("id = $%d", lim.ChapterID)
	}

	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		where = append(where, fmt.Sprintf(
			"(name ILIKE $%d OR display_name ILIKE $%d OR coalesce(city_name,'') ILIKE $%d)",
			len(args), len(args), len(args)))
	}
	if f.CityName != "" {
		add("city_name = $%d", f.CityName)
	}
	if f.AreaName != "" {
		add("area_name = $%d", f.AreaName)
	}

	clause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM chapters WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung chapter: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(
		"SELECT %s FROM chapters WHERE %s ORDER BY display_name ASC LIMIT $%d OFFSET $%d",
		columns, clause, len(args)-1, len(args),
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ambil chapter: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Chapter, 0, f.Limit)
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *c)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Chapter, error) {
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+" FROM chapters WHERE id = $1", id))
}

// Create inserts a chapter. An empty id is replaced by a generated uuid so
// chapters that don't come from BNI VM still get a stable key.
func (r *Repository) Create(ctx context.Context, in domain.CreateChapterInput) (*domain.Chapter, error) {
	id := ""
	if in.ID != nil {
		id = *in.ID
	}
	display := in.Name
	if in.DisplayName != nil && *in.DisplayName != "" {
		display = *in.DisplayName
	}

	const q = `
		INSERT INTO chapters (id, name, display_name, area_name, city_name)
		VALUES (coalesce(nullif($1,''), gen_random_uuid()::text), $2, $3, $4, $5)
		RETURNING ` + columns

	c, err := scan(r.db.QueryRow(ctx, q, id, in.Name, display, in.AreaName, in.CityName))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, httpx.Conflict("chapter dengan id tersebut sudah ada")
		}
		return nil, err
	}
	return c, nil
}

func (r *Repository) Update(ctx context.Context, id string, in domain.UpdateChapterInput) (*domain.Chapter, error) {
	sets := []string{"synced_at = now()"}
	args := []any{}
	set := func(col string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if in.Name != nil {
		set("name", *in.Name)
	}
	if in.DisplayName != nil {
		set("display_name", *in.DisplayName)
	}
	if in.AreaName != nil {
		set("area_name", *in.AreaName)
	}
	if in.CityName != nil {
		set("city_name", *in.CityName)
	}

	args = append(args, id)
	q := fmt.Sprintf("UPDATE chapters SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), columns)

	return scan(r.db.QueryRow(ctx, q, args...))
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM chapters WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("hapus chapter: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

// CountDependents reports how many members and invoices point at a chapter, so
// a delete that would break foreign keys can be refused with a clear message.
func (r *Repository) CountDependents(ctx context.Context, id string) (members, invoices int, err error) {
	const q = `
		SELECT (SELECT COUNT(*) FROM members  WHERE chapter_id = $1),
		       (SELECT COUNT(*) FROM invoices WHERE chapter_id = $1)`
	err = r.db.QueryRow(ctx, q, id).Scan(&members, &invoices)
	return
}

func isUniqueViolation(err error) bool {
	// 23505 = unique_violation. Matching on the string keeps pgconn out of the
	// import list for one check.
	return strings.Contains(err.Error(), "23505")
}

// Counts mengembalikan jumlah member dan nominal tunggakan tiap chapter.
//
// SUBQUERY, BUKAN JOIN, dan itu bukan soal selera. Meng-JOIN members dan
// invoices sekaligus menghasilkan hasil kali kartesian di dalam tiap chapter:
// 10 member dan 5 invoice menjadi 50 baris, sehingga cacah membernya terkali
// lima dan nominal tunggakannya terkali sepuluh. Keduanya tetap terlihat
// seperti angka yang wajar.
func (r *Repository) Counts(ctx context.Context) ([]domain.ChapterCounts, error) {
	where := "1=1"
	args := []any{}
	if lim := scope.Chapter(ctx); lim.Buntu {
		where = "1=0"
	} else if lim.Terbatas {
		where = "c.id = $1"
		args = append(args, lim.ChapterID)
	}

	rows, err := r.db.Query(ctx, `
		SELECT c.id,
		  (SELECT count(*) FROM members m WHERE m.chapter_id = c.id),
		  coalesce((SELECT sum(i.amount) FROM invoices i
		             WHERE i.chapter_id = c.id AND i.status IN ('sent','overdue')), 0)
		FROM chapters c
		WHERE `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("hitung ringkasan chapter: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ChapterCounts, 0, 16)
	for rows.Next() {
		var c domain.ChapterCounts
		if err := rows.Scan(&c.ChapterID, &c.MemberCount, &c.Outstanding); err != nil {
			return nil, fmt.Errorf("scan ringkasan chapter: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}
