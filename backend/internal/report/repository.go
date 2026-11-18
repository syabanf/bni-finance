// Package report menghitung halaman Laporan di basis data.
//
// Sebelumnya browser menarik seluruh invoice dan pembayaran lalu menjumlahkan
// sendiri. Angkanya benar, tetapi memuatnya berarti satu permintaan per 200
// baris. Di sini jumlahnya dihitung Postgres dari dua view, v_laporan_invoice
// dan v_laporan_pembayaran, yang juga menjadi satu-satunya tempat aturan
// "invoice mana yang terhitung ditagih" ditulis.
package report

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// saringan merakit dua WHERE, untuk invoice dan untuk pembayaran, dari filter
// yang sama, beserta argumen bernamanya.
//
// Tanggal disaring lewat kolom timestamp aslinya (created_at, paid_at) dengan
// batas yang di-cast dari date. Cast itu memakai zona waktu sesi, jadi hasilnya
// sama dengan membandingkan tanggalnya, dan indeks kolomnya tetap terpakai.
//
// Lingkup chapter pemanggil berlaku pada keduanya, dan pada daftar chapter.
func saringan(ctx context.Context, f domain.ReportFilter) (invoice, bayar, chapter string, args pgx.NamedArgs) {
	args = pgx.NamedArgs{}
	inv := []string{"TRUE"}
	pay := []string{"TRUE"}
	ch := []string{"TRUE"}

	if lim := scope.Chapter(ctx); lim.Buntu {
		inv, pay, ch = []string{"FALSE"}, []string{"FALSE"}, []string{"FALSE"}
	} else if lim.Terbatas {
		args["chapter"] = lim.ChapterID
		inv = append(inv, "chapter_id = @chapter")
		pay = append(pay, "chapter_id = @chapter")
		ch = append(ch, "c.id = @chapter")
	}
	if f.From != "" {
		args["dari"] = f.From
		inv = append(inv, "created_at >= @dari::date")
		pay = append(pay, "paid_at >= @dari::date")
	}
	if f.To != "" {
		args["sampai"] = f.To
		inv = append(inv, "created_at < (@sampai::date + 1)")
		pay = append(pay, "paid_at < (@sampai::date + 1)")
	}
	if f.Type != "" {
		args["tipe"] = f.Type
		inv = append(inv, "type = @tipe::invoice_type")
		pay = append(pay, "type = @tipe::invoice_type")
	}
	return strings.Join(inv, " AND "), strings.Join(pay, " AND "), strings.Join(ch, " AND "), args
}

