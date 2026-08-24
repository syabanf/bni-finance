package paperid

import (
	"context"
	"testing"
)

// Kedelapan payload di bawah disalin dari CONTOH DI DASHBOARD Paper.id, bukan
// dari dokumentasi. Keduanya berbeda, dan bedanya bukan kosmetik:
//
//   - Static VA membungkus payment_info di dalam `data`, bukan di akar
//   - Invoice Dibayar menaruh `invoice` di akar, bukan di `data.invoice`
//
// Parser yang hanya membaca satu bentuk akan diam-diam gagal pada bentuk yang
// lain — dan diam-diam berarti callback dijawab 200 tanpa apa pun terjadi.

const dbPembayaranKeluar = `{
  "additional_info": { "invoices": [ { "uuid": "580efeb6-5887-4973-ab13-099e22598adf", "number": "INV/2025/11/0001" } ] },
  "message": "transaction success", "payment_date": "01-01-2021 23:59:59",
  "payment_info": { "bank_transfer": { "amount": 200000, "created": "2021-08-09T11:23:50.550571+07:00",
      "paid_amount": 200000, "paid_at": "2021-08-09T11:23:50.550571+07:00", "status": "PAID",
      "updated": "2021-08-09T11:23:53.011389+07:00" },
    "channel": "bni", "method": "bank_transfer" },
  "ref_id": "987654xxxxx", "external_id": "123456xxxx"
}`

const dbPembayaranSupplier = `{
  "account_holder_name": "John Cena", "account_number": "7880633165",
  "account_type": "bank_account", "amount": 100000, "bank_code": "BCA",
  "company_id": "6e5b24c9-961b-4dw8-z588-1461dc6d2g92", "company_name": "Testing 123",
  "completed_time": "2023-12-11T17:02:13.000Z", "currency": "IDR",
  "disbursement_id": "PI-DISB1702984122NSR2Z", "partner_id": "d7981528-124e-4424-1414-c3895f06e7zd",
  "partner_name": "Testing Paperchain", "payment_id": "51236230-90b7-4120-9a08-23a2ae2061g4",
  "status": "COMPLETED"
}`

const dbPaylater = `{
  "transaction_id": "3852698c-b6b1-4b16-8646-9c7a5bc8a3f9", "ref_id": "000000021",
  "transaction_date": "10-11-2021", "payment_date": "2021-11-10",
  "transaction_status": "requested",
  "payment_info": { "method": "paper_usaha", "channel": "paper",
    "paper_usaha": { "payment_amount": 200000, "date": "10-11-2021", "due_date": "17-11-2021" },
    "additional_info": null, "message": "transaction is requested to be accepted or rejected" }
}`

const dbStaticVA = `{
  "company_id": "6e5b24c9-961b-4dw8-z588-1461dc6d2g92",
  "data": { "additional_info": {}, "message": "Static VA payment received",
    "partner": { "id": "d7981528-124e-4424-1414-c3895f06e7zd", "number": "PARTNER-001" },
    "payment_date": "2024-01-01",
    "payment_info": { "channel": "bca", "method": "static_va", "source": "recon_static_va",
      "status": "PAID",
      "va_facilitator": { "amount": 200000, "created": "2024-01-01T00:00:00+07:00",
        "paid_amount": 200000, "paid_at": "2024-01-01T00:00:00+07:00", "status": "PAID",
        "updated": "2024-01-01T00:00:01+07:00" } } }
}`

const dbDisbursement = `{
  "message": "Disbursement successful",
  "data": { "id": "DISB1700000000XXXXX", "type": "SCHEDULED_WITHDRAWAL", "status": "SUCCESS",
    "datetime": "2024-01-01T00:00:00+07:00", "channel": "BCA", "account_number": "1234567890",
    "amount": 1000000, "fee": 5000, "received_amount": 995000, "currency": "IDR",
    "active_balance_after": 5000000, "payment_references": [ "REF000000001XXXXX" ] }
}`

// Bentuk dashboard: invoice di AKAR, bukan di data.invoice.
const dbInvoiceDibayar = `{
  "invoice": { "amount": 10000, "amount_due": 10000, "id": "100414474",
    "number": "INV/API/001/0001", "status": "paid" },
  "message": "Invoice Paid",
  "payment_info": { "bank_transfer": { "amount": 200000,
      "created": "2021-08-09T11:23:50.550571+07:00", "paid_amount": 0, "paid_at": "",
      "status": "PAID", "updated": "2021-08-09T11:23:53.011389+07:00" },
    "channel": "bni", "method": "bank_transfer" }
}`

