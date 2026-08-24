package domain

import (
	"fmt"
	"time"
)

type InvoiceType string

const (
	TypeRegistration InvoiceType = "registration"
	TypeRenewal      InvoiceType = "renewal"
)

func (t InvoiceType) Valid() bool {
	return t == TypeRegistration || t == TypeRenewal
}

type InvoiceStatus string

const (
	StatusDraft     InvoiceStatus = "draft"
	StatusSent      InvoiceStatus = "sent"
	StatusPaid      InvoiceStatus = "paid"
	StatusOverdue   InvoiceStatus = "overdue"
	StatusCancelled InvoiceStatus = "cancelled"
	// StatusTerminated — pembayarannya dibatalkan dan keanggotaannya diputus.
	//
	// Berbeda dari StatusCancelled, dan bedanya bukan kosmetik: `cancelled`
	// adalah pembatalan biasa — salah terbit, member menunda, tagihan ditarik
	// kembali. `terminated` menandai tagihan yang gugur karena hubungannya
	// berakhir. Menyatukan keduanya membuat laporan tidak bisa lagi membedakan
	// tagihan yang batal dari keanggotaan yang putus.
	StatusTerminated InvoiceStatus = "terminated"
)

func (s InvoiceStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusSent, StatusPaid, StatusOverdue, StatusCancelled, StatusTerminated:
		return true
	}
	return false
}

// allowedTransitions mirrors the lifecycle the UI enforces: draft → sent →
// paid, cancellable until paid. Paid and cancelled are terminal.
// StatusTerminated bisa dicapai dari mana pun kecuali `paid`, dan sifatnya
// terminal. Tidak dari `paid` karena uangnya sudah masuk: memutus keanggotaan
// tidak membatalkan pembayaran yang sudah diterima, dan menandai tagihan lunas
// sebagai terminated akan menghilangkannya dari laporan pendapatan.
var allowedTransitions = map[InvoiceStatus][]InvoiceStatus{
	StatusDraft:      {StatusSent, StatusCancelled, StatusTerminated},
	StatusSent:       {StatusPaid, StatusOverdue, StatusCancelled, StatusTerminated},
	StatusOverdue:    {StatusPaid, StatusCancelled, StatusTerminated},
	StatusCancelled:  {StatusTerminated},
	StatusPaid:       {},
	StatusTerminated: {},
}

