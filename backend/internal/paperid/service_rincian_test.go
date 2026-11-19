package paperid

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/blackbox"
	"github.com/syabanf/bni-finance/backend/internal/domain"
)

func rincianContoh() []domain.BarisBiaya {
	return []domain.BarisBiaya{
		{Kode: domain.BiayaKeanggotaan, Nama: "Membership Fee", Jumlah: 10_381_280},
		{Kode: domain.BiayaPPN, Nama: "VAT (11%)", Jumlah: 1_141_920},
	}
}

// Invoice yang menyimpan rincian dikirim ke Paper.id sebagai satu baris per
// komponen, dan jumlah barisnya sama dengan nominal invoice.
func TestSendMengirimSatuBarisPerKomponen(t *testing.T) {
	s := draftSendable()
	s.Amount, s.Rincian = 11_523_200, rincianContoh()
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1", PaymentURL: "https://bayar/x"}}

	if _, err := newService(&stubStore{sendable: s}, gw, "tok").Send(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	var badan struct {
		Items []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Quantity    int    `json:"quantity"`
			Price       int64  `json:"price"`
		} `json:"items"`
	}
	mentah, _ := json.Marshal(buildCreateRequest(gw.gotIn))
	if err := json.Unmarshal(mentah, &badan); err != nil {
		t.Fatal(err)
	}
	if len(badan.Items) != 2 {
		t.Fatalf("terkirim %d baris, ingin 2: %s", len(badan.Items), mentah)
	}
	if badan.Items[0].Name != "Membership Fee" || badan.Items[1].Name != "VAT (11%)" {
		t.Errorf("nama baris: %q, %q", badan.Items[0].Name, badan.Items[1].Name)
	}
	var total int64
	for _, it := range badan.Items {
		total += it.Price
		if it.Quantity != 1 || it.Description == "" {
			t.Errorf("baris %q: kuantitas %d, deskripsi %q", it.Name, it.Quantity, it.Description)
		}
	}
	if total != s.Amount {
		t.Errorf("jumlah baris %d tidak sama dengan nominal invoice %d", total, s.Amount)
	}
}

// Tanpa rincian, invoice tetap satu baris berisi total seperti sebelumnya.
func TestSendTanpaRincianTetapSatuBaris(t *testing.T) {
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1", PaymentURL: "https://bayar/x"}}
	if _, err := newService(&stubStore{sendable: draftSendable()}, gw, "tok").Send(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	req := buildCreateRequest(gw.gotIn)
	if len(req.Items) != 1 || req.Items[0].Price != 1_500_000 {
		t.Errorf("items = %+v, ingin satu baris 1.500.000", req.Items)
	}
}

// Rincian yang tidak berjumlah sama dengan nominal tidak dikirim. Paper.id
// menagih jumlah barisnya, jadi baris yang salah berarti tagihan yang salah.
func TestRincianTidakCocokJatuhKeSatuBaris(t *testing.T) {
	s := draftSendable()
	s.Amount, s.Rincian = 9_999_999, rincianContoh()
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1", PaymentURL: "https://bayar/x"}}

	if _, err := newService(&stubStore{sendable: s}, gw, "tok").Send(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	req := buildCreateRequest(gw.gotIn)
	if len(req.Items) != 1 || req.Items[0].Price != 9_999_999 {
		t.Errorf("items = %+v, ingin satu baris berisi nominal invoice", req.Items)
	}
}

// Pengingat membawa baris yang sama dengan invoicenya.
func TestPengingatMembawaRincian(t *testing.T) {
	var payload map[string]any
	hulu := huluPenangkap(t, &payload)
	defer hulu.Close()

	s := remindableAt(domain.StatusSent, 0)
	s.Amount, s.Rincian = 11_523_200, rincianContoh()
	svc := NewService(&stubStore{sendable: s}, hulu.URL, "id", "secret", "tok", blackbox.New(20))
	if _, err := svc.Remind(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Remind: %v", err)
	}
	items, _ := payload["items"].([]any)
	if len(items) != 2 {
		t.Errorf("pengingat membawa %d baris, ingin 2", len(items))
	}
}
