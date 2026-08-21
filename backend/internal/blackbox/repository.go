package blackbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository menyimpan rekaman panggilan integrasi ke Postgres.
//
// Recorder tetap memegang ring buffer di memori; repository ini yang membuat
// riwayatnya bertahan melewati restart. Keduanya dipakai bersama, dan
// pembagiannya disengaja: memori menjawab "apa yang barusan terjadi" tanpa
// menyentuh database, tabel menjawab "waktu itu apa yang terjadi".
type Repository struct {
	pool *pgxpool.Pool

	// retain membatasi jumlah baris yang disimpan. Tanpa batas, tabel ini
	// tumbuh selamanya — setiap penerbitan invoice menambah dua baris, dan
	// tidak ada yang pernah menghapusnya.
	retain int
}

func NewRepository(pool *pgxpool.Pool, retain int) *Repository {
	if retain <= 0 {
		retain = 10_000
	}
	return &Repository{pool: pool, retain: retain}
}

// Insert menyimpan satu rekaman.
//
// Dipanggil dari jalur permintaan, jadi batas waktunya pendek: kegagalan
// menyimpan catatan diagnostik tidak boleh menahan penerbitan invoice. Error
// dikembalikan supaya pemanggil bisa mencatatnya, bukan supaya pemanggil gagal.
func (r *Repository) Insert(ctx context.Context, e Entry) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const q = `
		INSERT INTO integration_calls
		  (occurred_at, integration, direction, method, url,
		   request, response, status, success, duration_ms, error)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	_, err := r.pool.Exec(ctx, q,
		e.Time, e.Integration, e.Direction, e.Method, e.URL,
		nullJSON(e.Request), nullJSON(e.Response),
		e.Status, e.Success, e.DurationMS, nullText(e.Error))
	if err != nil {
		return fmt.Errorf("simpan rekaman integrasi: %w", err)
	}
	return nil
}

// Prune membuang baris terlama sehingga tersisa paling banyak retain baris.
//
// Memakai batas id, bukan OFFSET: id monoton naik, jadi satu perbandingan
// mengalahkan pemindaian yang harus melewati baris yang akan dipertahankan.
func (r *Repository) Prune(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		DELETE FROM integration_calls
		WHERE id <= COALESCE(
		  (SELECT id FROM integration_calls ORDER BY id DESC OFFSET $1 LIMIT 1), 0)`
	tag, err := r.pool.Exec(ctx, q, r.retain)
	if err != nil {
		return 0, fmt.Errorf("pangkas rekaman integrasi: %w", err)
	}
	return tag.RowsAffected(), nil
}

// List mengembalikan rekaman terbaru lebih dulu.
func (r *Repository) List(ctx context.Context, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 200
	}
	const q = `
		SELECT id, occurred_at, integration, direction, method, url,
		       request, response, status, success, duration_ms, coalesce(error, '')
		FROM integration_calls
		ORDER BY occurred_at DESC, id DESC
		LIMIT $1`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("baca rekaman integrasi: %w", err)
	}
	defer rows.Close()

	out := []Entry{}
	for rows.Next() {
		var (
			e         Entry
			id        int64
			req, resp []byte
		)
		if err := rows.Scan(&id, &e.Time, &e.Integration, &e.Direction, &e.Method,
			&e.URL, &req, &resp, &e.Status, &e.Success, &e.DurationMS, &e.Error); err != nil {
			return nil, fmt.Errorf("baca baris rekaman: %w", err)
		}
		e.ID = strconv.FormatInt(id, 10)
		e.Request = json.RawMessage(req)
		e.Response = json.RawMessage(resp)
		out = append(out, e)
	}
	return out, rows.Err()
}

// Clear mengosongkan tabel.
func (r *Repository) Clear(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "TRUNCATE integration_calls RESTART IDENTITY")
	if err != nil {
		return fmt.Errorf("bersihkan rekaman integrasi: %w", err)
	}
	return nil
}

// nullJSON memetakan JSON kosong ke NULL, bukan ke string "null" — kolomnya
// jsonb, dan "null" di dalam jsonb adalah nilai yang sah tetapi menyesatkan.
func nullJSON(m json.RawMessage) any {
	if len(m) == 0 {
		return nil
	}
	return []byte(m)
}

func nullText(s string) any {
	if s == "" {
		return nil
	}
	return s
}