// CanTransitionTo reports whether a status change is legal. Staying on the same
// status is always allowed (a no-op update).
func (s InvoiceStatus) CanTransitionTo(next InvoiceStatus) bool {
	if s == next {
		return true
	}
	for _, allowed := range allowedTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

// Invoice mirrors the `invoices` table. JSON uses camelCase so the existing
// TypeScript client can consume it without a mapping layer.
type Invoice struct {
	ID        string      `json:"id"`
	Number    string      `json:"number"`
	MemberID  string      `json:"memberId"`
	ChapterID string      `json:"chapterId"`
	Type      InvoiceType `json:"type"`
	Amount    int64       `json:"amount"`
	Currency  string      `json:"currency"`

	DueDate     Date `json:"dueDate"`
	PeriodStart Date `json:"periodStart"`
	PeriodEnd   Date `json:"periodEnd"`

	Status InvoiceStatus `json:"status"`

	// Paper.id
	PaperIDInvoiceID  *string    `json:"paperIdInvoiceId,omitempty"`
	PaperIDInvoiceURL *string    `json:"paperIdInvoiceUrl,omitempty"`
	PaperIDPaymentURL *string    `json:"paperIdPaymentUrl,omitempty"`
	PaperIDSentAt     *time.Time `json:"paperIdSentAt,omitempty"`

	// PaperIDReminderCount = berapa kali invoice ini dikirim ulang sebagai
	// pengingat. Menentukan sufiks nomor berikutnya, karena Paper.id membakar
	// nomor secara permanen dan menolak pengiriman kedua dengan nomor sama.
	PaperIDReminderCount int `json:"paperIdReminderCount"`

	// Xendit (self-payment)
	PaymentProvider     *string    `json:"paymentProvider,omitempty"`
	XenditExternalID    *string    `json:"xenditExternalId,omitempty"`
	XenditPaymentID     *string    `json:"xenditPaymentId,omitempty"`
	XenditPaymentMethod *string    `json:"xenditPaymentMethod,omitempty"`
	XenditVaBank        *string    `json:"xenditVaBank,omitempty"`
	XenditVaNumber      *string    `json:"xenditVaNumber,omitempty"`
	XenditQrisString    *string    `json:"xenditQrisString,omitempty"`
	XenditPaymentStatus *string    `json:"xenditPaymentStatus,omitempty"`
	XenditExpiresAt     *time.Time `json:"xenditExpiresAt,omitempty"`

	PaidAt     *time.Time `json:"paidAt,omitempty"`
	PaidAmount *int64     `json:"paidAmount,omitempty"`

	Notes        *string    `json:"notes,omitempty"`
	CreatedBy    *string    `json:"createdBy,omitempty"`
	CancelledBy  *string    `json:"cancelledBy,omitempty"`
	CancelledAt  *time.Time `json:"cancelledAt,omitempty"`
	CancelReason *string    `json:"cancelReason,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// LateFee DIHITUNG saat dibaca, tidak pernah tersimpan. Nil bila fiturnya
	// dimatikan atau tagihannya belum telat — lihat latefee.go.
	//
	// omitempty pada pointer: klien lama yang tidak mengenal field ini tidak
	// melihat perubahan apa pun, dan klien baru bisa membedakan "belum telat"
	// dari "fitur mati" lewat pengaturannya, bukan lewat nilai nol yang ambigu.
	LateFee *LateFee `json:"lateFee,omitempty"`
}

// CreateInvoiceInput is the POST body. Number is generated server-side when
// omitted so callers can't create duplicates by accident.
type CreateInvoiceInput struct {
	Number      *string     `json:"number"`
	MemberID    string      `json:"memberId"`
	ChapterID   string      `json:"chapterId"`
	Type        InvoiceType `json:"type"`
	Amount      int64       `json:"amount"`
	Currency    *string     `json:"currency"`
	DueDate     Date        `json:"dueDate"`
	PeriodStart Date        `json:"periodStart"`
	PeriodEnd   Date        `json:"periodEnd"`
	Notes       *string     `json:"notes"`
	CreatedBy   *string     `json:"createdBy"`
}

// MaxInvoiceAmount adalah pagar salah ketik, BUKAN aturan bisnis.
//
// Nilainya seratus miliar rupiah — sekitar 60.000 kali iuran yang sebenarnya,
// jadi tidak mungkin menghalangi invoice yang sah. Yang dihalangi adalah angka
// yang jelas kecelakaan: sebelum kolomnya dilebarkan ke bigint, nominal di atas
// 2,1 miliar dijawab 500 "terjadi kesalahan pada server"; sesudah dilebarkan,
// 9.223.372.036.854.775.807 rupiah diterima dengan 201 tanpa keluhan apa pun.
// Keduanya salah, dan yang kedua lebih berbahaya karena tersimpan.
//
// Ubah bila memang ada tagihan yang melampauinya.
const MaxInvoiceAmount int64 = 100_000_000_000

func (in CreateInvoiceInput) Validate() error {
	switch {
	case in.MemberID == "":
		return fmt.Errorf("memberId wajib diisi")
	case in.ChapterID == "":
		return fmt.Errorf("chapterId wajib diisi")
	case !in.Type.Valid():
		return fmt.Errorf("type harus 'registration' atau 'renewal'")
	case in.Amount <= 0:
		return fmt.Errorf("amount harus lebih besar dari 0")
	case in.Amount > MaxInvoiceAmount:
		return fmt.Errorf("amount %d melampaui batas wajar (%d) — periksa jumlah nolnya",
			in.Amount, MaxInvoiceAmount)
	case in.DueDate.IsZero():
		return fmt.Errorf("dueDate wajib diisi")
	case in.PeriodStart.IsZero() || in.PeriodEnd.IsZero():
		return fmt.Errorf("periodStart dan periodEnd wajib diisi")
	case in.PeriodEnd.Before(in.PeriodStart.Time):
		return fmt.Errorf("periodEnd tidak boleh lebih awal dari periodStart")
	}
	return nil
}

// UpdateInvoiceInput is a PATCH body — every field is optional; only the ones
// present are written.
type UpdateInvoiceInput struct {
	Amount       *int64         `json:"amount"`
	DueDate      *Date          `json:"dueDate"`
	PeriodStart  *Date          `json:"periodStart"`
	PeriodEnd    *Date          `json:"periodEnd"`
	Status       *InvoiceStatus `json:"status"`
	Notes        *string        `json:"notes"`
	CancelReason *string        `json:"cancelReason"`
	CancelledBy  *string        `json:"cancelledBy"`

	// Paper.id link, written when an invoice is issued through that provider.
	PaperIDInvoiceID  *string    `json:"paperIdInvoiceId"`
	PaperIDInvoiceURL *string    `json:"paperIdInvoiceUrl"`
	PaperIDPaymentURL *string    `json:"paperIdPaymentUrl"`
	PaperIDSentAt     *time.Time `json:"paperIdSentAt"`

	// Who made the change. Not stored on the invoice — these only label the
	// audit-log row written alongside the update.
	ActorID   *string `json:"actorId"`
	ActorName *string `json:"actorName"`
}

func (in UpdateInvoiceInput) Validate() error {
	if in.Amount != nil && *in.Amount <= 0 {
		return fmt.Errorf("amount harus lebih besar dari 0")
	}
	// Pagar yang sama seperti pada pembuatan — PATCH bisa menaruh nominal
	// mustahil ke invoice yang sudah ada persis seperti POST bisa.
	if in.Amount != nil && *in.Amount > MaxInvoiceAmount {
		return fmt.Errorf("amount %d melampaui batas wajar (%d) — periksa jumlah nolnya",
			*in.Amount, MaxInvoiceAmount)
	}
	if in.Status != nil && !in.Status.Valid() {
		return fmt.Errorf("status tidak dikenal")
	}
	if in.PeriodStart != nil && in.PeriodEnd != nil && in.PeriodEnd.Before(in.PeriodStart.Time) {
		return fmt.Errorf("periodEnd tidak boleh lebih awal dari periodStart")
	}
	return nil
}

// InvoiceFilter drives GET /invoices.
type InvoiceFilter struct {
	Status     string
	Type       string
	ChapterID  string
	MemberID   string
	Search     string
	DueFrom    string
	DueTo      string
	IssuedFrom string
	IssuedTo   string
	Limit      int
	Offset     int
}

// ValidateUpdateFrom memeriksa apakah patch boleh diterapkan pada invoice yang
// sedang berstatus current.
//
// Dipanggil DUA kali dengan sengaja: sekali di service supaya permintaan yang
// jelas salah ditolak cepat dengan pesan yang jelas, dan sekali lagi di dalam
// transaksi repository setelah barisnya dikunci. Yang kedua itu yang mengikat.
//
// Sebelumnya hanya ada pemeriksaan pertama, dan itu memvalidasi status yang
// dibaca SEBELUM transaksi dimulai. Dua permintaan bersamaan pada invoice
// `sent` — satu melunasi, satu membatalkan — sama-sama lolos, dan status
// akhirnya sekadar siapa yang menulis belakangan: invoice lunas bisa berakhir
// dibatalkan. Terukur: 11 dari 12 ronde.
func (s InvoiceStatus) ValidateUpdateFrom(in UpdateInvoiceInput) error {
	if in.Status != nil && !s.CanTransitionTo(*in.Status) {
		return fmt.Errorf("transisi status tidak diizinkan: %s → %s", s, *in.Status)
	}
	// Invoice lunas atau batal adalah catatan tertutup — nominal dan periodenya
	// tidak boleh ditulis ulang setelah fakta.
	if s == StatusPaid || s == StatusCancelled {
		if in.Amount != nil || in.DueDate != nil || in.PeriodStart != nil || in.PeriodEnd != nil {
			return fmt.Errorf("invoice berstatus %s tidak bisa diubah nominal/periodenya", s)
		}
	}
	return nil
}
