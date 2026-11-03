package sync

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// Result reports what a run changed.
type Result struct {
	Chapters    int `json:"chapters"`
	Members     int `json:"members"`
	Deactivated int `json:"deactivated"`
	// LinkedChapters and LinkedMembers count rows this run joined to BNI VM by name.
	LinkedChapters int `json:"linkedChapters"`
	LinkedMembers  int `json:"linkedMembers"`
	// NeedsReview names the BNI VM rows that matched stored rows ambiguously. They
	// got a new row rather than a guess, so they may be duplicates.
	NeedsReview []string  `json:"needsReview"`
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
	storedChapters, err := loadStored(ctx, tx, "SELECT id, bni_vm_id, ARRAY[id, name, display_name] FROM chapters", chapterKeys)
	if err != nil {
		return nil, err
	}
	remoteChapters := make([]remoteRow, len(chapters))
	for i, c := range chapters {
		remoteChapters[i] = remoteRow{ID: c.ID, Key: domain.KunciChapter(c.Name), Label: "Chapter " + c.Name}
	}
	chapterMatch := match(remoteChapters, storedChapters)

	localChapter := make(map[string]string, len(chapters))
	for _, c := range chapters {
		id := cmp.Or(chapterMatch.Local[c.ID], c.ID)
		localChapter[c.ID] = id
		const q = `
			INSERT INTO chapters (id, name, display_name, synced_at, bni_vm_id)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (id) DO UPDATE SET
			  name = excluded.name,
			  -- display_name is editable locally, so only fill it when unset;
			  -- a sync must not silently undo someone's rename.
			  display_name = COALESCE(NULLIF(chapters.display_name, ''), excluded.display_name),
			  synced_at = excluded.synced_at,
			  bni_vm_id = excluded.bni_vm_id`
		if _, err := tx.Exec(ctx, q, id, c.Name, c.Name, now, c.ID); err != nil {
			return nil, fmt.Errorf("simpan chapter %s: %w", c.ID, err)
		}
	}

	storedMembers, err := loadStored(ctx, tx, "SELECT id, bni_vm_id, ARRAY[chapter_id, name] FROM members", memberKeys)
	if err != nil {
		return nil, err
	}
	remoteMembers := make([]remoteRow, len(members))
	for i, m := range members {
		remoteMembers[i] = remoteRow{
			ID:    m.ID,
			Key:   memberKey(cmp.Or(localChapter[m.ChapterID], m.ChapterID), m.Name),
			Label: m.Name + " (" + cmp.Or(m.Chapter, m.ChapterID) + ")",
		}
	}
	memberMatch := match(remoteMembers, storedMembers)

	seen := make([]string, 0, len(members))
	for _, m := range members {
		status := m.Status
		if status != "active" && status != "inactive" && status != "pending" {
			status = "active"
		}
		const q = `
			INSERT INTO members (id, chapter_id, name, email, phone, company,
			                     business_field, status, joined_date, renewal_date, synced_at, bni_vm_id)
			VALUES ($1,$2,$3,nullif($4,''),nullif($5,''),nullif($6,''),nullif($7,''),$8,$9,$10,$11,$12)
			ON CONFLICT (id) DO UPDATE SET
			  chapter_id = excluded.chapter_id, name = excluded.name,
			  -- A value BNI VM left empty keeps what is stored.
			  email = coalesce(excluded.email, members.email),
			  phone = coalesce(excluded.phone, members.phone),
			  company = coalesce(excluded.company, members.company),
			  business_field = coalesce(excluded.business_field, members.business_field),
			  joined_date = coalesce(excluded.joined_date, members.joined_date),
			  -- A paid renewal advances this date here before BNI VM knows about
			  -- it, so a sync must not move it back. GREATEST ignores NULL.
			  renewal_date = greatest(excluded.renewal_date, members.renewal_date),
			  -- BNI VM sets status on insert. Afterwards it can only deactivate;
			  -- every other status belongs to PATCH and payments.
			  status = CASE WHEN excluded.status = 'inactive' THEN excluded.status ELSE members.status END,
			  synced_at = excluded.synced_at, bni_vm_id = excluded.bni_vm_id`
		_, err := tx.Exec(ctx, q,
			cmp.Or(memberMatch.Local[m.ID], m.ID), cmp.Or(localChapter[m.ChapterID], m.ChapterID), m.Name,
			m.Email, m.Phone, m.Company, m.BusinessField, status,
			nullableDate(m.JoinedDate), nullableDate(m.RenewalDate), now, m.ID)
		if err != nil {
			return nil, fmt.Errorf("simpan member %s: %w", m.ID, err)
		}
		seen = append(seen, m.ID)
	}

	// A member who disappeared upstream is DEACTIVATED, never deleted. Deleting
	// would fail the foreign key as soon as they have an invoice, and would
	// throw away billing history even when it succeeded.
	//
	// Only rows linked to BNI VM count as disappeared. Members from a BNI Connect
	// report or Tambah Member were never upstream to begin with.
	var deactivated int
	if len(seen) > 0 {
		const q = `
			UPDATE members SET status = 'inactive', synced_at = $2
			WHERE bni_vm_id IS NOT NULL AND bni_vm_id <> ALL($1) AND status <> 'inactive'`
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
		Chapters:       len(chapters),
		Members:        len(members),
		Deactivated:    deactivated,
		LinkedChapters: chapterMatch.Linked,
		LinkedMembers:  memberMatch.Linked,
		NeedsReview:    append(append([]string{}, chapterMatch.Ambiguous...), memberMatch.Ambiguous...),
		SyncedAt:       now,
	}, nil
}

// loadStored reads the rows a snapshot may match, with keys built from the text
// columns the query returns as its third value.
func loadStored(ctx context.Context, tx pgx.Tx, query string, keys func([]string) []string) ([]storedRow, error) {
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("baca baris tersimpan: %w", err)
	}
	defer rows.Close()

	var out []storedRow
	for rows.Next() {
		var s storedRow
		var fields []string
		if err := rows.Scan(&s.ID, &s.BNIVMID, &fields); err != nil {
			return nil, fmt.Errorf("baca baris tersimpan: %w", err)
		}
		s.Keys = keys(fields)
		out = append(out, s)
	}
	return out, rows.Err()
}

// chapterKeys takes id, name, and display_name. A chapter created from a report
// and one renamed locally can each be found under any of the three.
func chapterKeys(fields []string) []string {
	keys := make([]string, 0, len(fields))
	for _, f := range fields {
		if k := domain.KunciChapter(f); k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

// memberKeys takes chapter_id and name.
func memberKeys(fields []string) []string {
	return []string{memberKey(fields[0], fields[1])}
}

// memberKey scopes a name to its chapter: a name only identifies someone inside
// one chapter, which is the same assumption the BNI Connect import makes.
func memberKey(chapterID, name string) string {
	k := domain.KunciNamaMember(name)
	if k == "" {
		return ""
	}
	return chapterID + "\x00" + k
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
