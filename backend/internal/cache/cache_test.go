package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Tanpa REDIS_URL, *Redis bernilai nil dan setiap pemakainya harus berperilaku
// seperti satu instance tanpa cache. Panic di sini berarti API mati di setiap
// mesin pengembang dan di setiap deploy yang belum memasang Redis.
func TestRedisNilBerperilakuTanpaCache(t *testing.T) {
	ctx := context.Background()
	var r *Redis

	panggil := 0
	for i := 0; i < 2; i++ {
		v, err := Ingat(ctx, r, "k", time.Minute, func() (int, error) { panggil++; return 7, nil })
		if err != nil || v != 7 {
			t.Fatalf("Ingat = %d, %v", v, err)
		}
	}
	if panggil != 2 {
		t.Errorf("tanpa Redis, setiap baca harus ke sumber; terpanggil %d kali", panggil)
	}
	if !r.Sekali(ctx, "tugas", time.Hour) || !r.Sekali(ctx, "tugas", time.Hour) {
		t.Error("tanpa Redis, satu-satunya instance harus selalu mendapat giliran")
	}
	if r.Generasi(ctx, "g") != 0 || r.Hapus(ctx, "k") != nil || r.Ping(ctx) != nil || r.Close() != nil {
		t.Error("metode pada Redis nil harus diam")
	}
	r.NaikkanGenerasi(ctx, "g")
}

func TestBukaTanpaURLMengembalikanNil(t *testing.T) {
	r, err := Buka(context.Background(), "", nil)
	if r != nil || err != nil {
		t.Errorf("Buka(\"\") = %v, %v", r, err)
	}
	if _, err := Buka(context.Background(), "bukan-url", nil); err == nil {
		t.Error("REDIS_URL yang salah tulis harus menggagalkan start")
	}
}

func TestIngatTidakMenyimpanGalat(t *testing.T) {
	r := liveRedis(t)
	ctx := context.Background()
	gagal := errors.New("db mati")
	if _, err := Ingat(ctx, r, "galat", time.Minute, func() (int, error) { return 0, gagal }); !errors.Is(err, gagal) {
		t.Fatalf("galat sumber tidak diteruskan: %v", err)
	}
	v, err := Ingat(ctx, r, "galat", time.Minute, func() (int, error) { return 3, nil })
	if err != nil || v != 3 {
		t.Errorf("galat ikut tersimpan: dapat %d, %v", v, err)
	}
}
