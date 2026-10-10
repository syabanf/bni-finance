package kurs

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type storeUji struct {
	rate       int64
	app        map[string]string
	diperbarui []domain.UpdateFeeSettingsInput
}

func (s *storeUji) GetFees(context.Context) (*domain.FeeSettings, error) {
	return &domain.FeeSettings{UsdRate: s.rate}, nil
}
func (s *storeUji) UpdateFees(_ context.Context, in domain.UpdateFeeSettingsInput) (*domain.FeeSettings, error) {
	s.diperbarui = append(s.diperbarui, in)
	s.rate = *in.UsdRate
	return &domain.FeeSettings{UsdRate: s.rate}, nil
}
func (s *storeUji) GetApp(_ context.Context, key string) (*domain.AppSetting, error) {
	v, ok := s.app[key]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	return &domain.AppSetting{Key: key, Value: v}, nil
}
func (s *storeUji) SetApp(_ context.Context, key string, in domain.SetAppSettingInput) (*domain.AppSetting, error) {
	s.app[key] = in.Value
	return &domain.AppSetting{Key: key, Value: in.Value}, nil
}

func workerUji(t *testing.T, store *storeUji, halaman string, status int) *Worker {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		io.WriteString(w, halaman)
	}))
	t.Cleanup(srv.Close)
	w := NewWorker(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	w.url = srv.URL
	w.sekarang = func() time.Time { return time.Date(2026, 10, 10, 6, 0, 0, 0, time.UTC) }
	return w
}

// Kurs KMK yang berbeda diterapkan sekali, dengan penulis yang menyebut
// worker; pemeriksaan ulang tanpa perubahan tidak menulis ke fee_settings.
func TestPutaranMenerapkanKursBaru(t *testing.T) {
	store := &storeUji{rate: 16_000, app: map[string]string{}}
	w := workerUji(t, store, halamanContoh, http.StatusOK)

	w.Putaran(context.Background())
	if len(store.diperbarui) != 1 || *store.diperbarui[0].UsdRate != 17_935 {
		t.Fatalf("kurs harus diterapkan sekali ke 17935: %+v", store.diperbarui)
	}
	if got := *store.diperbarui[0].UpdatedBy; got != PenulisOtomatis {
		t.Errorf("updated_by = %q", got)
	}
	// Hanya kurs yang dikirim; USD harga tidak disentuh, Rupiah diturunkan
	// settings.Service dari USD tersimpan.
	if store.diperbarui[0].RegistrationFeeUSD != nil || store.diperbarui[0].RenewalFeeUSD != nil {
		t.Error("worker tidak boleh mengirim harga USD")
	}
	var catatan Terakhir
	if err := json.Unmarshal([]byte(store.app[KunciTerakhir]), &catatan); err != nil || !catatan.Diterapkan || catatan.Nomor != "48/MK/EF.2/2026" {
		t.Errorf("catatan pemeriksaan salah: %+v %v", catatan, err)
	}

	w.Putaran(context.Background())
	if len(store.diperbarui) != 1 {
		t.Errorf("pemeriksaan tanpa perubahan tidak boleh menulis: %d kali", len(store.diperbarui))
	}
}

func TestPutaranMenghormatiSakelarDanGagalDenganAman(t *testing.T) {
	t.Run("dimatikan", func(t *testing.T) {
		store := &storeUji{rate: 16_000, app: map[string]string{KunciOtomatis: "false"}}
		workerUji(t, store, halamanContoh, http.StatusOK).Putaran(context.Background())
		if len(store.diperbarui) != 0 {
			t.Error("sakelar mati tetapi kurs tetap diterapkan")
		}
	})
	t.Run("halaman galat", func(t *testing.T) {
		store := &storeUji{rate: 16_000, app: map[string]string{}}
		workerUji(t, store, "gangguan", http.StatusServiceUnavailable).Putaran(context.Background())
		if store.rate != 16_000 || len(store.diperbarui) != 0 {
			t.Error("galat halaman tidak boleh mengubah kurs")
		}
	})
	t.Run("halaman berubah bentuk", func(t *testing.T) {
		store := &storeUji{rate: 16_000, app: map[string]string{}}
		workerUji(t, store, "<html><p>Situs dalam pemeliharaan</p></html>", http.StatusOK).Putaran(context.Background())
		if store.rate != 16_000 {
			t.Error("halaman tak dikenal tidak boleh mengubah kurs")
		}
	})
}
