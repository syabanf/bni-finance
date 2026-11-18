package domain

import (
	"fmt"
	"time"
)

// Payment mirrors the `payments` table (including proof_url/note added by
// migration 0003 for manually recorded payments).
type Payment struct {
	ID        string    `json:"id"`
	InvoiceID string    `json:"invoiceId"`
	Amount    int64     `json:"amount"`
	PaidAt    time.Time `json:"paidAt"`

	PaymentMethod *string `json:"paymentMethod,omitempty"`

	PaperIDPaymentID *string `json:"paperIdPaymentId,omitempty"`
	PaperIDStatus    *string `json:"paperIdStatus,omitempty"`
	XenditPaymentID  *string `json:"xenditPaymentId,omitempty"`
	XenditStatus     *string `json:"xenditStatus,omitempty"`

	ProofURL *string `json:"proofUrl,omitempty"`
	Note     *string `json:"note,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// CreatePaymentInput is the POST body. Recording a payment also settles the
// invoice (see payment.Service.Create) — that's the domain rule the UI relies on.
type CreatePaymentInput struct {
	InvoiceID     string     `json:"invoiceId"`
	Amount        int64      `json:"amount"`
	PaidAt        *time.Time `json:"paidAt"`
	PaymentMethod *string    `json:"paymentMethod"`
	ProofURL      *string    `json:"proofUrl"`
	Note          *string    `json:"note"`
	/** Set false to record the payment without settling the invoice. */
	SettleInvoice *bool `json:"settleInvoice"`
}

func (in CreatePaymentInput) Validate() error {
	switch {
	case in.InvoiceID == "":
		return fmt.Errorf("invoiceId wajib diisi")
	case in.Amount <= 0:
		return fmt.Errorf("amount harus lebih besar dari 0")
	}
	return nil
}

// ShouldSettle defaults to true — recording a payment normally marks the
// invoice paid.
func (in CreatePaymentInput) ShouldSettle() bool {
	return in.SettleInvoice == nil || *in.SettleInvoice
}

type UpdatePaymentInput struct {
	Amount        *int64     `json:"amount"`
	PaidAt        *time.Time `json:"paidAt"`
	PaymentMethod *string    `json:"paymentMethod"`
	ProofURL      *string    `json:"proofUrl"`
	Note          *string    `json:"note"`
}

func (in UpdatePaymentInput) Validate() error {
	if in.Amount != nil && *in.Amount <= 0 {
		return fmt.Errorf("amount harus lebih besar dari 0")
	}
	return nil
}

type PaymentFilter struct {
	InvoiceID string
	Method    string
	PaidFrom  string
	PaidTo    string
	// Search mencocokkan nomor invoice atau nama member pembayarnya.
	Search string
	// ChapterID dan Type menyaring lewat invoice induknya: payments tidak
	// menyimpan chapter maupun tipe sendiri.
	ChapterID string
	Type      string
	Limit     int
	Offset    int
}

// PaymentSummary adalah agregat untuk SELURUH hasil filter, bukan halaman yang
// sedang tampil. Alasannya sama dengan InvoiceSummary: kartu ringkasan yang
// dihitung dari satu halaman menampilkan angka uang yang terlalu kecil tanpa
// tanda apa pun.
type PaymentSummary struct {
	Count  int   `json:"count"`
	Amount int64 `json:"amount"`
	// ByType dijumlahkan menurut tipe invoice induknya.
	ByType map[string]int64 `json:"byType"`
	// Methods adalah metode yang muncul di hasil filter TANPA saringan
	// metodenya, untuk mengisi pilihan saringan itu.
	Methods []string `json:"methods"`
}
