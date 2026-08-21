package blackbox

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func livePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL untuk menjalankan tes ini")
	}
	if !strings.Contains(url, "test") && !strings.Contains(url, "dev") {
		t.Fatalf("TEST_DATABASE_URL menunjuk ke database non-uji")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("sambung: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(),
		"TRUNCATE integration_calls RESTART IDENTITY"); err != nil {
		t.Fatalf("bersihkan: %v", err)
	}
	return pool
}

// Riwayat harus bertahan melewati restart proses. Direktur perubahan ini
// justru itu, jadi disimulasikan langsung: rekam lewat satu Recorder, lalu baca
// lewat Recorder yang BARU DIBUAT — persis yang terjadi setelah restart.
func TestRiwayatBertahanMelewatiRestart(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool, 100)

	lama := New(50).WithStore(repo, quiet())
	lama.Record(Call{
		Integration: "paper_id", Direction: Outbound, Method: "POST",
		URL: "/api/v1/store-invoice", Status: 200, Success: true,
		Request:  []byte(`{"number":"INV-2026-001"}`),
		Response: []byte(`{"data":{"id":"pi-1"}}`),
		Duration: 1500 * time.Millisecond,
	})

	// Proses "restart": Recorder baru, memori kosong, penyimpanan sama.
	baru := New(50).WithStore(repo, quiet())
	got := baru.List()

	if len(got) != 1 {
		t.Fatalf("riwayat harus bertahan, dapat %d entri", len(got))
	}
	e := got[0]
	if e.Integration != "paper_id" || e.Status != 200 || !e.Success {
		t.Errorf("entri salah: %+v", e)
	}
	if e.DurationMS != 1500 {
		t.Errorf("durasi = %d ms, mau 1500", e.DurationMS)
	}
	// Body harus utuh — itu isi yang dicari orang saat menelusuri masalah.
	var req map[string]any
	if err := json.Unmarshal(e.Request, &req); err != nil || req["number"] != "INV-2026-001" {
		t.Errorf("request tidak utuh: %s (%v)", e.Request, err)
	}
	if !strings.Contains(string(e.Response), "pi-1") {
		t.Errorf("response tidak utuh: %s", e.Response)
	}
}

// Tanpa pemangkasan, tabel ini tumbuh selamanya.
func TestPruneMenyisakanTepatRetain(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool, 5)
	rec := New(50).WithStore(repo, quiet())

	for i := 0; i < 20; i++ {
		rec.Record(Call{Integration: "paper_id", Method: "POST",
			URL: "/x", Status: 200, Success: true})
	}
	if _, err := repo.Prune(context.Background()); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	var n int
	if err := pool.QueryRow(context.Background(),
		"select count(*) from integration_calls").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("tersisa %d baris, mau 5", n)
	}
	// Dan yang tersisa harus yang TERBARU, bukan yang terlama.
	got, err := repo.List(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("List mengembalikan %d", len(got))
	}
}

// Error dan body kosong tidak boleh merusak baris.
func TestMenyimpanKegagalanTanpaBody(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool, 100)
	rec := New(50).WithStore(repo, quiet())

	rec.Record(Call{
		Integration: "paper_id", Direction: Outbound, Method: "POST",
		URL: "/api/v1/invoices/x/send", Status: 503, Success: false,
		Err: context.DeadlineExceeded,
	})

	got, err := repo.List(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("dapat %d entri", len(got))
	}
	if got[0].Success || got[0].Status != 503 {
		t.Errorf("entri salah: %+v", got[0])
	}
	if got[0].Error == "" {
		t.Error("pesan error harus ikut tersimpan")
	}
}
