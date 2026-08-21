package paperid

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/blackbox"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// Blackbox dipakai untuk menjawab "tadi kenapa kirimnya gagal". Sebelum
// perbaikan ini, tiga kegagalan paling sering justru tidak meninggalkan jejak
// apa pun: semuanya terjadi SEBELUM panggilan HTTP ke Paper.id, sementara satu-
// satunya perekaman ada di dalam klien HTTP. Dari dashboard, pengguna melihat
// "gagal" lalu membuka blackbox dan menemukannya kosong.
//
// Tes ini mengunci kontraknya: setiap hasil Send — berhasil maupun gagal di
// titik mana pun — meninggalkan tepat satu entri operasi.

type sendStub struct {
	sendable *Sendable
	sent     *domain.Invoice
	getErr   error
}

func (s *sendStub) GetSendable(context.Context, string) (*Sendable, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.sendable, nil
}
func (s *sendStub) MarkSent(context.Context, string, CreateResult, time.Time, time.Time, string) (*domain.Invoice, error) {
	return s.sent, nil
}
func (s *sendStub) MarkReminded(context.Context, string, CreateResult, time.Time, string) (*domain.Invoice, error) {
	return s.sent, nil
}
func (s *sendStub) SettleByRef(context.Context, string, string, string, string, int64, time.Time) (bool, error) {
	return false, nil
}
func (s *sendStub) GetSetting(context.Context, string) (string, error)     { return "", nil }
func (s *sendStub) PaperInvoiceID(context.Context, string) (string, error) { return "", nil }

// opsBaru mengambil entri operasi kirim, mengabaikan entri percakapan HTTP —
// keduanya memang sengaja dicatat terpisah.
func opsEntries(rec *blackbox.Recorder) []blackbox.Entry {
	var out []blackbox.Entry
	for _, e := range rec.List() {
		if strings.Contains(e.URL, "/send") {
			out = append(out, e)
		}
	}
	return out
}

func TestSendSelaluMasukBlackbox(t *testing.T) {
	draft := &Sendable{ID: "inv-1", Number: "INV-2026-001", Status: domain.StatusDraft}
	paid := &Sendable{ID: "inv-2", Number: "INV-2026-002", Status: domain.StatusPaid}

	kasus := []struct {
		nama        string
		konfigurasi bool
		store       *sendStub
		wantStatus  int
	}{
		{"gateway belum dikonfigurasi", false, &sendStub{sendable: draft}, 503},
		{"invoice tidak ditemukan", true, &sendStub{getErr: httpx.ErrNotFound}, 404},
		{"invoice bukan draft", true, &sendStub{sendable: paid}, 409},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			rec := blackbox.New(50)
			id, secret := "", ""
			if k.konfigurasi {
				id, secret = "uji-id", "uji-secret"
			}
			svc := NewService(k.store, "https://contoh.invalid", id, secret, "tok", rec)

			if _, err := svc.Send(context.Background(), "inv-1", SendOptions{}); err == nil {
				t.Fatal("harusnya gagal")
			}

			got := opsEntries(rec)
			if len(got) != 1 {
				t.Fatalf("harus tepat 1 entri operasi di blackbox, dapat %d — "+
					"kegagalan ini tidak terlihat oleh siapa pun", len(got))
			}
			if got[0].Success {
				t.Error("entri kegagalan tidak boleh ditandai success")
			}
			if got[0].Status != k.wantStatus {
				t.Errorf("status = %d, mau %d", got[0].Status, k.wantStatus)
			}
			if !strings.Contains(got[0].URL, "/send") {
				t.Errorf("URL harus menunjuk operasi kirim, dapat %q", got[0].URL)
			}
		})
	}
}

// Status yang bukan httpx.Error tetap harus tercatat, bukan hilang.
func TestSendMerekamGalatTakTerdugaSebagai500(t *testing.T) {
	rec := blackbox.New(50)
	svc := NewService(&sendStub{getErr: errors.New("koneksi database putus")},
		"https://contoh.invalid", "id", "secret", "tok", rec)

	if _, err := svc.Send(context.Background(), "inv-1", SendOptions{}); err == nil {
		t.Fatal("harusnya gagal")
	}
	got := opsEntries(rec)
	if len(got) != 1 || got[0].Status != 500 {
		t.Fatalf("galat tak terduga harus tercatat sebagai 500, dapat %+v", got)
	}
}

