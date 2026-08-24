package reminder

import (
	"context"
	"log/slog"
	"time"
)

// Worker mengirim pengingat jatuh tempo secara berkala.
//
// Goroutine di dalam binary yang sudah ada, bukan kontainer terpisah. Menambah
// kontainer berarti menambah satu hal lagi yang bisa mati diam-diam tanpa ada
// yang menyadari, dan pengingat yang berhenti tanpa tanda persis sama buruknya
// dengan pengingat yang tidak pernah dibuat.
//
// TIGA PENJAGA, dan ketiganya menahan kegagalan yang berbeda:
//
//	advisory lock   dua replika tidak berjalan bersamaan
//	reminder_log    satu invoice x offset hanya sekali, bahkan setelah restart
//	sakelar mati    bawaannya MATI; menyalakannya keputusan sadar
//
// Yang ketiga bukan kehati-hatian berlebihan: worker ini mengirim pesan
// sungguhan ke member dan membakar nomor invoice Paper.id secara permanen.
// Menyalakannya lewat bawaan berarti deploy pertama langsung menghubungi semua
// orang.

// Pengirim adalah bagian Paper.id yang dibutuhkan worker.
type Pengirim interface {
	Remind(ctx context.Context, invoiceID string, opts RemindOptions) error
}

// RemindOptions mencerminkan paperid.SendOptions tanpa mengimpornya, supaya
// worker bisa diuji tanpa menyeret seluruh klien Paper.id.
type RemindOptions struct {
	Email    *bool
	WhatsApp *bool
}

// Store adalah kontrak persistensi worker.
type Store interface {
	Pengaturan(ctx context.Context) (Pengaturan, error)
	Kandidat(ctx context.Context, offset, batas int) ([]Kandidat, error)
	Klaim(ctx context.Context, invoiceID string, offset int) (bool, error)
	Catat(ctx context.Context, invoiceID string, offset int, berhasil bool, pesan string) error
	AmbilKunci(ctx context.Context) (func(), bool, error)
}

var _ Store = (*Repository)(nil)

// MaksPerPutaran membatasi pengiriman satu putaran.
//
// Bukan aturan bisnis: worker yang mengirim ribuan pengingat sekaligus akan
// menahan koneksi Paper.id selama berjam-jam dan menunda semua pengiriman
// manual yang antre di belakangnya. Sisanya diambil putaran berikutnya, dan
// karena kandidat disaring per hari, tidak ada yang tertinggal.
const MaksPerPutaran = 200

type Worker struct {
	repo     Store
	kirim    Pengirim
	log      *slog.Logger
	sekarang func() time.Time
}

func NewWorker(repo Store, kirim Pengirim, log *slog.Logger) *Worker {
	return &Worker{repo: repo, kirim: kirim, log: log, sekarang: time.Now}
}

// Jalankan memutar worker sampai context dibatalkan.
//
// Putaran PERTAMA tidak langsung mengirim: ia menunggu satu interval lebih dulu.
// Restart yang beruntun — deploy bergilir, kontainer yang gagal lalu dijadwal
// ulang — kalau tidak, masing-masing memicu putaran penuh dalam hitungan detik.
func (w *Worker) Jalankan(ctx context.Context) {
	set, err := w.repo.Pengaturan(ctx)
	if err != nil {
		w.log.Error("worker pengingat: pengaturan tidak terbaca, worker tidak dijalankan", "error", err)
		return
	}
	w.log.Info("worker pengingat siap",
		"aktif", set.WorkerAktif, "interval", set.Interval, "offsets", set.Offsets)

	t := time.NewTicker(set.Interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker pengingat berhenti")
			return
		case <-t.C:
			w.Putaran(ctx)
		}
	}
}

// Putaran menjalankan satu siklus. Diekspor supaya bisa diuji tanpa menunggu
// ticker, dan supaya bisa dipicu manual saat menelusuri masalah.
func (w *Worker) Putaran(ctx context.Context) {
	set, err := w.repo.Pengaturan(ctx)
	if err != nil {
		w.log.Error("worker pengingat: pengaturan tidak terbaca", "error", err)
		return
	}
	// Dua sakelar, dan keduanya diperiksa tiap putaran: mematikannya harus
	// berlaku pada putaran berikutnya, bukan setelah proses direstart.
	if !set.WorkerAktif || !set.NotifikasiAktif || len(set.Offsets) == 0 {
		return
	}

	lepas, dapat, err := w.repo.AmbilKunci(ctx)
	if err != nil {
		w.log.Error("worker pengingat: gagal mengambil kunci", "error", err)
		return
	}
	if !dapat {
		// Replika lain sedang bekerja. Bukan galat.
		return
	}
	defer lepas()

	var terkirim, gagal, dilewati int
	for _, offset := range set.Offsets {
		kandidat, err := w.repo.Kandidat(ctx, offset, MaksPerPutaran)
		if err != nil {
			w.log.Error("worker pengingat: gagal mencari kandidat", "offset", offset, "error", err)
			continue
		}
		for _, k := range kandidat {
			if ctx.Err() != nil {
				return
			}
			// Diklaim SEBELUM dikirim. Lihat Repository.Klaim untuk alasan
			// urutannya.
			ok, err := w.repo.Klaim(ctx, k.InvoiceID, offset)
			if err != nil {
				w.log.Error("worker pengingat: gagal klaim", "invoice", k.Number, "error", err)
				continue
			}
			if !ok {
				dilewati++
				continue
			}

			err = w.kirim.Remind(ctx, k.InvoiceID, RemindOptions{})
			pesan := ""
			if err != nil {
				pesan = err.Error()
				gagal++
			} else {
				terkirim++
			}
			if err := w.repo.Catat(ctx, k.InvoiceID, offset, err == nil, pesan); err != nil {
				w.log.Error("worker pengingat: hasil tidak tercatat", "invoice", k.Number, "error", err)
			}
		}
	}

	if terkirim > 0 || gagal > 0 {
		w.log.Info("worker pengingat selesai",
			"terkirim", terkirim, "gagal", gagal, "dilewati", dilewati)
	}
}
