package publicpay

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
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

// VaBanks are the banks the UI offers.
var VaBanks = []string{"BCA", "BNI", "MANDIRI", "BRI"}

type Store interface {
	GetPublicInvoice(ctx context.Context, id string) (*PublicInvoice, error)
	getBillable(ctx context.Context, id string) (*billable, error)
	saveVirtualAccount(ctx context.Context, invoiceID, externalID, paymentID, bank, number string, expires time.Time) error
	saveQris(ctx context.Context, invoiceID, externalID, paymentID, qrString string, expires *time.Time) error
	SettleByExternalID(ctx context.Context, externalID, xenditPaymentID, status string, amount int64, paidAt time.Time) (bool, error)
	GetSetting(ctx context.Context, key string) (string, error)
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo          Store
	xendit        *xenditClient
	callbackToken string
	now           func() time.Time
	rec           *blackbox.Recorder
}

// recordInbound captures a Xendit callback, so the blackbox shows both
// directions of the integration.
func (s *Service) recordInbound(in WebhookInput, status int, success bool, err error) {
	if s.rec == nil {
		return
	}
	body, _ := json.Marshal(in)
	s.rec.Record(blackbox.Call{
		Integration: "xendit", Direction: blackbox.Inbound,
		Method: http.MethodPost, URL: "/api/v1/webhooks/xendit",
		Request: body, Status: status, Success: success, Err: err,
	})
}

// NewService wires the payment gateway. An empty secretKey leaves self-payment
// unavailable rather than half-working: creating a charge returns a clear error
// instead of a confusing failure from Xendit.
func NewService(repo Store, xenditSecretKey, callbackToken string, rec *blackbox.Recorder) *Service {
	var client *xenditClient
	if xenditSecretKey != "" {
		client = newXenditClient(xenditSecretKey)
		client.rec = rec
	}
	return &Service{repo: repo, xendit: client, callbackToken: callbackToken, now: time.Now, rec: rec}
}

// PublicView is what the /pay/:id page renders.
type PublicView struct {
	Invoice         *PublicInvoice `json:"invoice"`
	SelfPaymentMode bool           `json:"selfPaymentMode"`
}

func (s *Service) View(ctx context.Context, invoiceID string) (*PublicView, error) {
	inv, err := s.repo.GetPublicInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	mode, err := s.repo.GetSetting(ctx, "self_payment_mode")
	if err != nil {
		return nil, err
	}
	return &PublicView{Invoice: inv, SelfPaymentMode: mode == "true"}, nil
}

