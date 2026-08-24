package paperid

import (
	"encoding/json"
	"testing"
)

// Payload di bawah disalin PERSIS dari dokumentasi resmi Paper.id dan dari
// contoh di dashboard mereka. Bukan buatan sendiri — dan itu intinya.
//
// Versi parser sebelumnya ditulis dari tebakan: ia membaca payment_info.status
// dan payment_info.amount sebagai field datar, padahal keduanya bersarang di
// objek yang DINAMAI MENURUT METODE pembayaran. Diuji terhadap payload asli,
// ketiganya terbaca kosong — artinya tidak satu pun callback pernah bisa
// melunasi invoice, dan kegagalannya sunyi karena encoding/json tidak protes.

const bankTransferAsli = `{
  "additional_info": { "invoices": [ { "uuid": "580efeb6-5887-4973-ab13-099e22598adf", "number": "INV/2025/11/0001" } ] },
  "message": "transaction succeed",
  "payment_date": "2025-06-18",
  "payment_info": {
    "bank_transfer": { "amount": 12557, "created": "2025-06-18T04:38:54.415786634+07:00",
      "paid_amount": 12557, "paid_at": "2025-06-18T04:39:43.594065797+07:00",
      "status": "PAID", "updated": "2025-06-18T04:39:43.594065797+07:00" },
    "channel": "bca_manual", "method": "bank_transfer", "source": "open-api", "status": "PAID"
  },
  "ref_id": "PAY-REF/2025/06/IN/123"
}`

const qrisAsli = `{
  "additional_info": { "invoices": [ { "uuid": "u-qris", "number": "INV/2025/11/0002" } ] },
  "payment_date": "2025-06-18",
  "payment_info": { "channel": "qris", "event": "qr.payment", "method": "qris",
    "qris": { "amount": 10000, "paid_amount": 10000, "paid_at": "2025-06-18T05:16:59.511610249+07:00", "status": "PAID" },
    "source": "open-api", "status": "PAID" },
  "ref_id": "PAY-REF/2025/06/IN/124"
}`

const creditCardAsli = `{
  "additional_info": { "invoices": [ { "uuid": "u-cc", "number": "INV/2025/11/0003" } ] },
  "payment_info": { "channel": "Visa", "method": "credit_card",
    "credit_card": { "amount": 10000, "paid_amount": 10000, "paid_at": "2025-06-18T05:19:01.192687186+07:00", "status": "PAID" },
    "status": "PAID" },
  "ref_id": "PAY-REF/2025/06/IN/125"
}`

const ewalletAsli = `{
  "additional_info": { "invoices": [ { "uuid": "u-ew", "number": "INV/2025/11/0004" } ] },
  "payment_info": { "channel": "OVO", "method": "ewallet",
    "ewallet": { "amount": 10000, "paid_amount": 10000, "paid_at": "2025-06-18T05:21:21.741879097+07:00", "status": "PAID" },
    "status": "PAID" },
  "ref_id": "PAY-REF/2025/06/IN/127"
}`

// Contoh di dashboard Paper.id TIDAK punya payment_info.status yang datar —
// hanya yang bersarang. Membaca satu tempat saja membuat payload sah ini
// terbaca sebagai bukan-pelunasan.
const dashboardTanpaStatusDatar = `{
  "additional_info": { "invoices": [ { "uuid": "u-dash", "number": "INV/2025/11/0005" } ] },
  "message": "transaction success",
  "payment_date": "01-01-2021 23:59:59",
  "payment_info": {
    "bank_transfer": { "amount": 200000, "paid_amount": 200000,
      "paid_at": "2021-08-09T11:23:50.550571+07:00", "status": "PAID" },
    "channel": "bni", "method": "bank_transfer" },
  "ref_id": "987654xxxxx", "external_id": "123456xxxx"
}`

const invoiceCallbackAsli = `{
  "message": "Invoice has been paid",
  "data": { "invoice": { "id": "afef0ea1-caa6-4123-9735-749df749642c",
    "number": "INV/2025/06/126", "partner_id": "2d0e80ca-9a46-4e35-a951-a61987fb7c98",
    "status": "paid", "amount_due": 0, "total_amount": 10000,
    "updated_at": "2025-06-18 05:21:16.529804427 +0700 WIB" } }
}`

const rekonsiliasiPenuh = `{
  "additional_info": { "invoices": [ { "number": "test/2025/07/0049" }, { "number": "INV/2025/11/0011" } ] },
  "message": "reconciliation succeed", "reconciliation_date": "2025-08-21",
  "reconciled_amount": 123123, "source": "recon_static_va",
  "payment_info": { "channel": "Permata",
    "static_va": { "amount": 200000, "created_at": "2025-08-20T15:34:01.592096867+07:00" },
    "payment_type": "digital_payment", "method": "static_va", "status": "FULLY_RECONCILED" },
  "ref_id": "REF1755762993VPURY"
}`

