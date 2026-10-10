package paperid

import (
	"context"
	"strings"
	"testing"
)

// Callback Pembayaran ke Supplier: datar di akar, tanpa invoice, tanpa
// payment_info. Bentuk inilah yang di produksi berulang kali tercatat sebagai
// "TIDAK ADA identitas invoice ... payment_info TIDAK ADA".
const supplierPaymentJSON = `{
  "company_id": "c-1", "company_name": "BNI Finance",
  "partner_id": "p-9", "partner_name": "CV Vendor",
  "payment_id": "PAYOUT-2026-10-0007",
  "status": "SUCCESS",
  "account_holder_name": "CV Vendor", "account_number": "1234567890",
  "account_type": "bank", "amount": 2500000, "bank_code": "BNI",
  "currency": "IDR", "completed_time": "2026-10-09T10:00:00+07:00"
}`

// Inspektur tidak boleh mengeluh atas uang keluar yang memang tidak punya
// invoice. Catatan yang selalu berbunyi berhenti dibaca tepat saat ia penting.
func TestInspeksiTidakMengeluhAtasPembayaranSupplier(t *testing.T) {
	for _, n := range inspectPayload([]byte(supplierPaymentJSON)) {
		if strings.Contains(n, "TIDAK ADA identitas") || strings.Contains(n, "payment_info TIDAK ADA") {
			t.Errorf("catatan keliru untuk pembayaran supplier: %s", n)
		}
	}
}

// Dan di URL mana pun ia mendarat, uang keluar tidak melunasi invoice,
// walaupun statusnya SUCCESS.
func TestPembayaranSupplierTidakMelunasiDiEndpointMasuk(t *testing.T) {
	store := &stubStore{settleReturns: true}
	svc := newService(store, &stubGateway{}, "rahasia")
	settled, err := svc.HandleWebhook(context.Background(),
		"/api/v1/webhooks/paperid/payment-in", "rahasia", []byte(supplierPaymentJSON))
	if err != nil || settled || store.settleRef.called {
		t.Fatalf("uang keluar harus diakui tanpa melunasi: settled=%v err=%v settleDipanggil=%v",
			settled, err, store.settleRef.called)
	}
}

// Kunci yang hadir dengan nilai kosong BUKAN tanda keluarga pencairan.
// Payload yang dibangun dari struct membawa "disbursement_id": "" pada
// callback pembayaran biasa, dan itu tetap harus melunasi.
func TestKunciKosongBukanPencairan(t *testing.T) {
	if Pencairan([]byte(`{"disbursement_id":"","payment_id":"","additional_info":{"invoices":[{"uuid":"u"}]}}`)) {
		t.Error("kunci kosong dikira pencairan")
	}
	if !Pencairan([]byte(supplierPaymentJSON)) {
		t.Error("pembayaran supplier tidak dikenali sebagai pencairan")
	}
}
