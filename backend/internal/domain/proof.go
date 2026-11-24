package domain

import (
	"fmt"
	"strings"
	"time"
)

// Bukti bayar: langkah 5 alur renewal klien. ST mengirim bukti transfer
// member ke finance, dan finance yang memutuskan invoice itu lunas.
//
// Bukti belum menjadi pembayaran. Baris di tabel payments baru tercatat saat
// finance menyetujuinya, supaya laporan tidak memuat uang yang belum diperiksa.

type ProofStatus string

const (
	ProofPending  ProofStatus = "pending"
	ProofApproved ProofStatus = "approved"
	ProofRejected ProofStatus = "rejected"
)

func (s ProofStatus) Valid() bool {
	switch s {
	case ProofPending, ProofApproved, ProofRejected:
		return true
	}
	return false
}

type PaymentProof struct {
	ID            string      `json:"id"`
	InvoiceID     string      `json:"invoiceId"`
	ChapterID     string      `json:"chapterId"`
	Amount        int64       `json:"amount"`
	PaidAt        Date        `json:"paidAt"`
	PaymentMethod *string     `json:"paymentMethod,omitempty"`
	ProofURL      string      `json:"proofUrl"`
	Note          *string     `json:"note,omitempty"`
	Status        ProofStatus `json:"status"`
	SubmittedBy   *string     `json:"submittedBy,omitempty"`
	SubmittedAt   time.Time   `json:"submittedAt"`
	ReviewedBy    *string     `json:"reviewedBy,omitempty"`
	ReviewedAt    *time.Time  `json:"reviewedAt,omitempty"`
	ReviewNote    *string     `json:"reviewNote,omitempty"`
	PaymentID     *string     `json:"paymentId,omitempty"`

	// Diisi lewat join, supaya daftar verifikasi tidak perlu satu permintaan
	// tambahan per baris.
	InvoiceNumber string        `json:"invoiceNumber"`
	InvoiceAmount int64         `json:"invoiceAmount"`
	InvoiceStatus InvoiceStatus `json:"invoiceStatus"`
	MemberName    *string       `json:"memberName,omitempty"`
	ChapterName   *string       `json:"chapterName,omitempty"`
}

type SubmitProofInput struct {
	InvoiceID     string  `json:"invoiceId"`
	Amount        int64   `json:"amount"`
	PaidAt        Date    `json:"paidAt"`
	PaymentMethod *string `json:"paymentMethod"`
	ProofURL      string  `json:"proofUrl"`
	Note          *string `json:"note"`
}

func (in SubmitProofInput) Validate(sekarang time.Time) error {
	switch {
	case in.InvoiceID == "":
		return fmt.Errorf("invoiceId wajib diisi")
	case in.Amount <= 0:
		return fmt.Errorf("amount harus lebih besar dari 0")
	case in.Amount > MaxInvoiceAmount:
		return fmt.Errorf("amount di luar jangkauan")
	case in.PaidAt.IsZero():
		return fmt.Errorf("paidAt wajib diisi")
	case in.PaidAt.After(sekarang):
		return fmt.Errorf("paidAt tidak boleh di masa depan")
	case strings.TrimSpace(in.ProofURL) == "":
		return fmt.Errorf("proofUrl wajib diisi")
	}
	return nil
}

// ReviewProofInput adalah body setuju dan tolak. Catatan wajib saat menolak,
// karena ST perlu tahu apa yang harus diperbaiki sebelum mengirim ulang.
type ReviewProofInput struct {
	Note *string `json:"note"`
}

type ProofFilter struct {
	Status    string
	InvoiceID string
	ChapterID string
	Limit     int
	Offset    int
}
