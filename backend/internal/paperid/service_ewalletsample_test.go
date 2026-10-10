package paperid

import (
	"context"
	"testing"
)

// Callback e-wallet SUNGGUHAN dari staging Paper.id (OVO lewat payper), atas
// dokumen pengingat -R1. Dua field di payment_info tidak ada di dokumentasi:
// ewallet_type dan grand_amount. Sebelum ini inspektur mencatatnya sebagai
// "field tidak dikenal" pada setiap callback e-wallet, dan catatan yang muncul
// terus-menerus adalah catatan yang berhenti dibaca orang.
const contohEwalletPaperID = `{
  "ref_id": "REF1791615248kVvvz",
  "message": "transaction succeed",
  "external_id": "17916152485HT6C",
  "payment_date": "2026-10-10",
  "payment_info": {
    "method": "ewallet",
    "source": "payper",
    "status": "PAID",
    "channel": "OVO",
    "ewallet": {
      "amount": 12700000,
      "status": "PAID",
      "created": "2026-10-10T13:54:08.412246833+07:00",
      "paid_at": "2026-10-10T13:54:12.13120811+07:00",
      "updated": "2026-10-10T13:54:12.13120811+07:00",
      "paid_amount": 12700000,
      "supplier_payment_fee": 0
    },
    "ewallet_type": "ovo",
    "grand_amount": 12941300
  },
  "additional_info": {
    "invoices": [
      { "uuid": "6a6c98df-5065-404b-b886-99d61716ab3f", "number": "INV-2026-032-R1" }
    ]
  }
}`

func TestContohEwalletPaperIDMelunasi(t *testing.T) {
	store := &stubStore{settleReturns: true}
	svc := newService(store, &stubGateway{}, "rahasia")
	settled, err := svc.HandleWebhook(context.Background(),
		"/api/v1/webhooks/paperid/payment-in", "rahasia", []byte(contohEwalletPaperID))
	if err != nil || !settled {
		t.Fatalf("callback e-wallet harus melunasi: settled=%v err=%v", settled, err)
	}
	r := store.settleRef
	if r.paperID != "6a6c98df-5065-404b-b886-99d61716ab3f" || r.number != "INV-2026-032-R1" {
		t.Errorf("identitas invoice salah dibaca: %+v", r)
	}
	// Nominal dari ewallet.paid_amount, bukan grand_amount yang memuat biaya.
	if r.amount != 12_700_000 || r.method != "ewallet:OVO" || r.status != "PAID" {
		t.Errorf("detail pembayaran salah dibaca: amount=%d method=%q status=%q", r.amount, r.method, r.status)
	}
	for _, n := range inspectPayload([]byte(contohEwalletPaperID)) {
		t.Errorf("inspektur mengeluh atas callback e-wallet sungguhan: %s", n)
	}
}
