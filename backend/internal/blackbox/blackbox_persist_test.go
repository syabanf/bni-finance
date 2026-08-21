package blackbox

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
)

// Store tiruan: cukup untuk membuktikan kontraknya tanpa menuntut Postgres.
type fakeStore struct {
	rows     []Entry
	pruned   int
	failNext error
}

func (f *fakeStore) Insert(_ context.Context, e Entry) error {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}
	f.rows = append([]Entry{e}, f.rows...)
	return nil
}
func (f *fakeStore) List(_ context.Context, limit int) ([]Entry, error) {
	if limit > len(f.rows) {
		limit = len(f.rows)
	}
	return f.rows[:limit], nil
}
func (f *fakeStore) Clear(context.Context) error          { f.rows = nil; return nil }
func (f *fakeStore) Prune(context.Context) (int64, error) { f.pruned++; return 0, nil }

type errStore struct{ fakeStore }

func (e *errStore) List(context.Context, int) ([]Entry, error) {
	return nil, errors.New("database mati")
}

func quiet() *slog.Logger { return slog.New(slog.DiscardHandler) }

// Inti perubahannya: rekaman harus sampai ke penyimpanan, bukan hanya memori.
func TestRecordMenulisKePenyimpanan(t *testing.T) {
	fs := &fakeStore{}
	r := New(10).WithStore(fs, quiet())

	r.Record(Call{Integration: "paper_id", Direction: Outbound,
		Method: "POST", URL: "/x", Success: true, Status: 200})

	if len(fs.rows) != 1 {
		t.Fatalf("penyimpanan harus punya 1 baris, dapat %d", len(fs.rows))
	}
	if fs.rows[0].Integration != "paper_id" || !fs.rows[0].Success {
		t.Errorf("baris tersimpan salah: %+v", fs.rows[0])
	}
}

// Riwayat dibaca dari penyimpanan, bukan dari memori — kalau tidak, halaman
// blackbox tetap hanya menampilkan yang tersisa sejak restart terakhir.
func TestListMembacaDariPenyimpanan(t *testing.T) {
	fs := &fakeStore{rows: []Entry{
		{ID: "9", Integration: "paper_id", URL: "/lama-tapi-tersimpan"},
	}}
	r := New(10).WithStore(fs, quiet())

	got := r.List()
	if len(got) != 1 || got[0].URL != "/lama-tapi-tersimpan" {
		t.Fatalf("harus membaca dari penyimpanan, dapat %+v", got)
	}
}

// Kotak hitam yang diam-diam berhenti merekam lebih buruk daripada yang tidak
// ada — tetapi ia juga tidak boleh menjatuhkan panggilan integrasinya.
func TestGagalMenyimpanTidakMenjatuhkanPerekaman(t *testing.T) {
	fs := &fakeStore{failNext: errors.New("disk penuh")}
	r := New(10).WithStore(fs, quiet())

	r.Record(Call{Integration: "paper_id", Method: "POST", URL: "/x"})

	// Tidak panik, dan salinan memorinya tetap ada sebagai cadangan.
	r.store = nil // paksa List membaca memori
	if got := r.List(); len(got) != 1 {
		t.Fatalf("salinan memori harus tetap ada saat penyimpanan gagal, dapat %d", len(got))
	}
}

// Pembacaan yang gagal harus jatuh ke memori, bukan mengosongkan halaman tepat
// saat seseorang sedang menelusuri masalah.
func TestListJatuhKeMemoriSaatPenyimpananGagal(t *testing.T) {
	r := New(10).WithStore(&errStore{}, quiet())
	r.Record(Call{Integration: "paper_id", Method: "POST", URL: "/masih-di-memori"})

	got := r.List()
	if len(got) != 1 || got[0].URL != "/masih-di-memori" {
		t.Fatalf("harus jatuh ke memori, dapat %+v", got)
	}
}

func TestClearMengosongkanPenyimpananJuga(t *testing.T) {
	fs := &fakeStore{}
	r := New(10).WithStore(fs, quiet())
	r.Record(Call{Integration: "paper_id", Method: "POST", URL: "/x"})

	r.Clear()

	if len(fs.rows) != 0 {
		t.Errorf("penyimpanan harus ikut kosong, tersisa %d", len(fs.rows))
	}
	if len(r.List()) != 0 {
		t.Error("memori harus kosong")
	}
}

// Tanpa penyimpanan ia tetap berfungsi penuh — tes dan mode Data Contoh tidak
// boleh menuntut database.
func TestTanpaPenyimpananTetapBerjalan(t *testing.T) {
	r := New(10)
	r.Record(Call{Integration: "paper_id", Method: "POST", URL: "/x",
		Request: json.RawMessage(`{"a":1}`)})
	if len(r.List()) != 1 {
		t.Fatal("recorder tanpa store harus tetap merekam ke memori")
	}
}
