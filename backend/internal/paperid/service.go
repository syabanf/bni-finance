package paperid

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/blackbox"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// dueDaysSettingKey mirrors the frontend: the due date is issue date + N days.
const dueDaysSettingKey = "invoice_due_days_after"
const defaultDueDays = 30

type Store interface {
	GetSendable(ctx context.Context, invoiceID string) (*Sendable, error)
	MarkSent(ctx context.Context, invoiceID string, res CreateResult, dueDate, sentAt time.Time, actor string) (*domain.Invoice, error)
	SettleByRef(ctx context.Context, paperInvoiceID, number, method, status string, amount int64, paidAt time.Time) (bool, error)
	GetSetting(ctx context.Context, key string) (string, error)
	PaperInvoiceID(ctx context.Context, invoiceID string) (string, error)
}

var _ Store = (*Repository)(nil)

// Gateway is the Paper.id API, kept an interface so the service can be tested
// without a live upstream.
type Gateway interface {
	CreateInvoice(ctx context.Context, in CreateInput) (*CreateResult, error)
}

var _ Gateway = (*Client)(nil)

type Service struct {
	repo          Store
	gateway       Gateway
	baseURL       string
	callbackToken string
	now           func() time.Time
	rec           *blackbox.Recorder
}

// NewService wires the integration. An empty clientID/secret leaves Paper.id
// unavailable rather than half-working: Send returns a clear 503.
//
// rec may be nil; recording is then disabled.
func NewService(repo Store, baseURL, clientID, clientSecret, callbackToken string, rec *blackbox.Recorder) *Service {
	var gw Gateway
	if clientID != "" && clientSecret != "" {
		gw = NewClient(baseURL, clientID, clientSecret).WithRecorder(rec)
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Service{
		repo: repo, gateway: gw, baseURL: baseURL,
		callbackToken: callbackToken, now: time.Now, rec: rec,
	}
}

// recordInbound captures a callback we received, so the blackbox shows both
// sides of the integration.
func (s *Service) recordInbound(in WebhookInput, status int, success bool, err error) {
	if s.rec == nil {
		return
	}
	body, _ := json.Marshal(in)
	s.rec.Record(blackbox.Call{
		Integration: "paper_id", Direction: blackbox.Inbound,
		Method: http.MethodPost, URL: "/api/v1/webhooks/paperid",
		Request: body, Status: status, Success: success, Err: err,
	})
}

// Keys controlling how a sent invoice reaches the member. Absent means off, so
// a fresh install and every staging environment stay silent until someone turns
// delivery on deliberately.
const (
	sendEmailSettingKey    = "paperid_send_email"
	sendWhatsAppSettingKey = "paperid_send_whatsapp"
)

// SendOptions controls which Paper.id delivery channels fire.
//
// Both are *bool, and nil means "use the operational setting". That default
// matters: issuing invoices happens in bulk, and if each caller had to pass the
// flags, one path that forgot would silently stop delivering to members while
// still reporting success. The policy lives in app_settings so it is decided
// once, not re-decided at every call site.
type SendOptions struct {
	Email    *bool `json:"sendEmail"`
	WhatsApp *bool `json:"sendWhatsApp"`
}

// resolve fills the unset channels from app_settings.
func (s *Service) resolve(ctx context.Context, opts SendOptions) (email, whatsapp bool) {
	setting := func(key string) bool {
		v, err := s.repo.GetSetting(ctx, key)
		// A settings read failure must not silently enable messaging.
		return err == nil && v == "true"
	}
	if opts.Email != nil {
		email = *opts.Email
	} else {
		email = setting(sendEmailSettingKey)
	}
	if opts.WhatsApp != nil {
		whatsapp = *opts.WhatsApp
	} else {
		whatsapp = setting(sendWhatsAppSettingKey)
	}
	return email, whatsapp
}

// Send pushes a draft invoice to Paper.id and records the result.
func (s *Service) Send(ctx context.Context, invoiceID string, opts SendOptions) (*domain.Invoice, error) {
	if s.gateway == nil {
		return nil, httpx.NewError(http.StatusServiceUnavailable,
			"Paper.id belum dikonfigurasi — isi PAPER_ID_CLIENT_ID & PAPER_ID_CLIENT_SECRET", nil)
	}

	inv, err := s.repo.GetSendable(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.Status != domain.StatusDraft {
		return nil, httpx.Conflict("hanya invoice draft yang bisa dikirim ke Paper.id")
	}
	// Paper.id requires a phone on the customer; fail early with a clear reason
	// rather than letting the upstream reject it opaquely.
	if strings.TrimSpace(inv.Phone) == "" {
		return nil, httpx.BadRequest("member belum punya nomor telepon — Paper.id mewajibkannya")
	}

	now := s.now()
	dueDate := now.AddDate(0, 0, s.dueDays(ctx))

	sendEmail, sendWhatsApp := s.resolve(ctx, opts)
	// Asking Paper.id to email a customer with no address is a guaranteed
	// upstream failure, and it would abort a send that is otherwise fine —
	// WhatsApp still reaches them. Drop the channel we cannot use instead.
	if strings.TrimSpace(inv.Email) == "" {
		sendEmail = false
	}

	res, err := s.gateway.CreateInvoice(ctx, CreateInput{
		Number:        inv.Number,
		InvoiceDate:   now,
		DueDate:       dueDate,
		Amount:        inv.Amount,
		ItemName:      itemName(inv.Type),
		ItemDesc:      itemDesc(inv.Type),
		CustomerID:    inv.MemberID, // stable, so repeat invoices reuse the customer
		CustomerName:  inv.Name,
		CustomerEmail: inv.Email,
		CustomerPhone: inv.Phone,
		SendEmail:     sendEmail,
		SendWhatsApp:  sendWhatsApp,
	})
	if err != nil {
		return nil, gatewayError(err)
	}

	return s.repo.MarkSent(ctx, invoiceID, *res, dueDate, now, "Admin")
}

func (s *Service) dueDays(ctx context.Context) int {
	v, err := s.repo.GetSetting(ctx, dueDaysSettingKey)
	if err != nil || v == "" {
		return defaultDueDays
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n <= 0 {
		return defaultDueDays
	}
	return n
}

// --- webhook ----------------------------------------------------------------

// WebhookInput matches the documented Paper.id payment callback.
type WebhookInput struct {
	RefID       string `json:"ref_id"`
	ExternalID  string `json:"external_id"`
	PaymentDate string `json:"payment_date"`
	PaymentInfo struct {
		Method     string  `json:"method"`
		Channel    string  `json:"channel"`
		Amount     float64 `json:"amount"`
		PaidAmount float64 `json:"paid_amount"`
		PaidAt     string  `json:"paid_at"`
		Status     string  `json:"status"`
	} `json:"payment_info"`
	AdditionalInfo struct {
		Invoices []struct {
			UUID   string `json:"uuid"`
			Number string `json:"number"`
		} `json:"invoices"`
	} `json:"additional_info"`
}

// HandleWebhook verifies the shared secret and settles the invoice.
//
// Paper.id's docs describe no signature, so the callback URL registered in their
// dashboard carries a secret token (?token=… or the x-paper-callback-token
// header) that we compare here. An unconfigured token rejects every callback
// rather than accepting them all.
func (s *Service) HandleWebhook(ctx context.Context, token string, in WebhookInput) (settled bool, err error) {
	if s.callbackToken == "" {
		err := httpx.Unauthorized("callback Paper.id belum dikonfigurasi")
		s.recordInbound(in, http.StatusUnauthorized, false, err)
		return false, err
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.callbackToken)) != 1 {
		err := httpx.Unauthorized("token callback tidak valid")
		s.recordInbound(in, http.StatusUnauthorized, false, err)
		return false, err
	}

	// Only a completed payment settles; other events are acknowledged (200) but
	// change nothing.
	status := strings.ToUpper(strings.TrimSpace(in.PaymentInfo.Status))
	if status != "PAID" && status != "SETTLED" && status != "SUCCESS" && status != "SUCCEEDED" {
		// Acknowledged but not acted on — still worth recording.
		s.recordInbound(in, http.StatusOK, true, nil)
		return false, nil
	}

	var uuid, number string
	if len(in.AdditionalInfo.Invoices) > 0 {
		uuid = in.AdditionalInfo.Invoices[0].UUID
		number = in.AdditionalInfo.Invoices[0].Number
	}
	if uuid == "" && number == "" {
		err := httpx.BadRequest("callback tidak memuat invoice yang bisa dicocokkan")
		s.recordInbound(in, http.StatusBadRequest, false, err)
		return false, err
	}

	amount := int64(in.PaymentInfo.PaidAmount)
	if amount <= 0 {
		amount = int64(in.PaymentInfo.Amount)
	}
	if amount <= 0 {
		return false, httpx.BadRequest("amount pada callback tidak valid")
	}

	method := strings.TrimSpace(in.PaymentInfo.Method)
	if in.PaymentInfo.Channel != "" {
		method = strings.TrimSpace(method + ":" + in.PaymentInfo.Channel)
	}
	if method == "" {
		method = "paper_id"
	}

	settled, err = s.repo.SettleByRef(ctx, uuid, number, method, status, amount, s.paidAt(in))
	if err != nil {
		s.recordInbound(in, http.StatusInternalServerError, false, err)
		return false, err
	}
	s.recordInbound(in, http.StatusOK, true, nil)
	return settled, nil
}

// paidAt prefers payment_info.paid_at, then payment_date, then now.
func (s *Service) paidAt(in WebhookInput) time.Time {
	for _, raw := range []string{in.PaymentInfo.PaidAt, in.PaymentDate} {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if t, err := time.Parse(layout, raw); err == nil {
				return t
			}
		}
	}
	return s.now()
}

func itemName(t domain.InvoiceType) string {
	if t == domain.TypeRegistration {
		return "Biaya Pendaftaran Member BNI Grow"
	}
	return "Perpanjangan Keanggotaan BNI Grow"
}

func itemDesc(t domain.InvoiceType) string {
	if t == domain.TypeRegistration {
		return "Pendaftaran anggota baru (berlaku 1 tahun)"
	}
	return "Perpanjangan keanggotaan tahunan"
}

// gatewayError maps an upstream failure to a status code, keeping Paper.id's
// own message so a rejection isn't mistaken for a bug on our side.
//
// A duplicate invoice number is called out specifically: it means the invoice
// already exists on Paper.id — usually because a previous attempt succeeded
// upstream but timed out before we could persist it — so it's a 409 the operator
// can act on, not a generic gateway failure.
func gatewayError(err error) error {
	var ae *apiError
	if errors.As(err, &ae) {
		if isDuplicateNumber(ae) {
			return httpx.Conflict(
				"invoice ini sudah pernah dibuat di Paper.id (nomor sudah dipakai) — " +
					"periksa dashboard Paper.id sebelum mengirim ulang")
		}
		return httpx.NewError(http.StatusBadGateway, "Paper.id menolak: "+ae.Message, err)
	}
	return httpx.NewError(http.StatusBadGateway, "tidak bisa menghubungi Paper.id", err)
}

func isDuplicateNumber(ae *apiError) bool {
	m := strings.ToLower(ae.Message)
	return strings.Contains(m, "number sudah dipakai") ||
		strings.Contains(m, "number already") ||
		strings.Contains(m, "already used") ||
		strings.Contains(m, "duplicate")
}