const rekonsiliasiSebagian = `{
  "additional_info": { "invoices": [ { "number": "INV/2025/11/0012" } ] },
  "reconciled_amount": 50000,
  "payment_info": { "channel": "Permata", "method": "static_va",
    "static_va": { "amount": 200000 }, "status": "PARTIALLY_RECONCILED" },
  "ref_id": "REF-SEBAGIAN"
}`

func parse(t *testing.T, raw string) WebhookInput {
	t.Helper()
	var in WebhookInput
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return in
}

func TestPayloadAsliTerbacaBenar(t *testing.T) {
	kasus := []struct {
		nama      string
		raw       string
		mauStatus string
		mauNomor  string
		mauAmount float64
		mauMetode string
	}{
		{"bank transfer", bankTransferAsli, "PAID", "INV/2025/11/0001", 12557, "bank_transfer"},
		{"qris", qrisAsli, "PAID", "INV/2025/11/0002", 10000, "qris"},
		{"kartu kredit", creditCardAsli, "PAID", "INV/2025/11/0003", 10000, "credit_card"},
		{"e-wallet", ewalletAsli, "PAID", "INV/2025/11/0004", 10000, "ewallet"},
		{"dashboard tanpa status datar", dashboardTanpaStatusDatar, "PAID", "INV/2025/11/0005", 200000, "bank_transfer"},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			in := parse(t, k.raw)
			if got := in.settlementStatus(); got != k.mauStatus {
				t.Errorf("status = %q, mau %q", got, k.mauStatus)
			}
			_, number := in.invoiceRef()
			if number != k.mauNomor {
				t.Errorf("nomor invoice = %q, mau %q", number, k.mauNomor)
			}
			pay := summarizePayment(in.PaymentInfo)
			if pay.Detail.PaidAmount != k.mauAmount {
				t.Errorf("paid_amount = %v, mau %v", pay.Detail.PaidAmount, k.mauAmount)
			}
			if pay.Method != k.mauMetode {
				t.Errorf("metode = %q, mau %q", pay.Method, k.mauMetode)
			}
			if pay.Detail.PaidAt == "" && k.nama != "dashboard tanpa status datar" {
				t.Error("paid_at kosong")
			}
		})
	}
}

// Invoice Callback bentuknya sama sekali berbeda: tidak ada payment_info sama
// sekali, dan identitas invoicenya di data.invoice.
func TestInvoiceCallbackTerbaca(t *testing.T) {
	in := parse(t, invoiceCallbackAsli)
	if got := in.settlementStatus(); got != "PAID" {
		t.Errorf("status = %q, mau PAID", got)
	}
	uuid, number := in.invoiceRef()
	if uuid != "afef0ea1-caa6-4123-9735-749df749642c" || number != "INV/2025/06/126" {
		t.Errorf("identitas invoice salah: %q / %q", uuid, number)
	}
	if in.Data.Invoice.TotalAmount != 10000 {
		t.Errorf("total_amount = %v", in.Data.Invoice.TotalAmount)
	}
}

// FULLY_RECONCILED melunasi; PARTIALLY_RECONCILED TIDAK — tagihannya belum
// lunas, dan menandainya lunas akan menghentikan penagihan atas sisa yang
// masih terutang.
func TestRekonsiliasiHanyaMelunasiBilaPenuh(t *testing.T) {
	penuh := parse(t, rekonsiliasiPenuh)
	if got := penuh.settlementStatus(); got != "FULLY_RECONCILED" {
		t.Errorf("status = %q, mau FULLY_RECONCILED", got)
	}
	if _, number := penuh.invoiceRef(); number != "test/2025/07/0049" {
		t.Errorf("nomor invoice = %q", number)
	}
	if penuh.ReconciledAmount != 123123 {
		t.Errorf("reconciled_amount = %v", penuh.ReconciledAmount)
	}

	sebagian := parse(t, rekonsiliasiSebagian)
	if got := sebagian.settlementStatus(); got != "PARTIALLY_RECONCILED" {
		t.Errorf("status = %q, mau PARTIALLY_RECONCILED", got)
	}
}

// Inspeksi format tidak boleh mengeluh atas payload yang SAH — kalau ia
// menandai bentuk resmi sebagai asing, catatannya jadi bising dan orang
// berhenti membacanya tepat saat catatan itu penting.
func TestInspeksiTidakMengeluhAtasPayloadAsli(t *testing.T) {
	for nama, raw := range map[string]string{
		"bank transfer": bankTransferAsli, "qris": qrisAsli,
		"invoice callback": invoiceCallbackAsli, "rekonsiliasi": rekonsiliasiPenuh,
	} {
		if notes := inspectPayload([]byte(raw)); len(notes) > 0 {
			t.Errorf("%s: payload sah tidak boleh menghasilkan catatan, dapat %v", nama, notes)
		}
	}
}