// Sisi BERHASIL juga wajib tercatat — permintaannya "success atau gagal".
// Hulu Paper.id diganti server tiruan supaya tidak ada nomor invoice sungguhan
// yang terbakar; klien HTTP asli tetap ikut diuji.
func TestSendBerhasilJugaMasukBlackbox(t *testing.T) {
	hulu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"id":"pi-99","number":"INV-2026-001",` +
			`"payper_url":"https://bayar.contoh/pi-99","pdf_url":"https://pdf.contoh/pi-99"},` +
			`"status_code":200}`))
	}))
	defer hulu.Close()

	rec := blackbox.New(50)
	store := &sendStub{
		sendable: &Sendable{
			ID: "inv-1", Number: "INV-2026-001", Amount: 250000,
			Type: domain.TypeRenewal, Status: domain.StatusDraft,
			DueDate:  time.Now().Add(24 * time.Hour),
			MemberID: "mem-1", Name: "Uji Kirim",
			Email: "uji@contoh.local", Phone: "6282240274833",
		},
		sent: &domain.Invoice{ID: "inv-1", Number: "INV-2026-001", Status: domain.StatusSent},
	}
	svc := NewService(store, hulu.URL, "id", "secret", "tok", rec)

	if _, err := svc.Send(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("kirim harusnya berhasil: %v", err)
	}

	ops := opsEntries(rec)
	if len(ops) != 1 || !ops[0].Success || ops[0].Status != 200 {
		t.Fatalf("entri operasi berhasil tidak benar: %+v", ops)
	}

	// Dan percakapan HTTP ke hulu tetap tercatat terpisah — dua sudut pandang
	// berbeda atas satu kejadian, keduanya berguna saat mendiagnosis.
	var upstream int
	for _, e := range rec.List() {
		if strings.Contains(e.URL, "store-invoice") {
			upstream++
			if !e.Success {
				t.Error("panggilan hulu 200 tidak boleh ditandai gagal")
			}
		}
	}
	if upstream != 1 {
		t.Errorf("panggilan hulu harus tercatat 1 kali, dapat %d", upstream)
	}
}

// Entri tanpa response setengah berguna: halaman blackbox menampilkan
// permintaannya lalu berhenti, dan pertanyaan "hasilnya apa" tetap tak terjawab
// — padahal itu justru alasan orang membukanya. Sempat begitu: entri operasi
// punya request tetapi response-nya string kosong, dan yang dilaporkan
// pengguna persis "kirim ke Paper.id tapi tidak ada response di blackbox".
//
// Blackbox ini untuk melihat integrasi secara UTUH, jadi kedua sisi wajib
// terisi: apa yang diminta, dan apa hasilnya.
func TestEntriKirimSelaluPunyaResponse(t *testing.T) {
	t.Run("gagal memuat pesan yang diterima pemanggil", func(t *testing.T) {
		rec := blackbox.New(50)
		svc := NewService(&sendStub{sendable: &Sendable{
			ID: "inv-2", Status: domain.StatusPaid,
		}}, "https://contoh.invalid", "id", "secret", "tok", rec)

		_, err := svc.Send(context.Background(), "inv-2", SendOptions{})
		if err == nil {
			t.Fatal("harusnya gagal")
		}
		ops := opsEntries(rec)
		if len(ops) != 1 {
			t.Fatalf("harus 1 entri, dapat %d", len(ops))
		}
		if len(ops[0].Response) == 0 {
			t.Fatal("response kosong — entri kegagalan tidak menjelaskan apa pun")
		}
		// Pesannya harus yang sama dengan yang diterima pemanggil, bukan
		// ringkasan lain yang bisa menyimpang darinya.
		if !strings.Contains(string(ops[0].Response), err.Error()) {
			t.Errorf("response %s tidak memuat pesan error %q", ops[0].Response, err.Error())
		}
	})

	t.Run("berhasil memuat hasil yang bisa ditindaklanjuti", func(t *testing.T) {
		hulu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data":{"id":"pi-1","number":"INV-2026-009",` +
				`"payper_url":"https://bayar.contoh/pi-1"},"status_code":200}`))
		}))
		defer hulu.Close()

		rec := blackbox.New(50)
		paperID, payURL := "pi-1", "https://bayar.contoh/pi-1"
		store := &sendStub{
			sendable: &Sendable{
				ID: "inv-9", Number: "INV-2026-009", Amount: 250000,
				Type: domain.TypeRenewal, Status: domain.StatusDraft,
				DueDate:  time.Now().Add(24 * time.Hour),
				MemberID: "mem-9", Name: "Uji Kirim",
				Email: "uji@contoh.local", Phone: "6282240274833",
			},
			sent: &domain.Invoice{
				ID: "inv-9", Number: "INV-2026-009", Status: domain.StatusSent,
				PaperIDInvoiceID: &paperID, PaperIDPaymentURL: &payURL,
			},
		}
		svc := NewService(store, hulu.URL, "id", "secret", "tok", rec)
		if _, err := svc.Send(context.Background(), "inv-9", SendOptions{}); err != nil {
			t.Fatalf("kirim harusnya berhasil: %v", err)
		}

		ops := opsEntries(rec)
		if len(ops) != 1 || len(ops[0].Response) == 0 {
			t.Fatalf("entri operasi harus punya response: %+v", ops)
		}
		// Yang menjadikannya berguna: nomor, status, dan tautan bayar.
		for _, wajib := range []string{"INV-2026-009", "sent", "pi-1", payURL} {
			if !strings.Contains(string(ops[0].Response), wajib) {
				t.Errorf("response tidak memuat %q: %s", wajib, ops[0].Response)
			}
		}
	})
}