// PaymentResult matches the frontend's XenditPaymentResult.
type PaymentResult struct {
	Method    string     `json:"method"`
	Bank      string     `json:"bank,omitempty"`
	VaNumber  string     `json:"vaNumber,omitempty"`
	QrString  string     `json:"qrString,omitempty"`
	Amount    int64      `json:"amount"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type CreatePaymentInput struct {
	Method string `json:"method"`
	Bank   string `json:"bank"`
}

// CreatePayment opens a Xendit charge for an invoice.
//
// Self Payment Mode is re-checked here rather than trusted from the client:
// the toggle is a server-side setting, so turning it off must actually stop
// charges being created, not just hide the button.
func (s *Service) CreatePayment(ctx context.Context, invoiceID string, in CreatePaymentInput) (*PaymentResult, error) {
	mode, err := s.repo.GetSetting(ctx, "self_payment_mode")
	if err != nil {
		return nil, err
	}
	if mode != "true" {
		return nil, httpx.Forbidden("pembayaran mandiri sedang dinonaktifkan")
	}
	if s.xendit == nil {
		return nil, httpx.NewError(503, "gateway pembayaran belum dikonfigurasi", nil)
	}

	method := strings.ToLower(strings.TrimSpace(in.Method))
	if method != "va" && method != "qris" {
		return nil, httpx.BadRequest("method harus 'va' atau 'qris'")
	}
	bank := strings.ToUpper(strings.TrimSpace(in.Bank))
	if method == "va" && !contains(VaBanks, bank) {
		return nil, httpx.BadRequest("bank harus salah satu dari: " + strings.Join(VaBanks, ", "))
	}

	inv, err := s.repo.getBillable(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.Status == domain.StatusPaid || inv.Status == domain.StatusCancelled {
		return nil, httpx.Conflict("invoice berstatus " + string(inv.Status) + ", tidak bisa dibayar")
	}

	// A fresh reference each time: Xendit rejects reusing an external_id that
	// still has an active VA or QR attached.
	suffix, err := randomSuffix()
	if err != nil {
		return nil, err
	}
	externalID := fmt.Sprintf("%s-%s-%s", inv.Number, method, suffix)

	if method == "va" {
		expiry := s.now().Add(vaExpiryHours * time.Hour)
		va, err := s.xendit.createVirtualAccount(ctx, externalID, bank, inv.MemberName, inv.Amount, expiry)
		if err != nil {
			return nil, gatewayError(err)
		}
		if err := s.repo.saveVirtualAccount(ctx, inv.ID, externalID, va.ID, va.BankCode, va.AccountNumber, expiry); err != nil {
			return nil, err
		}
		return &PaymentResult{
			Method: "va", Bank: va.BankCode, VaNumber: va.AccountNumber,
			Amount: inv.Amount, ExpiresAt: &expiry,
		}, nil
	}

	if inv.Amount > QrisMaxAmount {
		return nil, httpx.NewError(422,
			"nominal melebihi batas QRIS (maks Rp 10.000.000) — gunakan Virtual Account", nil)
	}
	qr, err := s.xendit.createQris(ctx, externalID, inv.Amount)
	if err != nil {
		return nil, gatewayError(err)
	}

	var expires *time.Time
	if qr.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339, *qr.ExpiresAt); err == nil {
			expires = &t
		}
	}
	if err := s.repo.saveQris(ctx, inv.ID, externalID, qr.ID, qr.QRString, expires); err != nil {
		return nil, err
	}
	return &PaymentResult{Method: "qris", QrString: qr.QRString, Amount: inv.Amount, ExpiresAt: expires}, nil
}

// WebhookInput covers both callback shapes Xendit sends. VA callbacks use
// external_id; QR callbacks use reference_id.
type WebhookInput struct {
	ExternalID  string `json:"external_id"`
	ReferenceID string `json:"reference_id"`
	ID          string `json:"id"`
	PaymentID   string `json:"payment_id"`
	Amount      int64  `json:"amount"`
	Status      string `json:"status"`
	Created     string `json:"created"`
}

func (in WebhookInput) reference() string {
	if in.ExternalID != "" {
		return in.ExternalID
	}
	return in.ReferenceID
}

func (in WebhookInput) paymentIdentifier() string {
	if in.PaymentID != "" {
		return in.PaymentID
	}
	return in.ID
}

// HandleWebhook verifies the callback token and settles the invoice.
//
// The token check is the only thing standing between this endpoint and anyone
// on the internet marking invoices paid, so an unconfigured token refuses every
// callback rather than accepting them all.
func (s *Service) HandleWebhook(ctx context.Context, callbackToken string, in WebhookInput) (settled bool, err error) {
	if s.callbackToken == "" {
		err := httpx.Unauthorized("webhook belum dikonfigurasi")
		s.recordInbound(in, http.StatusUnauthorized, false, err)
		return false, err
	}
	if subtle.ConstantTimeCompare([]byte(callbackToken), []byte(s.callbackToken)) != 1 {
		err := httpx.Unauthorized("callback token tidak valid")
		s.recordInbound(in, http.StatusUnauthorized, false, err)
		return false, err
	}
	if in.reference() == "" {
		return false, httpx.BadRequest("external_id atau reference_id wajib ada")
	}

	// Xendit sends several event types; only a completed payment settles.
	status := strings.ToUpper(in.Status)
	if status != "" && status != "PAID" && status != "SUCCEEDED" && status != "COMPLETED" && status != "ACTIVE" {
		return false, nil
	}
	if in.Amount <= 0 {
		return false, httpx.BadRequest("amount tidak valid")
	}

	paidAt := s.now()
	if in.Created != "" {
		if t, err := time.Parse(time.RFC3339, in.Created); err == nil {
			paidAt = t
		}
	}
	settled, err = s.repo.SettleByExternalID(ctx, in.reference(), in.paymentIdentifier(), status, in.Amount, paidAt)
	if err != nil {
		s.recordInbound(in, http.StatusInternalServerError, false, err)
		return false, err
	}
	s.recordInbound(in, http.StatusOK, true, nil)
	return settled, nil
}

// gatewayError keeps Xendit's own message but reports it as a bad gateway, so
// an upstream failure is never mistaken for a bug on our side.
func gatewayError(err error) error {
	var xe *xenditError
	if errors.As(err, &xe) {
		return httpx.NewError(502, "gateway pembayaran menolak: "+xe.Message, err)
	}
	return httpx.NewError(502, "tidak bisa menghubungi gateway pembayaran", err)
}

func randomSuffix() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("buat referensi: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
