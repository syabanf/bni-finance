package paperid

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// The test console: fire a Paper.id call with hand-written data and see exactly
// what went over the wire. It exists because the normal send path needs a real
// draft invoice and real member — too much setup when you only want to know
// whether the credentials work or what the upstream does with a given payload.
//
// Nothing here touches our invoices table. Test invoices are pushed to Paper.id
// under a TEST- prefix so they can never collide with a real invoice number.

// testNumberPrefix keeps console invoices apart from real ones. Paper.id rejects
// a duplicate number permanently, so a test must never spend a real one.
const testNumberPrefix = "TEST"

// ConfigStatus reports how the integration is wired, without revealing secrets.
type ConfigStatus struct {
	Configured         bool   `json:"configured"`
	BaseURL            string `json:"baseUrl"`
	CallbackConfigured bool   `json:"callbackConfigured"`
}

// Status describes the current Paper.id wiring for the test page.
func (s *Service) Status(context.Context) (*ConfigStatus, error) {
	return &ConfigStatus{
		Configured:         s.gateway != nil,
		BaseURL:            s.baseURL,
		CallbackConfigured: s.callbackToken != "",
	}, nil
}

// TestInvoiceInput is the console form. Every field has a sensible default so
// the page works with an empty body.
type TestInvoiceInput struct {
	CustomerName  string `json:"customerName"`
	CustomerPhone string `json:"customerPhone"`
	CustomerEmail string `json:"customerEmail"`
	Amount        int64  `json:"amount"`
	ItemName      string `json:"itemName"`
	Notes         string `json:"notes"`

	SendEmail    bool `json:"sendEmail"`
	SendWhatsApp bool `json:"sendWhatsApp"`

	// DryRun builds the payload and returns it WITHOUT calling Paper.id.
	// Defaults to true: a real call creates a real invoice upstream and
	// permanently consumes its number, so sending must be a deliberate choice.
	DryRun *bool `json:"dryRun"`
}

func (in TestInvoiceInput) dryRun() bool {
	return in.DryRun == nil || *in.DryRun
}

// TestInvoiceResult is what the console shows: the exact request, the raw
// response, and whether it worked.
type TestInvoiceResult struct {
	DryRun     bool            `json:"dryRun"`
	Method     string          `json:"method"`
	URL        string          `json:"url"`
	Request    json.RawMessage `json:"request"`
	Response   json.RawMessage `json:"response,omitempty"`
	Success    bool            `json:"success"`
	DurationMS int64           `json:"durationMs"`
	Error      string          `json:"error,omitempty"`

	// Extracted from a successful response, for convenience.
	PaperInvoiceID string `json:"paperInvoiceId,omitempty"`
	Number         string `json:"number,omitempty"`
	PaymentURL     string `json:"paymentUrl,omitempty"`
	InvoicePDFURL  string `json:"invoicePdfUrl,omitempty"`
}

// TestInvoice runs one console call.
func (s *Service) TestInvoice(ctx context.Context, in TestInvoiceInput) (*TestInvoiceResult, error) {
	now := s.now()
	payload := s.buildTestPayload(in, now)

	url := strings.TrimRight(s.baseURL, "/") + "/api/v1/store-invoice"
	// Serialise the WIRE shape, not the internal struct — the console must show
	// exactly what Paper.id receives.
	body, err := json.Marshal(buildCreateRequest(payload))
	if err != nil {
		return nil, fmt.Errorf("encode payload uji: %w", err)
	}

	result := &TestInvoiceResult{
		DryRun:  in.dryRun(),
		Method:  http.MethodPost,
		URL:     url,
		Request: body,
	}

	if in.dryRun() {
		// Show what WOULD be sent. Success here means "payload built", not
		// "upstream accepted" — the page labels it as such.
		result.Success = true
		return result, nil
	}

	if s.gateway == nil {
		return nil, httpx.NewError(http.StatusServiceUnavailable,
			"Paper.id belum dikonfigurasi — isi PAPER_ID_CLIENT_ID & PAPER_ID_CLIENT_SECRET", nil)
	}

	started := time.Now()
	res, callErr := s.gateway.CreateInvoice(ctx, payload)
	result.DurationMS = time.Since(started).Milliseconds()

	if callErr != nil {
		result.Success = false
		result.Error = callErr.Error()
		// The real request/response pair is in the blackbox; the console shows
		// the reason so you don't have to switch pages to see why it failed.
		return result, nil
	}

	result.Success = true
	result.PaperInvoiceID = res.PaperInvoiceID
	result.Number = res.Number
	result.PaymentURL = res.PaymentURL
	result.InvoicePDFURL = res.InvoicePDFURL

	// Echo the useful fields back as the "response" block.
	echo, _ := json.Marshal(map[string]any{
		"id":          res.PaperInvoiceID,
		"number":      res.Number,
		"payper_url":  res.PaymentURL,
		"pdf_url":     res.InvoicePDFURL,
		"status_code": http.StatusCreated,
	})
	result.Response = echo
	return result, nil
}

