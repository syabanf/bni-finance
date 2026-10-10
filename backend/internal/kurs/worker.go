package kurs

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// Kunci app_settings yang dipakai worker.
const (
	// KunciOtomatis mematikan pembaruan otomatis bila bernilai "false".
	// Kosong atau nilai lain berarti menyala: kurs yang diperbarui adalah
	// keadaan yang diminta, dan mematikannya harus keputusan seseorang.
	KunciOtomatis = "kurs_kmk_otomatis"
	// KunciTerakhir menyimpan hasil pemeriksaan terakhir sebagai JSON, untuk
	// ditampilkan di halaman Pengaturan.
	KunciTerakhir = "kurs_kmk_terakhir"
)

// PenulisOtomatis tercatat di updated_by fee_settings, supaya riwayat biaya
// membedakan kurs yang diubah orang dari yang diubah worker.
const PenulisOtomatis = "Kurs KMK otomatis"

// Store adalah bagian settings.Service yang dipakai worker.
type Store interface {
	GetFees(ctx context.Context) (*domain.FeeSettings, error)
	UpdateFees(ctx context.Context, in domain.UpdateFeeSettingsInput) (*domain.FeeSettings, error)
	GetApp(ctx context.Context, key string) (*domain.AppSetting, error)
	SetApp(ctx context.Context, key string, in domain.SetAppSettingInput) (*domain.AppSetting, error)
}

// Terakhir adalah isi KunciTerakhir.
type Terakhir struct {
	KMK
	Diperiksa time.Time `json:"diperiksa"`
	// Diterapkan true bila pemeriksaan ini mengubah kurs di fee_settings.
	Diterapkan bool `json:"diterapkan"`
}

// Worker memeriksa kurs KMK sekali sehari.
//
// KMK terbit mingguan (berlaku Rabu sampai Selasa), tapi tanggal terbitnya
// tidak selalu tepat. Memeriksa harian berarti kurs baru dipakai paling lambat
// sehari setelah terbit, dan pemeriksaan yang tidak menemukan perubahan tidak
// menulis apa pun ke fee_settings.
type Worker struct {
	store    Store
	client   *http.Client
	url      string
	log      *slog.Logger
	interval time.Duration
	sekarang func() time.Time
}

func NewWorker(store Store, log *slog.Logger) *Worker {
	return &Worker{
		store: store, client: &http.Client{Timeout: 30 * time.Second},
		url: SumberURL, log: log, interval: 24 * time.Hour, sekarang: time.Now,
	}
}

// Jalankan memeriksa sekali saat start, lalu setiap interval, sampai ctx batal.
//
// Pemeriksaan pertama langsung jalan karena ia hanya membaca halaman publik;
// tidak ada pesan yang terkirim dan tidak ada nomor yang terbakar. Server yang
// baru dideploy tidak perlu menunggu sehari untuk kurs yang benar.
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

// Putaran menjalankan satu pemeriksaan. Diekspor supaya bisa diuji tanpa ticker.
func (w *Worker) Putaran(ctx context.Context) {
	if !w.aktif(ctx) {
		return
	}
	kmk, err := Ambil(ctx, w.client, w.url)
	if err != nil {
		// Kurs lama tetap berlaku. Halaman yang sedang tidak bisa dibuka atau
		// berganti bentuk tidak boleh menghentikan penagihan.
		w.log.Warn("kurs KMK tidak terbaca, kurs tersimpan tetap dipakai", "error", err)
		return
	}

	fees, err := w.store.GetFees(ctx)
	if err != nil {
		w.log.Error("kurs KMK: biaya tersimpan tidak terbaca", "error", err)
		return
	}

	diterapkan := false
	if fees.UsdRate != kmk.USD {
		penulis := PenulisOtomatis
		if _, err := w.store.UpdateFees(ctx, domain.UpdateFeeSettingsInput{
			UsdRate: &kmk.USD, UpdatedBy: &penulis,
		}); err != nil {
			w.log.Error("kurs KMK gagal diterapkan", "error", err, "kmk", kmk.Nomor)
			return
		}
		diterapkan = true
		w.log.Info("kurs KMK diterapkan", "kmk", kmk.Nomor, "lama", fees.UsdRate, "baru", kmk.USD)
	}

	isi, _ := json.Marshal(Terakhir{KMK: *kmk, Diperiksa: w.sekarang(), Diterapkan: diterapkan})
	if _, err := w.store.SetApp(ctx, KunciTerakhir, domain.SetAppSettingInput{Value: string(isi)}); err != nil {
		w.log.Warn("kurs KMK: catatan pemeriksaan gagal disimpan", "error", err)
	}
}

func (w *Worker) aktif(ctx context.Context) bool {
	s, err := w.store.GetApp(ctx, KunciOtomatis)
	if errors.Is(err, httpx.ErrNotFound) {
		return true
	}
	if err != nil {
		w.log.Warn("kurs KMK: sakelar tidak terbaca, pemeriksaan dilewati", "error", err)
		return false
	}
	return s.Value != "false"
}
