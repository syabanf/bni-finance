// Package member implements CRUD over the `members` table, plus the
// renewal-due query the dashboard and reminder flows rely on.
package member

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

// Members are always read with their chapter joined — every list view in the
// UI shows the chapter name, so a second query would be pure overhead.
const columns = `
	m.id, m.chapter_id, m.name, m.email, m.phone, m.company, m.business_field,
	m.status, m.joined_date, m.renewal_date, m.synced_at,
	c.id, c.name, c.display_name, c.area_name, c.city_name, c.synced_at`

const from = `FROM members m LEFT JOIN chapters c ON c.id = m.chapter_id`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface {
	Scan(dest ...any) error
}

// scan reads a member row plus the LEFT JOIN-ed chapter. Chapter columns are
// nullable here even though chapter_id is NOT NULL, so they're scanned into
// pointers and only assembled when present.
func scan(row scannable, extra ...any) (*domain.Member, error) {
	var m domain.Member
	var joined, renewal domain.Date

	var (
		chID, chName, chDisplay *string
		chArea, chCity          *string
		chSynced                *time.Time
	)

	dest := []any{
		&m.ID, &m.ChapterID, &m.Name, &m.Email, &m.Phone, &m.Company, &m.BusinessField,
		&m.Status, &joined, &renewal, &m.SyncedAt,
		&chID, &chName, &chDisplay, &chArea, &chCity, &chSynced,
	}
	dest = append(dest, extra...)

	if err := row.Scan(dest...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("scan member: %w", err)
	}

	if !joined.IsZero() {
		m.JoinedDate = &joined
	}
	if !renewal.IsZero() {
		m.RenewalDate = &renewal
	}
	if chID != nil && chName != nil && chDisplay != nil && chSynced != nil {
		m.Chapter = &domain.Chapter{
			ID:          *chID,
			Name:        *chName,
			DisplayName: *chDisplay,
			AreaName:    chArea,
			CityName:    chCity,
			SyncedAt:    *chSynced,
		}
	}
	return &m, nil
}

func (r *Repository) List(ctx context.Context, f domain.MemberFilter) ([]domain.Member, int, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if f.ChapterID != "" {
		add("m.chapter_id = $%d", f.ChapterID)
	}
	if f.Status != "" {
		add("m.status = $%d", domain.MemberStatus(f.Status))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(
			"(m.name ILIKE $%d OR coalesce(m.email,'') ILIKE $%d OR coalesce(m.company,'') ILIKE $%d)",
			n, n, n))
	}
	if f.RenewalFrom != "" {
		add("m.renewal_date >= $%d", f.RenewalFrom)
	}
	if f.RenewalTo != "" {
		add("m.renewal_date <= $%d", f.RenewalTo)
	}

	clause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+from+" WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung member: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf("SELECT %s %s WHERE %s ORDER BY m.name ASC LIMIT $%d OFFSET $%d",
		columns, from, clause, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ambil member: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Member, 0, f.Limit)
	for rows.Next() {
		m, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *m)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Member, error) {
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+" "+from+" WHERE m.id = $1", id))
}

// RenewalDue lists active members whose membership lapses within the next
// `days` days, soonest first — the input for renewal invoices and reminders.
func (r *Repository) RenewalDue(ctx context.Context, days, limit int) ([]domain.RenewalDueMember, error) {
	query := fmt.Sprintf(`
		SELECT %s, (m.renewal_date - CURRENT_DATE)::int
		%s
		WHERE m.status = 'active'
		  AND m.renewal_date IS NOT NULL
		  AND m.renewal_date >= CURRENT_DATE
		  AND m.renewal_date <= CURRENT_DATE + ($1 || ' days')::interval
		ORDER BY m.renewal_date ASC
		LIMIT $2`, columns, from)

	rows, err := r.db.Query(ctx, query, days, limit)
	if err != nil {
		return nil, fmt.Errorf("ambil member jatuh tempo: %w", err)
	}
	defer rows.Close()

	items := make([]domain.RenewalDueMember, 0, limit)
	for rows.Next() {
		var daysUntil int
		m, err := scan(rows, &daysUntil)
		if err != nil {
			return nil, err
		}
		items = append(items, domain.RenewalDueMember{Member: *m, DaysUntilDue: daysUntil})
	}
	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, in domain.CreateMemberInput) (*domain.Member, error) {
	id := ""
	if in.ID != nil {
		id = *in.ID
	}
	status := domain.MemberActive
	if in.Status != nil {
		status = *in.Status
	}

	const insert = `
		INSERT INTO members (id, chapter_id, name, email, phone, company, business_field,
		                     status, joined_date, renewal_date)
		VALUES (coalesce(nullif($1,''), gen_random_uuid()::text), $2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`

	var newID string
	err := r.db.QueryRow(ctx, insert,
		id, in.ChapterID, in.Name, in.Email, in.Phone, in.Company, in.BusinessField,
		status, in.JoinedDate, in.RenewalDate,
	).Scan(&newID)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "23505"):
			return nil, httpx.Conflict("member dengan id tersebut sudah ada")
		case strings.Contains(err.Error(), "23503"):
			return nil, httpx.BadRequest("chapterId tidak ditemukan")
		}
		return nil, fmt.Errorf("simpan member: %w", err)
	}

	// Re-read so the response carries the joined chapter like every other read.
	return r.GetByID(ctx, newID)
}

func (r *Repository) Update(ctx context.Context, id string, in domain.UpdateMemberInput) (*domain.Member, error) {
	sets := []string{"synced_at = now()"}
	args := []any{}
	set := func(col string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if in.ChapterID != nil {
		set("chapter_id", *in.ChapterID)
	}
	if in.Name != nil {
		set("name", *in.Name)
	}
	if in.Email != nil {
		set("email", *in.Email)
	}
	if in.Phone != nil {
		set("phone", *in.Phone)
	}
	if in.Company != nil {
		set("company", *in.Company)
	}
	if in.BusinessField != nil {
		set("business_field", *in.BusinessField)
	}
	if in.Status != nil {
		set("status", *in.Status)
	}
	if in.JoinedDate != nil {
		set("joined_date", *in.JoinedDate)
	}
	if in.RenewalDate != nil {
		set("renewal_date", *in.RenewalDate)
	}

	args = append(args, id)
	q := fmt.Sprintf("UPDATE members SET %s WHERE id = $%d RETURNING id",
		strings.Join(sets, ", "), len(args))

	var updatedID string
	if err := r.db.QueryRow(ctx, q, args...).Scan(&updatedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		if strings.Contains(err.Error(), "23503") {
			return nil, httpx.BadRequest("chapterId tidak ditemukan")
		}
		return nil, fmt.Errorf("ubah member: %w", err)
	}
	return r.GetByID(ctx, updatedID)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM members WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("hapus member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

// CountInvoices reports how many invoices a member has — a delete would
// otherwise fail on the foreign key with an opaque message.
func (r *Repository) CountInvoices(ctx context.Context, id string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM invoices WHERE member_id = $1", id).Scan(&n)
	return n, err
}