// buildTestPayload fills the blanks so the console works with an empty form.
func (s *Service) buildTestPayload(in TestInvoiceInput, now time.Time) CreateInput {
	name := strings.TrimSpace(in.CustomerName)
	if name == "" {
		name = "Uji Coba BNI Finance"
	}
	phone := strings.TrimSpace(in.CustomerPhone)
	if phone == "" {
		phone = "081200000000"
	}
	amount := in.Amount
	if amount <= 0 {
		amount = 1_500_000
	}
	itemName := strings.TrimSpace(in.ItemName)
	if itemName == "" {
		itemName = "Uji Koneksi Paper.id"
	}

	return CreateInput{
		Number:        testInvoiceNumber(now),
		InvoiceDate:   now,
		DueDate:       now.AddDate(0, 0, defaultDueDays),
		Amount:        amount,
		ItemName:      itemName,
		ItemDesc:      "Invoice uji dari konsol — bukan tagihan sungguhan",
		CustomerID:    "console-test",
		CustomerName:  name,
		CustomerEmail: strings.TrimSpace(in.CustomerEmail),
		CustomerPhone: phone,
		Notes:         strings.TrimSpace(in.Notes),
		SendEmail:     in.SendEmail,
		SendWhatsApp:  in.SendWhatsApp,
	}
}

// testSeq makes the suffix unique WITHIN this process. It starts at a random
// offset so two processes restarted in the same second are unlikely to line up.
//
// A plain rand.Intn(10000) per call was not enough: with a one-second timestamp
// the suffix is the only entropy, and by the birthday bound 50 calls in that
// second collide ~11% of the time. Paper.id refuses a repeated number forever,
// so a collision permanently burns it. Counting removes the birthday effect
// entirely; the random start is only there for the restart case.
var testSeq = rand.Uint32()

// testInvoiceNumber is the timestamp plus a per-process counter, so rapid
// clicks cannot collide.
func testInvoiceNumber(now time.Time) string {
	n := atomic.AddUint32(&testSeq, 1)
	return fmt.Sprintf("%s-%s-%04d", testNumberPrefix, now.Format("20060102-150405"), n%10000)
}

// --- simulated callback -----------------------------------------------------

// TestCallbackInput simulates an incoming Paper.id payment callback. Waiting for
// a real staging payment is impractical, so this drives the same settle path
// with a hand-made payload.
type TestCallbackInput struct {
	// InvoiceID is OUR invoice. It must already have been sent to Paper.id.
	InvoiceID string `json:"invoiceId"`
	Amount    int64  `json:"amount"`
	Method    string `json:"method"`
	Channel   string `json:"channel"`
	Status    string `json:"status"`

	// DryRun builds the callback payload without applying it. Default true:
	// applying it marks a real invoice paid.
	DryRun *bool `json:"dryRun"`
}

func (in TestCallbackInput) dryRun() bool {
	return in.DryRun == nil || *in.DryRun
}

type TestCallbackResult struct {
	DryRun  bool            `json:"dryRun"`
	URL     string          `json:"url"`
	Request json.RawMessage `json:"request"`
	Settled bool            `json:"settled"`
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
}

// TestCallback fires a simulated payment callback through the real webhook
// handler, so what it exercises is the production path — not a copy of it.
func (s *Service) TestCallback(ctx context.Context, in TestCallbackInput) (*TestCallbackResult, error) {
	if strings.TrimSpace(in.InvoiceID) == "" {
		return nil, httpx.BadRequest("invoiceId wajib diisi")
	}

	inv, err := s.repo.GetSendable(ctx, in.InvoiceID)
	if err != nil {
		return nil, err
	}
	if inv.Status == domain.StatusDraft {
		return nil, httpx.Conflict("invoice masih draft — kirim ke Paper.id lebih dulu")
	}

	amount := in.Amount
	if amount <= 0 {
		amount = inv.Amount
	}
	method := strings.TrimSpace(in.Method)
	if method == "" {
		method = "bank_transfer"
	}
	channel := strings.TrimSpace(in.Channel)
	if channel == "" {
		channel = "bni"
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "PAID"
	}

	paperID, err := s.repo.PaperInvoiceID(ctx, in.InvoiceID)
	if err != nil {
		return nil, err
	}

	// Payload disusun PERSIS seperti bentuk Paper.id yang sebenarnya, termasuk
	// objek bersarang yang dinamai menurut metode pembayaran. Simulator yang
	// memakai bentuk lebih sederhana daripada aslinya tidak menguji apa pun
	// yang bisa gagal di kenyataan.
	var payload WebhookInput
	payload.PaymentDate = s.now().Format("2006-01-02")
	payload.RefID = "SIMULASI-" + inv.Number
	payload.PaymentInfo, _ = json.Marshal(map[string]any{
		"channel": channel,
		"method":  method,
		"status":  status,
		method: map[string]any{
			"amount":      amount,
			"paid_amount": amount,
			"paid_at":     s.now().Format(time.RFC3339),
			"status":      status,
			"created":     s.now().Format(time.RFC3339),
		},
	})
	payload.AdditionalInfo.Invoices = []struct {
		UUID   string `json:"uuid"`
		Number string `json:"number"`
	}{{UUID: paperID, Number: inv.Number}}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode callback uji: %w", err)
	}

	result := &TestCallbackResult{
		DryRun:  in.dryRun(),
		URL:     "/api/v1/webhooks/paperid",
		Request: body,
	}
	if in.dryRun() {
		result.Success = true
		return result, nil
	}

	// Lewat HandleWebhook dengan body MENTAH, bukan struct — supaya simulasi
	// menempuh jalur parsing yang sama dengan callback sungguhan. Menyuntikkan
	// struct langsung akan melewati satu-satunya langkah yang bisa gagal karena
	// perbedaan format.
	settled, err := s.HandleWebhook(ctx, "/api/v1/webhooks/paperid", s.callbackToken, body)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}
	result.Success = true
	result.Settled = settled
	return result, nil
}
