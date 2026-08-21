package paperid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/blackbox"
	"github.com/syabanf/bni-finance/backend/internal/domain"
)

func remindableAt(status domain.InvoiceStatus, reminders int) *Sendable {
	return &Sendable{
		ID: "inv-1", Number: "INV-2026-001", Amount: 250000,
		Type: domain.TypeRenewal, Status: status,
		DueDate:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		MemberID: "mem-1", Name: "Budi", Email: "b@contoh.local",
		Phone: "6282240274833", ReminderCount: reminders,
	}
}

// Hulu tiruan yang menangkap payload, supaya isinya bisa diperiksa tanpa
// menyentuh Paper.id sungguhan dan tanpa membakar nomor invoice.
func huluPenangkap(t *testing.T, got *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(got)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"id":"pi-r1","number":"X","payper_url":"https://bayar/r1"},"status_code":200}`))
	}))
}

// Nomor pengingat HARUS berbeda dari nomor kanonik: Paper.id membakar nomor
// secara permanen dan menolak pengiriman kedua dengan nomor sama. Kalau aturan
// ini lepas, setiap pengingat gagal 409 di hulu.
func TestPengingatMemakaiNomorTurunan(t *testing.T) {
	var payload map[string]any
	hulu := huluPenangkap(t, &payload)
	defer hulu.Close()

	store := &stubStore{sendable: remindableAt(domain.StatusSent, 0)}
	svc := NewService(store, hulu.URL, "id", "secret", "tok", blackbox.New(20))

	if _, err := svc.Remind(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Remind: %v", err)
	}
	if got := payload["number"]; got != "INV-2026-001-R1" {
		t.Errorf("nomor dokumen = %v, mau INV-2026-001-R1", got)
	}

	// Pengingat kedua harus naik, bukan mengulang -R1.
	store.sendable = remindableAt(domain.StatusSent, 1)
	if _, err := svc.Remind(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Remind kedua: %v", err)
	}
	if got := payload["number"]; got != "INV-2026-001-R2" {
		t.Errorf("nomor pengingat kedua = %v, mau INV-2026-001-R2", got)
	}
}

// Memundurkan jatuh tempo membuat tunggakan tampak belum jatuh tempo — persis
// kebalikan dari maksud sebuah pengingat.
func TestPengingatTidakMemundurkanJatuhTempo(t *testing.T) {
	var payload map[string]any
	hulu := huluPenangkap(t, &payload)
	defer hulu.Close()

	svc := NewService(&stubStore{sendable: remindableAt(domain.StatusOverdue, 0)},
		hulu.URL, "id", "secret", "tok", blackbox.New(20))

	if _, err := svc.Remind(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Remind: %v", err)
	}
	// Paper.id memakai format dd-mm-yyyy.
	if got, want := payload["due_date"], "01-03-2026"; got != want {
		t.Errorf("due_date = %v, mau %v (tanggal asli invoice, bukan dihitung ulang)", got, want)
	}
}

// Batas yang paling penting: tagihan tertutup tidak boleh diingatkan.
func TestPengingatDitolakUntukStatusYangSalah(t *testing.T) {
	kasus := []struct {
		status domain.InvoiceStatus
		pesan  string
	}{
		{domain.StatusDraft, "belum pernah dikirim"},
		{domain.StatusPaid, "tidak bisa diingatkan"},
		{domain.StatusCancelled, "tidak bisa diingatkan"},
	}
	for _, k := range kasus {
		t.Run(string(k.status), func(t *testing.T) {
			hulu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("Paper.id TIDAK boleh dihubungi untuk status ini")
			}))
			defer hulu.Close()

			rec := blackbox.New(20)
			svc := NewService(&stubStore{sendable: remindableAt(k.status, 0)},
				hulu.URL, "id", "secret", "tok", rec)

			_, err := svc.Remind(context.Background(), "inv-1", SendOptions{})
			if err == nil {
				t.Fatalf("status %s harus ditolak", k.status)
			}
			if !strings.Contains(err.Error(), k.pesan) {
				t.Errorf("pesan %q tidak memuat %q", err.Error(), k.pesan)
			}
			// Dan kegagalannya tetap tercatat — blackbox untuk integrasi utuh.
			var ops int
			for _, e := range rec.List() {
				if strings.Contains(e.URL, "/remind") {
					ops++
					if e.Success || len(e.Response) == 0 {
						t.Errorf("entri pengingat gagal harus punya response: %+v", e)
					}
				}
			}
			if ops != 1 {
				t.Errorf("harus 1 entri pengingat di blackbox, dapat %d", ops)
			}
		})
	}
}

// Pengingat bukan penerbitan: menaikkan overdue kembali ke sent akan menghapus
// fakta bahwa tagihan itu sudah lewat jatuh tempo.
func TestPengingatTidakMengubahStatus(t *testing.T) {
	var payload map[string]any
	hulu := huluPenangkap(t, &payload)
	defer hulu.Close()

	store := &stubStore{sendable: remindableAt(domain.StatusOverdue, 0)}
	svc := NewService(store, hulu.URL, "id", "secret", "tok", blackbox.New(20))

	if _, err := svc.Remind(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Remind: %v", err)
	}
	if store.sentWith != nil {
		t.Error("Remind tidak boleh memanggil MarkSent — itu mengubah status")
	}
	if store.remindedWith == nil {
		t.Fatal("Remind harus memanggil MarkReminded")
	}
}
