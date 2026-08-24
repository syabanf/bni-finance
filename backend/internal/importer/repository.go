package importer

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) ChapterIDs(ctx context.Context) (map[string]bool, error) {
	rows, err := r.db.Query(ctx, "SELECT id FROM chapters")
	if err != nil {
		return nil, fmt.Errorf("baca id chapter: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan id chapter: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (r *Repository) ChapterRows(ctx context.Context) (map[string]ChapterRow, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, name, coalesce(display_name,''), coalesce(area_name,''), coalesce(city_name,'') FROM chapters")
	if err != nil {
		return nil, fmt.Errorf("baca chapter: %w", err)
	}
	defer rows.Close()
	out := map[string]ChapterRow{}
	for rows.Next() {
		var c ChapterRow
		if err := rows.Scan(&c.ID, &c.Name, &c.DisplayName, &c.AreaName, &c.CityName); err != nil {
			return nil, fmt.Errorf("scan chapter: %w", err)
		}
		out[c.ID] = c
	}
	return out, rows.Err()
}

func (r *Repository) MemberRows(ctx context.Context) (map[string]MemberRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, chapter_id, name, coalesce(email,''), coalesce(phone,''),
		       coalesce(company,''), coalesce(business_field,''), status::text
		FROM members`)
	if err != nil {
		return nil, fmt.Errorf("baca member: %w", err)
	}
	defer rows.Close()
	out := map[string]MemberRow{}
	for rows.Next() {
		var m MemberRow
		if err := rows.Scan(&m.ID, &m.ChapterID, &m.Name, &m.Email, &m.Phone,
			&m.Company, &m.BusinessField, &m.Status); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		out[m.ID] = m
	}
	return out, rows.Err()
}

// UpsertChapters menulis seluruh baris dalam SATU transaksi.
//
// Semua atau tidak sama sekali. Impor yang berhenti di tengah meninggalkan
// separuh berkas tertulis dan separuh tidak — dan karena orang sudah membaca
// pratinjau yang menjanjikan seluruhnya, tidak ada yang tahu bagian mana yang
// masuk tanpa membandingkan baris per baris.
func (r *Repository) UpsertChapters(ctx context.Context, rows []ChapterRow) error {
	return r.dalamTransaksi(ctx, func(tx pgx.Tx) error {
		for _, c := range rows {
			if _, err := tx.Exec(ctx, `
				INSERT INTO chapters (id, name, display_name, area_name, city_name)
				VALUES ($1,$2,$3,$4,$5)
				ON CONFLICT (id) DO UPDATE SET
					name         = excluded.name,
					-- coalesce nullif: kolom yang KOSONG di berkas tidak menimpa
					-- nilai tersimpan. Berkas yang hanya memuat sebagian kolom
					-- adalah hal biasa, dan menganggap yang hilang sebagai
					-- "kosongkan" akan menghapus data dalam satu impor yang
					-- tampak wajar.
					display_name = coalesce(nullif(excluded.display_name,''), chapters.display_name),
					area_name    = coalesce(nullif(excluded.area_name,''),    chapters.area_name),
					city_name    = coalesce(nullif(excluded.city_name,''),    chapters.city_name)`,
				c.ID, c.Name, c.DisplayName, c.AreaName, c.CityName); err != nil {
				return fmt.Errorf("tulis chapter %s: %w", c.ID, err)
			}
		}
		return nil
	})
}

func (r *Repository) UpsertMembers(ctx context.Context, rows []MemberRow) error {
	return r.dalamTransaksi(ctx, func(tx pgx.Tx) error {
		for _, m := range rows {
			if _, err := tx.Exec(ctx, `
				INSERT INTO members (id, chapter_id, name, email, phone, company, business_field, status)
				VALUES ($1,$2,$3,nullif($4,''),nullif($5,''),nullif($6,''),nullif($7,''),$8::member_status)
				ON CONFLICT (id) DO UPDATE SET
					chapter_id     = excluded.chapter_id,
					name           = excluded.name,
					email          = coalesce(excluded.email,          members.email),
					phone          = coalesce(excluded.phone,          members.phone),
					company        = coalesce(excluded.company,        members.company),
					business_field = coalesce(excluded.business_field, members.business_field),
					status         = excluded.status`,
				m.ID, m.ChapterID, m.Name, m.Email, m.Phone, m.Company, m.BusinessField, m.Status); err != nil {
				return fmt.Errorf("tulis member %s: %w", m.ID, err)
			}
		}
		return nil
	})
}

func (r *Repository) dalamTransaksi(ctx context.Context, f func(pgx.Tx) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := f(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("simpan impor: %w", err)
	}
	return nil
}
