package paperid

import (
	"context"
	"testing"
)

// Contoh payload PERSIS seperti yang ditampilkan dashboard Paper.id pada
// kotak "Contoh callback payload" di pengaturan Pembayaran Masuk. Bukan data
// pribadi; ini contoh resmi mereka. Format payment_date-nya "01-01-2021
// 23:59:59", berbeda dari dokumentasi API yang memakai "2025-06-18".
const contohDashboardPaperID = `{
  "additional_info": {
    "invoices": [
      { "uuid": "580efeb6-5887-4973-ab13-099e22598adf", "number": "INV/2025/11/0001" }
    ]
  },
  "message": "transaction success",
  "payment_date": "01-01-2021 23:59:59",
  "payment_info": {
    "bank_transfer": {
      "amount": 200000,
      "created": "2021-08-09T11:23:50.550571+07:00",
      "paid_amount": 200000,
      "paid_at": "2021-08-09T11:23:50.550571+07:00",
      "status": "PAID",
      "updated": "2021-08-09T11:23:53.011389+07:00"
    },
    "channel": "bni",
    "method": "bank_transfer"
  },
  "ref_id": "987654xxxxx",
  "external_id": "123456xxxx"
}`

func TestContohDashboardPaperIDMelunasi(t *testing.T) {
	store := &stubStore{settleReturns: true}
	svc := newService(store, &stubGateway{}, "rahasia")
	settled, err := svc.HandleWebhook(context.Background(),
		"/api/v1/webhooks/paperid/payment-in", "rahasia", []byte(contohDashboardPaperID))
	if err != nil || !settled {
		t.Fatalf("contoh dashboard harus melunasi: settled=%v err=%v", settled, err)
	}
	r := store.settleRef
	if r.paperID != "580efeb6-5887-4973-ab13-099e22598adf" || r.number != "INV/2025/11/0001" {
		t.Errorf("identitas invoice salah dibaca: %+v", r)
	}
	if r.amount != 200000 || r.method != "bank_transfer:bni" || r.status != "PAID" {
		t.Errorf("detail pembayaran salah dibaca: amount=%d method=%q status=%q", r.amount, r.method, r.status)
	}
	// Catatan inspektur tidak boleh mengeluhkan bentuk resmi ini.
	for _, n := range inspectPayload([]byte(contohDashboardPaperID)) {
		t.Errorf("inspektur mengeluh atas contoh resmi: %s", n)
	}
}