const dbSisaTagihan = `{
  "message": "Invoice outstanding amount due has been updated",
  "data": { "invoice": { "id": "afef0ea1-caa6-4123-9735-749df749642c",
      "number": "INV/2025/06/126", "partner_id": "2d0e80ca-9a46-4e35-a951-a61987fb7c98",
      "status": "partially paid", "amount_due": 300000, "total_amount": 1000000,
      "updated_at": "2026-04-23 15:25:13.000000 +0700 WIB" },
    "payment": { "parent_external_id": "REF1768318335GRNRB", "previous_external_id": null,
      "external_id": "REF1768318335GRNRB", "reconciled_amount": 600000 },
    "connected_documents": [ { "id": "afef0ea1-caa6-4123-9735-749df749642c", "type": "pay-01",
      "number": "SR/2026/0001", "date": "02-06-2026", "name": "Sales Receipt",
      "payment_method": "Cash", "notes": "cash", "amount": 600000 } ] }
}`

// Kedelapan bentuk harus terbaca — status dan identitas invoicenya, di mana pun
// letaknya.
func TestKedelapanPayloadDashboardTerbaca(t *testing.T) {
	kasus := []struct {
		nama      string
		raw       string
		mauStatus string
		mauNomor  string
		mauAmount float64
	}{
		{"Pembayaran Keluar/Masuk", dbPembayaranKeluar, "PAID", "INV/2025/11/0001", 200000},
		{"Static VA (payment_info di dalam data)", dbStaticVA, "PAID", "", 200000},
		{"Invoice Dibayar (invoice di akar)", dbInvoiceDibayar, "PAID", "INV/API/001/0001", 200000},
		{"Callback Sisa Tagihan", dbSisaTagihan, "PARTIALLY PAID", "INV/2025/06/126", 0},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			in := parse(t, k.raw)
			if got := in.settlementStatus(); got != k.mauStatus {
				t.Errorf("status = %q, mau %q", got, k.mauStatus)
			}
			if k.mauNomor != "" {
				if _, number := in.invoiceRef(); number != k.mauNomor {
					t.Errorf("nomor invoice = %q, mau %q", number, k.mauNomor)
				}
			}
			if k.mauAmount > 0 {
				pay := summarizePayment(in.paymentInfo())
				got := pay.Detail.PaidAmount
				if got == 0 {
					got = pay.Detail.Amount
				}
				if got != k.mauAmount {
					t.Errorf("amount = %v, mau %v", got, k.mauAmount)
				}
			}
		})
	}
}

// Payload yang SAH tidak boleh menghasilkan keluhan format. Catatan yang selalu
// berbunyi adalah catatan yang berhenti dibaca tepat saat ia penting.
func TestInspeksiTidakMengeluhAtasKedelapanPayloadDashboard(t *testing.T) {
	for nama, raw := range map[string]string{
		"Pembayaran Keluar":      dbPembayaranKeluar,
		"Pembayaran ke Supplier": dbPembayaranSupplier,
		"Paylater Diajukan":      dbPaylater,
		"Static VA":              dbStaticVA,
		"Disbursement Pay In":    dbDisbursement,
		"Invoice Dibayar":        dbInvoiceDibayar,
		"Sisa Tagihan Invoice":   dbSisaTagihan,
	} {
		if notes := inspectPayload([]byte(raw)); len(notes) > 0 {
			t.Errorf("%s: payload sah tidak boleh dikeluhkan, dapat %v", nama, notes)
		}
	}
}

// Static VA yang sah tapi belum menunjuk invoice harus DITERIMA, bukan ditolak.
//
// Paper.id mengirim ulang callback yang dijawab 4xx. Menolak bentuk yang memang
// benar berarti antrean kiriman ulang yang tidak pernah habis — untuk callback
// yang sejak awal tidak salah apa pun.
func TestStaticVATanpaInvoiceDiterima200(t *testing.T) {
	store := &stubStore{}
	svc := newService(store, &stubGateway{}, "rahasia")

	settled, err := svc.HandleWebhook(context.Background(),
		"/api/v1/webhooks/paperid/static-va", "rahasia", []byte(dbStaticVA))
	if err != nil {
		t.Fatalf("Static VA tanpa invoice ditolak %d: %v", statusOf(err), err)
	}
	if settled {
		t.Error("tidak ada invoice yang ditunjuk, tapi sesuatu dilaporkan lunas")
	}
	if store.settleRef.called {
		t.Error("SettleByRef dipanggil tanpa sasaran invoice")
	}
}

// Paylater "requested" bukan pembayaran dan tidak boleh melunasi apa pun.
func TestPaylaterRequestedTidakMelunasi(t *testing.T) {
	store := &stubStore{}
	svc := newService(store, &stubGateway{}, "rahasia")

	settled, err := svc.HandleWebhook(context.Background(),
		"/api/v1/webhooks/paperid/paylater", "rahasia", []byte(dbPaylater))
	if err != nil {
		t.Fatalf("paylater ditolak %d: %v", statusOf(err), err)
	}
	if settled || store.settleRef.called {
		t.Errorf("paylater requested melunasi invoice (settled=%v, dipanggil=%v)",
			settled, store.settleRef.called)
	}
}
