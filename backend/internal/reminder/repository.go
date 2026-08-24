package reminder

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// lockID adalah kunci advisory yang dipegang worker selama satu putaran.
//
// Angkanya sembarang tapi TETAP: dua proses yang memakai angka sama tidak akan
// pernah berjalan bersamaan. Tanpa ini, dua replika yang dijalankan berdampingan
// akan sama-sama melihat invoice yang sama dan berlomba menulis reminder_log —
// dan meski indeks uniknya menahan duplikat, keduanya tetap membuang panggilan
// dan menghasilkan galat yang membingungkan.
const lockID int64 = 0x726d6e64 // "rmnd"

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// Pengaturan adalah konfigurasi worker, dibaca ulang tiap putaran.
//
// Dibaca ulang dan tidak di-cache: mematikan worker lewat pengaturan harus
// berlaku pada putaran berikutnya, bukan setelah proses direstart. Saat pesan
// sudah telanjur salah terkirim, menunggu restart bukan pilihan.
type Pengaturan struct {
	NotifikasiAktif bool
	WorkerAktif     bool
	Offsets         []int
	Interval        time.Duration
}

func (r *Repository) Pengaturan(ctx context.Context) (Pengaturan, error) {
	rows, err := r.db.Query(ctx, `
		SELECT key, value FROM app_settings
		WHERE key IN ('notifications_enabled','reminder_worker_enabled',
		              'reminder_offsets','reminder_worker_interval')`)
	if err != nil {
		return Pengaturan{}, fmt.Errorf("baca pengaturan pengingat: %w", err)
	}
	defer rows.Close()

	// Bawaannya MATI untuk keduanya. Pengaturan yang gagal terbaca tidak boleh
	// menghasilkan worker yang menyala sendiri: pesan yang telanjur keluar tidak
	// bisa ditarik kembali, dan nomor Paper.id yang terbakar tidak bisa dipakai
	// ulang.
	out := Pengaturan{Interval: time.Hour}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return Pengaturan{}, fmt.Errorf("scan pengaturan pengingat: %w", err)
		}
		v = strings.TrimSpace(v)
		switch k {
		case "notifications_enabled":
			out.NotifikasiAktif = v == "true"
		case "reminder_worker_enabled":
			out.WorkerAktif = v == "true"
		case "reminder_offsets":
			out.Offsets = bacaOffsets(v)
		case "reminder_worker_interval":
			if d, err := time.ParseDuration(v); err == nil && d >= time.Minute {
				out.Interval = d
			}
		}
	}
	return out, rows.Err()
}

// bacaOffsets membaca "7,3,1" menjadi [7 3 1].
//
// Nilai yang tidak masuk akal dibuang diam-diam, bukan membuat seluruh daftar
// gagal: satu salah ketik di antara lima angka tidak boleh mematikan keempat
// pengingat lainnya. Nol dan negatif dibuang karena "mengingatkan H+0" adalah
// jatuh tempo hari ini — itu urusan penanda overdue, bukan pengingat.
func bacaOffsets(s string) []int {
	var out []int
	lihat := map[int]bool{}
	for _, bagian := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(bagian))
		if err != nil || n <= 0 || n > 365 || lihat[n] {
			continue
		}
		lihat[n] = true
		out = append(out, n)
	}
	return out
}

// Kandidat adalah satu invoice yang perlu diingatkan.
type Kandidat struct {
	InvoiceID string
	Number    string
	Offset    int
}

