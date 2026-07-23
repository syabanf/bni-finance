package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// Result reports what a run changed.
type Result struct {
	Chapters    int       `json:"chapters"`
	Members     int       `json:"members"`
	Deactivated int       `json:"deactivated"`
	SyncedAt    time.Time `json:"syncedAt"`
}

// Apply writes the whole snapshot in ONE transaction. A partial write could
// leave members pointing at a chapter that was never inserted, which the
// foreign key would reject halfway through — better all or nothing.
func (r *Repository) Apply(ctx context.Context, members []RemoteMember, now time.Time) (*Result, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	// Chapters first: members reference them.
	chapters := deriveChapters(members)
	for _, c := range chapters {
		const q = `
			INSERT INTO chapters (id, name, display_name, synced_at)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (id) DO UPDATE SET
			  name = excluded.name,
			  -- display_name is editable locally, so only fill it when unset;
			  -- a sync must not silently undo someone's rename.
			  display_name = COALESCE(NULLIF(chapters.display_name, ''), excluded.display_name),
			  synced_at = excluded.synced_at`
		if _, err := tx.Exec(ctx, q, c.ID, c.Name, c.Name, now); err != nil {
			return nil, fmt.Errorf("simpan chapter %s: %w", c.ID, err)
		}
	}

	seen := make([]string, 0, len(members))
	for _, m := range members {
		status := m.Status
		if status != "active" && status != "inactive" && status != "pending" {
			status = "active"
		}
		const q = `
			INSERT INTO members (id, chapter_id, name, email, phone, company,
			                     business_field, status, joined_date, renewal_date, synced_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			ON CONFLICT (id) DO UPDATE SET
			  chapter_id = excluded.chapter_id, name = excluded.name,
			  email = excluded.email, phone = excluded.phone, company = excluded.company,
			  business_field = excluded.business_field, status = excluded.status,
			  joined_date = excluded.joined_date, renewal_date = excluded.renewal_date,
			  synced_at = excluded.synced_at`
		_, err := tx.Exec(ctx, q,
			m.ID, m.ChapterID, m.Name, m.Email, m.Phone, m.Company,
			m.BusinessField, status, nullableDate(m.JoinedDate), nullableDate(m.RenewalDate), now)
		if err != nil {
			return nil, fmt.Errorf("simpan member %s: %w", m.ID, err)
		}
		seen = append(seen, m.ID)
	}

	// A member who disappeared upstream is DEACTIVATED, never deleted. Deleting
	// would fail the foreign key as soon as they have an invoice, and would
	// throw away billing history even when it succeeded.
	var deactivated int
	if len(seen) > 0 {
		const q = `
			UPDATE members SET status = 'inactive', synced_at = $2
			WHERE id <> ALL($1) AND status <> 'inactive'`
		tag, err := tx.Exec(ctx, q, seen, now)
		if err != nil {
			return nil, fmt.Errorf("nonaktifkan member yang hilang: %w", err)
		}
		deactivated = int(tag.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit sinkronisasi: %w", err)
	}

	return &Result{
		Chapters:    len(chapters),
		Members:     len(members),
		Deactivated: deactivated,
		SyncedAt:    now,
	}, nil
}

type chapterRow struct{ ID, Name string }

// deriveChapters pulls the distinct chapters out of the member rows — BNI VM
// has no chapters endpoint of its own.
func deriveChapters(members []RemoteMember) []chapterRow {
	seen := make(map[string]bool, 16)
	out := make([]chapterRow, 0, 16)
	for _, m := range members {
		if m.ChapterID == "" || seen[m.ChapterID] {
			continue
		}
		seen[m.ChapterID] = true
		name := m.Chapter
		if name == "" {
			name = m.ChapterID
		}
		out = append(out, chapterRow{ID: m.ChapterID, Name: name})
	}
	return out
}

// nullableDate turns an empty or absent date into SQL NULL rather than the
// zero date, which would read as 0001-01-01 in every report.
func nullableDate(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

// Setting reads one app_settings value; used to find the BNI VM token.
func (r *Repository) Setting(ctx context.Context, key string) (string, error) {
	var v string
	err := r.db.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = $1", key).Scan(&v)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("ambil setting %s: %w", key, err)
	}
	return v, nil
}
