package invoice

import (
	"context"
	"log/slog"
	"time"
)

// Penyapu adalah bagian Repository yang dipakai Worker.
type Penyapu interface {
	TandaiOverdue(ctx context.Context) (int, error)
}

var _ Penyapu = (*Repository)(nil)

// Worker menandai invoice terkirim yang sudah lewat jatuh tempo sebagai
// overdue.
//
// Sebelumnya status itu hanya ditulis oleh browser, saat invoicenya kebetulan
// tampil di daftar seseorang. Invoice yang tidak pernah dibuka tetap berstatus
// sent, dan angka Overdue di dashboard, yang dihitung server dari status itu,
// ikut terlalu kecil.
//
// Tidak perlu giliran antar replika: pernyataannya idempoten. Replika kedua
// yang menjalankannya pada saat yang sama tidak menemukan baris tersisa.
type Worker struct {
	repo     Penyapu
	log      *slog.Logger
	interval time.Duration
	setelah  func(context.Context)
}

// NewWorker memeriksa setiap 15 menit. Tanggal hanya berganti sekali sehari,
// tetapi interval pendek membuat server yang baru menyala tengah hari tidak
// menunggu lama, dan satu putaran tanpa hasil hanyalah satu pemindaian indeks.
func NewWorker(repo Penyapu, log *slog.Logger) *Worker {
	return &Worker{repo: repo, log: log, interval: 15 * time.Minute}
}

// SetelahMenandai mendaftarkan fungsi yang dipanggil bila ada invoice yang
// baru ditandai. Dipakai untuk menggugurkan cache angka.
func (w *Worker) SetelahMenandai(fn func(context.Context)) *Worker {
	w.setelah = fn
	return w
}

// Jalankan menyapu sekali saat start, lalu setiap interval, sampai ctx batal.
func (w *Worker) Jalankan(ctx context.Context) {
	w.Putaran(ctx)
	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.Putaran(ctx)
		}
	}
}

// Putaran menjalankan satu sapuan. Diekspor supaya bisa diuji tanpa ticker.
func (w *Worker) Putaran(ctx context.Context) {
	n, err := w.repo.TandaiOverdue(ctx)
	if err != nil {
		// Putaran berikutnya mencoba lagi; status yang tertinggal 15 menit
		// tidak merusak apa pun.
		w.log.Warn("gagal menandai invoice overdue", "error", err)
		return
	}
	if n == 0 {
		return
	}
	w.log.Info("invoice ditandai overdue", "jumlah", n)
	if w.setelah != nil {
		w.setelah(ctx)
	}
}
