package cache

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"
)

// liveRedis menyambung ke TEST_REDIS_URL dan mengosongkan kunci aplikasi.
func liveRedis(t *testing.T) *Redis {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("set TEST_REDIS_URL untuk menjalankan tes ini")
	}
	r, err := Buka(context.Background(), url, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	ctx := context.Background()
	keys, err := r.rdb.Keys(ctx, awalan+"*").Result()
	if err != nil {
		t.Fatalf("redis tidak bisa dihubungi: %v", err)
	}
	if len(keys) > 0 {
		r.rdb.Del(ctx, keys...)
	}
	return r
}

func TestIngatMenyimpanSampaiDihapus(t *testing.T) {
	r := liveRedis(t)
	ctx := context.Background()
	panggil := 0
	ambil := func() (map[string]int, error) { panggil++; return map[string]int{"n": panggil}, nil }

	a, _ := Ingat(ctx, r, "biaya", time.Minute, ambil)
	b, _ := Ingat(ctx, r, "biaya", time.Minute, ambil)
	if panggil != 1 || b["n"] != a["n"] {
		t.Fatalf("baca kedua tidak dari cache: panggil=%d a=%v b=%v", panggil, a, b)
	}
	if err := r.Hapus(ctx, "biaya"); err != nil {
		t.Fatal(err)
	}
	c, _ := Ingat(ctx, r, "biaya", time.Minute, ambil)
	if panggil != 2 || c["n"] != 2 {
		t.Errorf("setelah Hapus, baca harus ke sumber lagi: panggil=%d c=%v", panggil, c)
	}
}

func TestGenerasiNaikMenggugurkanKunciLama(t *testing.T) {
	r := liveRedis(t)
	ctx := context.Background()
	g0 := r.Generasi(ctx, "gen")
	r.NaikkanGenerasi(ctx, "gen")
	if g1 := r.Generasi(ctx, "gen"); g1 != g0+1 {
		t.Errorf("generasi %d -> %d", g0, g1)
	}
}

// Dua replika berebut giliran yang sama: hanya satu yang menang.
func TestSekaliHanyaSatuPemenang(t *testing.T) {
	r := liveRedis(t)
	ctx := context.Background()
	if !r.Sekali(ctx, "kurs:giliran", time.Minute) {
		t.Fatal("pemanggil pertama harus mendapat giliran")
	}
	if r.Sekali(ctx, "kurs:giliran", time.Minute) {
		t.Error("pemanggil kedua juga mendapat giliran")
	}
}

// TTL penghitung dihitung dari kegagalan PERTAMA. Bila setiap Tambah
// memperpanjangnya, penyerang yang menebak pelan tidak pernah lepas, dan
// pemilik akun yang salah ketik terkunci jauh lebih lama dari JendelaGagal.
func TestTambahTidakMemperpanjangTTL(t *testing.T) {
	r := liveRedis(t)
	ctx := context.Background()
	if _, err := r.Tambah(ctx, "login:gagal:a", 2*time.Second); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	n, err := r.Tambah(ctx, "login:gagal:a", 2*time.Second)
	if err != nil || n != 2 {
		t.Fatalf("Tambah = %d, %v", n, err)
	}
	_, sisa, err := r.Hitung(ctx, "login:gagal:a")
	if err != nil || sisa > time.Second {
		t.Errorf("sisa TTL %v: Tambah kedua memperpanjang jendela", sisa)
	}
	if n, _, _ := r.Hitung(ctx, "login:gagal:kosong"); n != 0 {
		t.Errorf("kunci yang tidak ada = %d", n)
	}
}

// Setelah satu kegagalan, Redis dilewati selama jeda: baca langsung ke sumber
// tanpa menunggu batasWaktu lebih dulu.
func TestGagalMenjedakanRedis(t *testing.T) {
	r := liveRedis(t)
	ctx := context.Background()
	r.catat("get", "k", errors.New("putus"))
	if r.siap() {
		t.Fatal("Redis masih dipanggil sesudah gagal")
	}
	if _, err := r.Tambah(ctx, "login:gagal:x", time.Minute); !errors.Is(err, errJeda) {
		t.Errorf("Tambah selama jeda = %v, ingin errJeda supaya pembatas jatuh ke memori", err)
	}
	if !r.Sekali(ctx, "tugas", time.Minute) {
		t.Error("selama jeda giliran harus didapat")
	}
	r.lewatiSampai.Store(0)
	if !r.siap() {
		t.Error("jeda tidak berakhir")
	}
}