// Kandidat mencari invoice yang jatuh tempo TEPAT `offset` hari lagi dan belum
// pernah diingatkan pada offset itu.
//
// "Tepat", bukan "kurang dari": dengan `<=`, invoice yang sama akan terus
// muncul setiap putaran sampai jatuh tempo, dan satu-satunya yang menahannya
// hanyalah reminder_log. Menyaringnya di sini membuat pekerjaan tiap putaran
// tetap kecil, dan reminder_log menjadi pengaman kedua alih-alih satu-satunya.
func (r *Repository) Kandidat(ctx context.Context, offset int, batas int) ([]Kandidat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.id, i.number
		FROM invoices i
		WHERE i.status IN ('sent','overdue')
		  -- $1 di-cast EKSPLISIT ke integer. Tanpa itu Postgres tidak bisa
		  -- memilih antara "date + integer" dan "date + interval", dan
		  -- menolaknya dengan "operator is not unique" — worker berjalan
		  -- normal, tapi tidak pernah menemukan satu kandidat pun.
		  AND i.due_date = current_date + ($1::integer)
		  AND NOT EXISTS (
		        SELECT 1 FROM reminder_log l
		        WHERE l.invoice_id = i.id AND l.offset_hari = $1)
		ORDER BY i.due_date, i.number
		LIMIT $2`, offset, batas)
	if err != nil {
		return nil, fmt.Errorf("cari kandidat pengingat: %w", err)
	}
	defer rows.Close()

	var out []Kandidat
	for rows.Next() {
		var k Kandidat
		if err := rows.Scan(&k.InvoiceID, &k.Number); err != nil {
			return nil, fmt.Errorf("scan kandidat: %w", err)
		}
		k.Offset = offset
		out = append(out, k)
	}
	return out, rows.Err()
}

// Klaim menandai satu pengingat sebagai MILIK proses ini, sebelum dikirim.
//
// Mengembalikan false bila sudah pernah diklaim. Inilah penjaga sebenarnya
// terhadap kiriman ganda, dan urutannya menentukan: baris ditulis SEBELUM
// Paper.id dipanggil.
//
// Kalau panggilannya kemudian gagal, satu pengingat terlewat — merugikan, tapi
// bisa dikirim manual dan terlihat di kolom error. Kalau urutannya dibalik dan
// proses mati di antara panggilan dan pencatatan, pengingatnya sudah terkirim
// tanpa tercatat, dan putaran berikutnya mengirimnya lagi. Kelalaian lebih
// murah daripada duplikat yang membakar nomor invoice.
func (r *Repository) Klaim(ctx context.Context, invoiceID string, offset int) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO reminder_log (invoice_id, offset_hari)
		VALUES ($1,$2) ON CONFLICT DO NOTHING`, invoiceID, offset)
	if err != nil {
		return false, fmt.Errorf("klaim pengingat: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// Catat menyimpan hasil pengiriman pada baris yang sudah diklaim.
func (r *Repository) Catat(ctx context.Context, invoiceID string, offset int, berhasil bool, pesan string) error {
	var e *string
	if pesan != "" {
		e = &pesan
	}
	_, err := r.db.Exec(ctx, `
		UPDATE reminder_log SET berhasil = $3, error = $4, sent_at = now()
		WHERE invoice_id = $1 AND offset_hari = $2`, invoiceID, offset, berhasil, e)
	if err != nil {
		return fmt.Errorf("catat hasil pengingat: %w", err)
	}
	return nil
}

// AmbilKunci mencoba memegang advisory lock untuk satu putaran.
//
// Session-scoped, jadi WAJIB dilepas — dan dilepas lewat defer di pemanggilnya.
// Lock yang tidak dilepas akan menahan seluruh putaran berikutnya sampai
// prosesnya mati.
func (r *Repository) AmbilKunci(ctx context.Context) (lepas func(), dapat bool, err error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("ambil koneksi: %w", err)
	}
	var ok bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&ok); err != nil {
		conn.Release()
		return nil, false, fmt.Errorf("ambil kunci: %w", err)
	}
	if !ok {
		conn.Release()
		return nil, false, nil
	}
	return func() {
		// Context terpisah: context putaran bisa sudah dibatalkan saat worker
		// berhenti, dan kunci yang gagal dilepas karena itu akan menahan
		// seluruh proses berikutnya.
		lepasCtx, batal := context.WithTimeout(context.Background(), 5*time.Second)
		defer batal()
		_, _ = conn.Exec(lepasCtx, "SELECT pg_advisory_unlock($1)", lockID)
		conn.Release()
	}, true, nil
}
