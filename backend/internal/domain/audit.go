package domain

import (
	"fmt"
	"time"
)

type AuditAction string

const (
	AuditCreated   AuditAction = "created"
	AuditSent      AuditAction = "sent"
	AuditPaid      AuditAction = "paid"
	AuditCancelled AuditAction = "cancelled"
	AuditOverdue   AuditAction = "overdue"
	AuditUpdated   AuditAction = "updated"
)

func (a AuditAction) Valid() bool {
	switch a {
	case AuditCreated, AuditSent, AuditPaid, AuditCancelled, AuditOverdue, AuditUpdated:
		return true
	}
	return false
}

// ActionForStatus maps a status change onto the audit action that describes it.
func ActionForStatus(s InvoiceStatus) AuditAction {
	switch s {
	case StatusSent:
		return AuditSent
	case StatusPaid:
		return AuditPaid
	case StatusCancelled:
		return AuditCancelled
	case StatusOverdue:
		return AuditOverdue
	default:
		return AuditUpdated
	}
}

// AuditEntry is one row of `invoice_audit_log` — the timeline the invoice
// detail page renders.
type AuditEntry struct {
	ID        string         `json:"id"`
	InvoiceID string         `json:"invoiceId"`
	Action    AuditAction    `json:"action"`
	OldStatus *InvoiceStatus `json:"oldStatus,omitempty"`
	NewStatus *InvoiceStatus `json:"newStatus,omitempty"`
	ActorID   *string        `json:"actorId,omitempty"`
	ActorName *string        `json:"actorName,omitempty"`
	Notes     *string        `json:"notes,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// CreateAuditEntryInput adds a manual note to an invoice timeline. Status
// changes are recorded automatically by the invoice repository, so this is only
// for human annotations.
type CreateAuditEntryInput struct {
	Action    *AuditAction `json:"action"`
	ActorID   *string      `json:"actorId"`
	ActorName *string      `json:"actorName"`
	Notes     *string      `json:"notes"`
}

func (in CreateAuditEntryInput) Validate() error {
	if in.Action != nil && !in.Action.Valid() {
		return fmt.Errorf("action tidak dikenal")
	}
	if in.Notes == nil || *in.Notes == "" {
		return fmt.Errorf("notes wajib diisi untuk catatan manual")
	}
	return nil
}
