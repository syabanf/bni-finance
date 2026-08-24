package paperid

import (
	"encoding/json"
	"strings"
)

// Bentuk payload callback Paper.id yang SEBENARNYA, disalin dari dokumentasi
// resmi mereka dan diverifikasi terhadap contoh di dashboard.
//
// Versi sebelumnya ditulis dari tebakan dan tidak pernah bisa melunasi apa pun:
// ia membaca payment_info.status dan payment_info.amount sebagai field datar,
// padahal keduanya bersarang di dalam objek yang DINAMAI MENURUT METODE
// pembayaran — bank_transfer, qris, credit_card, ewallet, static_va. Diuji
// terhadap payload asli, ketiganya terbaca kosong.
//
// Ada tiga keluarga callback, dan bentuknya berbeda-beda:
//
//	Payment      additional_info.invoices[] + payment_info.[metode]
//	             dipakai untuk Payment In dan Payment Out — payloadnya sama,
//	             hanya URL pendaftarannya yang membedakan
//	Invoice      data.invoice — hanya dikirim saat status invoice menjadi PAID
//	Reconciliation  additional_info.invoices[] + reconciled_amount
//
// Karena itu satu struct dengan field opsional, bukan tiga struct terpisah:
// satu endpoint harus bisa menerima bentuk mana pun tanpa menebak lebih dulu.

// WebhookInput adalah payload callback apa adanya.
type WebhookInput struct {
	RefID       string `json:"ref_id"`
	ExternalID  string `json:"external_id"`
	Message     string `json:"message"`
	PaymentDate string `json:"payment_date"`

	// PaymentInfo dibiarkan mentah karena kuncinya dinamis: selain channel,
	// method, dan status, ada satu objek bernama sesuai metode pembayaran.
	PaymentInfo json.RawMessage `json:"payment_info"`

	AdditionalInfo struct {
		Invoices []struct {
			UUID   string `json:"uuid"`
			Number string `json:"number"`
		} `json:"invoices"`
	} `json:"additional_info"`

	// Data hanya terisi pada Invoice Callback.
	Data struct {
		Invoice struct {
			ID          string  `json:"id"`
			Number      string  `json:"number"`
			PartnerID   string  `json:"partner_id"`
			Status      string  `json:"status"`
			AmountDue   float64 `json:"amount_due"`
			TotalAmount float64 `json:"total_amount"`
			UpdatedAt   string  `json:"updated_at"`
		} `json:"invoice"`
	} `json:"data"`

	// Reconciliation Callback.
	ReconciledAmount   float64 `json:"reconciled_amount"`
	ReconciliationDate string  `json:"reconciliation_date"`
	Source             string  `json:"source"`
}

// paymentDetail adalah isi objek payment_info.[metode].
type paymentDetail struct {
	Amount     float64 `json:"amount"`
	PaidAmount float64 `json:"paid_amount"`
	PaidAt     string  `json:"paid_at"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"created_at"`
	Created    string  `json:"created"`
	Updated    string  `json:"updated"`
}

// paymentSummary merangkum payment_info menjadi bentuk datar yang bisa dipakai.
type paymentSummary struct {
	Channel string
	Method  string
	Status  string
	Detail  paymentDetail
}

// kunci yang BUKAN objek metode pembayaran.
var bukanMetode = map[string]bool{
	"channel": true, "method": true, "status": true, "message": true,
	"source": true, "event": true, "payment_type": true,
}

// summarizePayment membaca payment_info yang bentuknya dinamis.
//
// Status dicari dua tempat, dan urutannya penting: payment_info.status lebih
// dulu karena payload terbaru menyertakannya, lalu objek metodenya sebagai
// cadangan — contoh di dashboard Paper.id hanya punya yang bersarang. Membaca
// satu tempat saja membuat separuh payload yang sah terbaca kosong.
func summarizePayment(raw json.RawMessage) paymentSummary {
	var out paymentSummary
	if len(raw) == 0 {
		return out
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	unstring(m["channel"], &out.Channel)
	unstring(m["method"], &out.Method)
	unstring(m["status"], &out.Status)

	// Objek metode: dicoba lewat nama di field method lebih dulu, lalu kunci
	// apa pun yang tersisa — supaya metode baru dari Paper.id tetap terbaca
	// tanpa perlu menambahkan namanya ke daftar.
	if out.Method != "" {
		if v, ok := m[out.Method]; ok && json.Unmarshal(v, &out.Detail) == nil {
			if out.Status == "" {
				out.Status = out.Detail.Status
			}
			return out
		}
	}
	for k, v := range m {
		if bukanMetode[k] {
			continue
		}
		var d paymentDetail
		if json.Unmarshal(v, &d) == nil && (d.Status != "" || d.Amount != 0 || d.PaidAmount != 0) {
			out.Detail = d
			if out.Status == "" {
				out.Status = d.Status
			}
			return out
		}
	}
	return out
}

func unstring(raw json.RawMessage, dst *string) {
	if len(raw) == 0 {
		return
	}
	_ = json.Unmarshal(raw, dst)
}

// invoiceRef mengembalikan identitas invoice dari bentuk callback mana pun.
//
// Payment dan Reconciliation memakai additional_info.invoices; Invoice Callback
// memakai data.invoice. Satu tempat yang memutuskan, supaya penambahan bentuk
// baru tidak menyebar ke seluruh berkas.
func (in WebhookInput) invoiceRef() (uuid, number string) {
	if in.Data.Invoice.ID != "" || in.Data.Invoice.Number != "" {
		return in.Data.Invoice.ID, in.Data.Invoice.Number
	}
	if len(in.AdditionalInfo.Invoices) > 0 {
		return in.AdditionalInfo.Invoices[0].UUID, in.AdditionalInfo.Invoices[0].Number
	}
	return "", ""
}

// settlementStatus mengembalikan status yang menentukan pelunasan, dari bentuk
// callback mana pun.
func (in WebhookInput) settlementStatus() string {
	if s := in.Data.Invoice.Status; s != "" {
		return strings.ToUpper(strings.TrimSpace(s))
	}
	return strings.ToUpper(strings.TrimSpace(summarizePayment(in.PaymentInfo).Status))
}
