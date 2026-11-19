package paperid

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/blackbox"
	"github.com/syabanf/bni-finance/backend/internal/domain"
)

const (
	rekening = "Transfer ke BCA 1234567890 a.n. Contoh Organisasi"
	syarat   = "Pembayaran paling lambat pada tanggal jatuh tempo."
)

// Detail rekening serta syarat dan ketentuan diatur dari Pengaturan, lalu ikut
// terkirim di kolom notes dan terms_condition invoice Paper.id.
func TestSendMembawaCatatanDanSyarat(t *testing.T) {
	store := &stubStore{
		sendable: draftSendable(),
		settings: map[string]string{"paper_catatan": "  " + rekening + "\n", "paper_syarat": syarat},
	}
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1", PaymentURL: "https://bayar/x"}}

	if _, err := newService(store, gw, "tok").Send(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gw.gotIn.Notes != rekening {
		t.Errorf("notes = %q, ingin detail rekening tanpa spasi tepi", gw.gotIn.Notes)
	}
	if gw.gotIn.Terms != syarat {
		t.Errorf("terms = %q, ingin syarat dari pengaturan", gw.gotIn.Terms)
	}
}

// Tanpa pengaturan, kedua kolom tidak dikirim sama sekali. Kolom kosong yang
// ikut terkirim mencetak judul "Syarat & Ketentuan" tanpa isi di invoice orang.
func TestTanpaCatatanKolomnyaTidakDikirim(t *testing.T) {
	store := &stubStore{sendable: draftSendable()}
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1", PaymentURL: "https://bayar/x"}}

	if _, err := newService(store, gw, "tok").Send(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	badan, err := json.Marshal(buildCreateRequest(gw.gotIn))
	if err != nil {
		t.Fatal(err)
	}
	for _, kolom := range []string{`"notes"`, `"terms_condition"`} {
		if strings.Contains(string(badan), kolom) {
			t.Errorf("kolom %s ikut terkirim walau kosong: %s", kolom, badan)
		}
	}
}

// Pengingat menyebut invoice yang diingatkan, lalu membawa detail rekening dan
// syarat yang sama: member yang membayar dari pesan pengingat membutuhkannya.
func TestPengingatMembawaCatatanDanSyarat(t *testing.T) {
	var payload map[string]any
	hulu := huluPenangkap(t, &payload)
	defer hulu.Close()

	store := &stubStore{
		sendable: remindableAt(domain.StatusSent, 0),
		settings: map[string]string{"paper_catatan": rekening, "paper_syarat": syarat},
	}
	svc := NewService(store, hulu.URL, "id", "secret", "tok", blackbox.New(20))
	if _, err := svc.Remind(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Remind: %v", err)
	}

	ingin := "Pengingat pembayaran untuk invoice INV-2026-001\n\n" + rekening
	if payload["notes"] != ingin {
		t.Errorf("notes = %q, ingin %q", payload["notes"], ingin)
	}
	if payload["terms_condition"] != syarat {
		t.Errorf("terms_condition = %v, ingin syarat dari pengaturan", payload["terms_condition"])
	}
}
