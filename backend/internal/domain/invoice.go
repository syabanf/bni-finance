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
)

func (s InvoiceStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusSent, StatusPaid, StatusOverdue, StatusCancelled:
		return true
	}
	return false
}

// allowedTransitions mirrors the lifecycle the UI enforces: draft → sent →
// paid, cancellable until paid. Paid and cancelled are terminal.
var allowedTransitions = map[InvoiceStatus][]InvoiceStatus{
	StatusDraft:     {StatusSent, StatusCancelled},
	StatusSent:      {StatusPaid, StatusOverdue, StatusCancelled},
	StatusOverdue:   {StatusPaid, StatusCancelled},
	StatusPaid:      {},
	StatusCancelled: {},
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

	// Who made the change. Not stored on the invoice — these only label the
	// audit-log row written alongside the update.
	ActorID   *string `json:"actorId"`
	ActorName *string `json:"actorName"`
}

func (in UpdateInvoiceInput) Validate() error {
	if in.Amount != nil && *in.Amount <= 0 {
		return fmt.Errorf("amount harus lebih besar dari 0")
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