// Summary menghitung seluruh angka laporan selain tabel detail.
func (r *Repository) Summary(ctx context.Context, f domain.ReportFilter) (*domain.ReportSummary, error) {
	inv, pay, ch, args := saringan(ctx, f)
	out := &domain.ReportSummary{
		Chapters: []domain.ReportChapterRow{},
		Types:    map[domain.InvoiceType]int{domain.TypeRegistration: 0, domain.TypeRenewal: 0},
		Monthly:  []domain.ReportMonthly{},
		Methods:  []domain.ReportMethod{},
	}

	// Per chapter. Berangkat dari tabel chapters supaya chapter tanpa invoice
	// tetap tampil. Invoice dan pembayaran diagregasi di subquery masing-masing
	// lebih dulu: meng-JOIN keduanya langsung mengalikan barisnya.
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.display_name,
		       coalesce(i.jumlah, 0), coalesce(i.ditagih, 0), coalesce(i.renewal, 0),
		       coalesce(i.pendaftaran, 0), coalesce(i.outstanding, 0),
		       coalesce(i.n_renewal, 0), coalesce(i.n_pendaftaran, 0),
		       coalesce(p.diterima, 0)
		FROM chapters c
		LEFT JOIN (
			SELECT chapter_id,
			       count(*)                                                  AS jumlah,
			       sum(amount)                                               AS ditagih,
			       sum(amount) FILTER (WHERE type = 'renewal')               AS renewal,
			       sum(amount) FILTER (WHERE type <> 'renewal')              AS pendaftaran,
			       sum(amount) FILTER (WHERE status IN ('sent', 'overdue'))  AS outstanding,
			       count(*) FILTER (WHERE type = 'renewal')                  AS n_renewal,
			       count(*) FILTER (WHERE type = 'registration')             AS n_pendaftaran
			FROM v_laporan_invoice WHERE `+inv+`
			GROUP BY chapter_id) i ON i.chapter_id = c.id
		LEFT JOIN (
			SELECT chapter_id, sum(amount) AS diterima
			FROM v_laporan_pembayaran WHERE `+pay+`
			GROUP BY chapter_id) p ON p.chapter_id = c.id
		WHERE `+ch, args)
	if err != nil {
		return nil, fmt.Errorf("laporan per chapter: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c domain.ReportChapterRow
		var nRenewal, nPendaftaran int
		if err := rows.Scan(&c.ID, &c.Name, &c.Count, &c.Ditagih, &c.Renewal, &c.Pendaftaran,
			&c.Outstanding, &nRenewal, &nPendaftaran, &c.Diterima); err != nil {
			return nil, fmt.Errorf("scan laporan chapter: %w", err)
		}
		c.Rate = domain.CollectionRate(c.Diterima, c.Ditagih)
		out.Chapters = append(out.Chapters, c)

		// Total dijumlahkan dari baris-baris ini, bukan dari query terpisah,
		// supaya kartu ringkasan tidak pernah berbeda dari tabel di bawahnya.
		out.Count += c.Count
		out.Ditagih += c.Ditagih
		out.Diterima += c.Diterima
		out.Outstanding += c.Outstanding
		out.Types[domain.TypeRenewal] += nRenewal
		out.Types[domain.TypeRegistration] += nPendaftaran
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out.Rate = domain.CollectionRate(out.Diterima, out.Ditagih)

	// Tren bulanan: terbit menurut created_at, diterima menurut paid_at. Dua
	// belas bulan TERAKHIR yang punya data, lalu dibalik jadi urut naik.
	bulan, err := r.db.Query(ctx, `
		SELECT bulan, coalesce(i.issued, 0), coalesce(p.paid, 0)
		FROM (SELECT to_char(created_at, 'YYYY-MM') AS bulan, sum(amount) AS issued
		      FROM v_laporan_invoice WHERE `+inv+` GROUP BY 1) i
		FULL JOIN (SELECT to_char(paid_at, 'YYYY-MM') AS bulan, sum(amount) AS paid
		           FROM v_laporan_pembayaran WHERE `+pay+` GROUP BY 1) p USING (bulan)
		ORDER BY bulan DESC
		LIMIT 12`, args)
	if err != nil {
		return nil, fmt.Errorf("laporan bulanan: %w", err)
	}
	defer bulan.Close()
	for bulan.Next() {
		var m domain.ReportMonthly
		if err := bulan.Scan(&m.Month, &m.Issued, &m.Paid); err != nil {
			return nil, fmt.Errorf("scan laporan bulanan: %w", err)
		}
		out.Monthly = append([]domain.ReportMonthly{m}, out.Monthly...)
	}
	if err := bulan.Err(); err != nil {
		return nil, err
	}

	metode, err := r.db.Query(ctx, `
		SELECT metode, count(*), sum(amount)
		FROM v_laporan_pembayaran WHERE `+pay+`
		GROUP BY metode
		ORDER BY sum(amount) DESC, metode`, args)
	if err != nil {
		return nil, fmt.Errorf("laporan metode: %w", err)
	}
	defer metode.Close()
	for metode.Next() {
		var m domain.ReportMethod
		if err := metode.Scan(&m.Method, &m.Count, &m.Amount); err != nil {
			return nil, fmt.Errorf("scan laporan metode: %w", err)
		}
		out.Methods = append(out.Methods, m)
	}
	if err := metode.Err(); err != nil {
		return nil, err
	}

	// Subquery, bukan JOIN: payments dan view sama-sama punya created_at, dan
	// klausa invoice menyebut kolomnya tanpa awalan.
	bayar := []string{"p.invoice_id IN (SELECT id FROM v_laporan_invoice WHERE " + inv + ")"}
	bayar = append(bayar, rentangBayar(f)...)
	if err := r.db.QueryRow(ctx,
		"SELECT coalesce(sum(p.amount), 0) FROM payments p WHERE "+strings.Join(bayar, " AND "), args,
	).Scan(&out.DetailDiterima); err != nil {
		return nil, fmt.Errorf("laporan diterima per detail: %w", err)
	}
	return out, nil
}

// rentangBayar adalah saringan periode untuk tabel payments beralias p.
func rentangBayar(f domain.ReportFilter) []string {
	var out []string
	if f.From != "" {
		out = append(out, "p.paid_at >= @dari::date")
	}
	if f.To != "" {
		out = append(out, "p.paid_at < (@sampai::date + 1)")
	}
	return out
}

// Detail mengembalikan satu halaman invoice yang terbit pada periode, terbaru
// dulu, beserta jumlah seluruhnya.
//
// Kolom diterima menjumlahkan pembayaran invoice itu yang MASUK pada periode,
// dasar yang sama dengan tabel per chapter. Baris-baris ini harus bisa
// dijumlahkan kembali ke baris chapternya.
func (r *Repository) Detail(ctx context.Context, f domain.ReportFilter, limit, offset int) ([]domain.ReportDetailRow, int, error) {
	inv, _, _, args := saringan(ctx, f)

	var total int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM v_laporan_invoice WHERE "+inv, args).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung detail laporan: %w", err)
	}

	bayar := append([]string{"p.invoice_id = v.id"}, rentangBayar(f)...)
	args["limit"], args["offset"] = limit, offset

	rows, err := r.db.Query(ctx, `
		SELECT v.id, v.number, v.member_name, v.chapter_name, v.type, v.created_at, v.due_date,
		       v.status, v.amount,
		       coalesce((SELECT sum(p.amount) FROM payments p WHERE `+strings.Join(bayar, " AND ")+`), 0)
		FROM v_laporan_invoice v
		WHERE `+inv+`
		ORDER BY v.created_at DESC, v.number
		LIMIT @limit OFFSET @offset`, args)
	if err != nil {
		return nil, 0, fmt.Errorf("detail laporan: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ReportDetailRow, 0, limit)
	for rows.Next() {
		var d domain.ReportDetailRow
		if err := rows.Scan(&d.ID, &d.Number, &d.Member, &d.Chapter, &d.Type, &d.Terbit,
			&d.JatuhTempo, &d.Status, &d.Ditagih, &d.Diterima); err != nil {
			return nil, 0, fmt.Errorf("scan detail laporan: %w", err)
		}
		items = append(items, d)
	}
	return items, total, rows.Err()
}
