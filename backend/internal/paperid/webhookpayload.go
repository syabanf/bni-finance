package paperid

import (
	"encoding/json"
	"strings"
)

// Bentuk payload callback Paper.id yang SEBENARNYA, disalin dari dokumentasi
// resmi mereka DAN dari contoh di dashboard. Keduanya perlu, karena keduanya
// tidak sama.
//
// Versi pertama ditulis dari tebakan dan tidak pernah bisa melunasi apa pun: ia
// membaca payment_info.status dan payment_info.amount sebagai field datar,
// padahal keduanya bersarang di dalam objek yang DINAMAI MENURUT METODE
// pembayaran — bank_transfer, qris, credit_card, ewallet, va_facilitator.
//
// Versi kedua memperbaiki itu, tapi masih menganggap letak setiap bagian tetap.
// Contoh di dashboard membuktikan tidak:
//
//	Static VA        payment_info ada DI DALAM data, bukan di akar
//	Invoice Dibayar  invoice ada DI AKAR, bukan di data.invoice
//
// Jadi letaknya tidak bisa diandalkan — yang bisa hanya namanya. Setiap
// pengambilan di berkas ini karena itu mencari di kedua tempat. Kalau tidak,
// kegagalannya SUNYI: callback dijawab 200, tidak ada invoice yang dilunasi,
// dan tidak ada yang merah.
//
// Tujuh keluarga callback, satu struct dengan field opsional — satu endpoint
// harus bisa menerima bentuk mana pun tanpa menebak lebih dulu.

// invoiceBody adalah objek invoice, di mana pun ia muncul.
//
// Dashboard memakai `amount`, dokumentasi memakai `total_amount`, untuk hal
// yang sama. Keduanya dibaca; totalAmount() memutuskan mana yang terisi.
type invoiceBody struct {
	ID          string  `json:"id"`
	Number      string  `json:"number"`
	PartnerID   string  `json:"partner_id"`
	Status      string  `json:"status"`
	Amount      float64 `json:"amount"`
	AmountDue   float64 `json:"amount_due"`
	TotalAmount float64 `json:"total_amount"`
	UpdatedAt   string  `json:"updated_at"`
}

func (i invoiceBody) totalAmount() float64 {
	if i.TotalAmount != 0 {
		return i.TotalAmount
	}
	return i.Amount
}

func (i invoiceBody) kosong() bool { return i.ID == "" && i.Number == "" }

// WebhookInput adalah payload callback apa adanya.
type WebhookInput struct {
	RefID       string `json:"ref_id"`
	ExternalID  string `json:"external_id"`
	Message     string `json:"message"`
	PaymentDate string `json:"payment_date"`

	// PaymentInfo dibiarkan mentah karena kuncinya dinamis: selain channel,
	// method, dan status, ada satu objek bernama sesuai metode pembayaran.
	PaymentInfo json.RawMessage `json:"payment_info"`

	// Invoice di akar — bentuk "Invoice Dibayar" pada dashboard.
	Invoice invoiceBody `json:"invoice"`

	AdditionalInfo struct {
		Invoices []struct {
			UUID   string `json:"uuid"`
			Number string `json:"number"`
		} `json:"invoices"`
	} `json:"additional_info"`

	// Paylater Diajukan.
	TransactionID     string `json:"transaction_id"`
	TransactionDate   string `json:"transaction_date"`
	TransactionStatus string `json:"transaction_status"`

	// Pembayaran ke Supplier — seluruh detailnya datar di akar.
	DisbursementID string `json:"disbursement_id"`
	PaymentID      string `json:"payment_id"`
	Status         string `json:"status"`

	// Data terisi pada Invoice, Sisa Tagihan, Static VA, dan Disbursement.
	Data struct {
		Invoice     invoiceBody     `json:"invoice"`
		PaymentInfo json.RawMessage `json:"payment_info"`
		Message     string          `json:"message"`
		PaymentDate string          `json:"payment_date"`
		Partner     struct {
			ID     string `json:"id"`
			Number string `json:"number"`
		} `json:"partner"`
		Payment struct {
			ExternalID       string  `json:"external_id"`
			ParentExternalID string  `json:"parent_external_id"`
			ReconciledAmount float64 `json:"reconciled_amount"`
		} `json:"payment"`
		// Disbursement Pay In.
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
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
	"source": true, "event": true, "payment_type": true, "additional_info": true,
}

// paymentInfo mengembalikan payment_info dari tempat mana pun ia berada.
//
// Static VA membungkusnya di dalam `data`; keluarga lain menaruhnya di akar.
// Membaca satu tempat saja membuat seluruh callback Static VA — yang justru
// paling sering dipakai untuk pelunasan otomatis — terbaca kosong.
func (in WebhookInput) paymentInfo() json.RawMessage {
	if len(in.PaymentInfo) > 0 && string(in.PaymentInfo) != "null" {
		return in.PaymentInfo
	}
	return in.Data.PaymentInfo
}

// invoice mengembalikan objek invoice dari tempat mana pun ia berada.
func (in WebhookInput) invoice() invoiceBody {
	if !in.Data.Invoice.kosong() {
		return in.Data.Invoice
	}
	return in.Invoice
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
	// tanpa perlu menambahkan namanya ke daftar. Static VA membuktikan ini
	// perlu: method-nya "static_va" tapi objeknya bernama "va_facilitator".
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
// Satu tempat yang memutuskan, supaya penambahan bentuk baru tidak menyebar ke
// seluruh berkas.
func (in WebhookInput) invoiceRef() (uuid, number string) {
	if inv := in.invoice(); !inv.kosong() {
		return inv.ID, inv.Number
	}
	if len(in.AdditionalInfo.Invoices) > 0 {
		return in.AdditionalInfo.Invoices[0].UUID, in.AdditionalInfo.Invoices[0].Number
	}
	return "", ""
}

// settlementStatus mengembalikan status yang menentukan pelunasan, dari bentuk
// callback mana pun.
func (in WebhookInput) settlementStatus() string {
	if s := in.invoice().Status; s != "" {
		return strings.ToUpper(strings.TrimSpace(s))
	}
	if st := summarizePayment(in.paymentInfo()).Status; st != "" {
		return strings.ToUpper(strings.TrimSpace(st))
	}
	// Paylater Diajukan tidak punya status di payment_info sama sekali —
	// statusnya ada di transaction_status, dan "requested" memang bukan
	// pembayaran. Dibaca supaya terekam apa adanya, bukan supaya melunasi.
	if in.TransactionStatus != "" {
		return strings.ToUpper(strings.TrimSpace(in.TransactionStatus))
	}
	// Pembayaran ke Supplier menaruh statusnya datar di akar.
	return strings.ToUpper(strings.TrimSpace(in.Status))
}

// punyaSasaranLain melaporkan payload membawa identitas yang sah meski bukan
// identitas invoice.
//
// Static VA adalah alasannya ada: uang masuk ke virtual account milik SEBUAH
// PARTNER, bukan ke sebuah invoice. Pencocokan ke invoice baru datang belakangan
// lewat callback rekonsiliasi (source: recon_static_va). Payload seperti itu sah
// sepenuhnya — menjawabnya 400 membuat Paper.id menganggap callback gagal dan
// mengirim ulang tanpa henti, selamanya, atas bentuk yang memang benar.
func (in WebhookInput) punyaSasaranLain() bool {
	return in.Data.Partner.ID != "" || in.Data.Partner.Number != "" ||
		in.RefID != "" || in.ExternalID != "" || in.Data.Payment.ExternalID != ""
}
