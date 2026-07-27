// Package paperid integrates Paper.id: it pushes an invoice to Paper.id so the
// member gets a hosted invoice + payment link, and receives the payment
// callback. The client_id/client_secret stay on the server — the browser never
// sees them, same stance as the Xendit integration.
package paperid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/blackbox"
)

// DefaultBaseURL is Paper.id's staging host. Production is set via env.
const DefaultBaseURL = "https://open-api.stag-v2.paper.id"

// dateLayout is what Paper.id expects for invoice_date / due_date.
const dateLayout = "02-01-2006" // DD-MM-YYYY

type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	http         *http.Client
	// rec captures each call for the blackbox page. Optional — a nil recorder
	// is a no-op, so the client works unrecorded.
	rec *blackbox.Recorder
}

func NewClient(baseURL, clientID, clientSecret string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		// Staging latency is variable — observed anywhere from 6s to over 30s.
		// A create that times out on our side may still have succeeded upstream,
		// which burns the invoice number (Paper.id rejects a re-send as
		// duplicate), so give the request comfortable room rather than risk that.
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

// WithRecorder attaches a blackbox recorder. Only bodies are captured; the
// client_id/client_secret live in headers and never reach it.
func (c *Client) WithRecorder(rec *blackbox.Recorder) *Client {
	c.rec = rec
	return c
}

// --- request/response shapes (matching the live staging contract) -----------

type customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone"`
}

type item struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	Price       int64  `json:"price"`
}

type sendFlags struct {
	Email    bool `json:"email"`
	WhatsApp bool `json:"whatsapp"`
	SMS      bool `json:"sms"`
}

type createRequest struct {
	InvoiceDate string    `json:"invoice_date"`
	DueDate     string    `json:"due_date"`
	Number      string    `json:"number"`
	Customer    customer  `json:"customer"`
	Items       []item    `json:"items"`
	Send        sendFlags `json:"send"`
	Notes       string    `json:"notes,omitempty"`
}

// createResponse mirrors the real 201 body:
//
//	{"data":{"id","number","payper_url","pdf_url","pdf_url_short"},"status_code":201}
type createResponse struct {
	Data struct {
		ID          string `json:"id"`
		Number      string `json:"number"`
		PayperURL   string `json:"payper_url"`
		PDFURL      string `json:"pdf_url"`
		PDFURLShort string `json:"pdf_url_short"`
	} `json:"data"`
	StatusCode int `json:"status_code"`
}

// CreateInput is what the service hands the client — domain data, not wire shape.
type CreateInput struct {
	Number      string
	InvoiceDate time.Time
	DueDate     time.Time
	Amount      int64
	ItemName    string
	ItemDesc    string

	CustomerID    string
	CustomerName  string
	CustomerEmail string
	CustomerPhone string

	Notes string

	SendEmail    bool
	SendWhatsApp bool
	SendSMS      bool
}

// CreateResult is the useful part of the response.
type CreateResult struct {
	PaperInvoiceID string // data.id
	Number         string
	PaymentURL     string // data.payper_url — normalised to https
	InvoicePDFURL  string // data.pdf_url
}

// buildCreateRequest turns domain input into the exact wire shape Paper.id
// receives. Shared with the test console, so what the console displays is
// byte-for-byte what a real send would transmit — not a lookalike.
func buildCreateRequest(in CreateInput) createRequest {
	return createRequest{
		InvoiceDate: in.InvoiceDate.Format(dateLayout),
		DueDate:     in.DueDate.Format(dateLayout),
		Number:      in.Number,
		Customer: customer{
			ID:    in.CustomerID,
			Name:  in.CustomerName,
			Email: in.CustomerEmail,
			Phone: in.CustomerPhone,
		},
		Items: []item{{
			Name:        in.ItemName,
			Description: in.ItemDesc,
			Quantity:    1,
			Price:       in.Amount,
		}},
		Send:  sendFlags{Email: in.SendEmail, WhatsApp: in.SendWhatsApp, SMS: in.SendSMS},
		Notes: in.Notes,
	}
}

// CreateInvoice pushes a sales invoice to Paper.id.
func (c *Client) CreateInvoice(ctx context.Context, in CreateInput) (*CreateResult, error) {
	body := buildCreateRequest(in)

	var out createResponse
	if err := c.post(ctx, "/api/v1/store-invoice", body, &out); err != nil {
		return nil, err
	}
	if out.Data.ID == "" {
		return nil, fmt.Errorf("Paper.id tidak mengembalikan id invoice")
	}

	return &CreateResult{
		PaperInvoiceID: out.Data.ID,
		Number:         out.Data.Number,
		PaymentURL:     normalizeURL(out.Data.PayperURL),
		InvoicePDFURL:  out.Data.PDFURL,
	}, nil
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode permintaan Paper.id: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("buat permintaan Paper.id: %w", err)
	}
	// Paper.id authenticates on two custom headers, not Authorization.
	req.Header.Set("client_id", c.clientID)
	req.Header.Set("client_secret", c.clientSecret)
	req.Header.Set("Content-Type", "application/json")

	started := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		c.rec.Record(blackbox.Call{
			Integration: "paper_id", Direction: blackbox.Outbound,
			Method: http.MethodPost, URL: c.baseURL + path,
			Request: payload, Success: false, Duration: time.Since(started), Err: err,
		})
		return fmt.Errorf("hubungi Paper.id: %w", err)
	}
	defer resp.Body.Close()

	// Read the body once: it has to serve both the caller and the recorder.
	raw, readErr := io.ReadAll(resp.Body)
	c.rec.Record(blackbox.Call{
		Integration: "paper_id", Direction: blackbox.Outbound,
		Method: http.MethodPost, URL: c.baseURL + path,
		Request: payload, Response: raw,
		Status: resp.StatusCode, Success: resp.StatusCode < 300,
		Duration: time.Since(started),
	})
	if readErr != nil {
		return fmt.Errorf("baca balasan Paper.id: %w", readErr)
	}

	if resp.StatusCode >= 300 {
		return &apiError{Status: resp.StatusCode, Message: extractMessage(bytes.NewReader(raw), resp.StatusCode)}
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("baca balasan Paper.id: %w", err)
	}
	return nil
}

type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string { return e.Message }

// extractMessage pulls the most useful line out of an error body without
// assuming a single shape — Paper.id returns "message" or "error" depending on
// where validation fails.
func extractMessage(r interface{ Read([]byte) (int, error) }, status int) string {
	var body map[string]any
	if err := json.NewDecoder(r).Decode(&body); err == nil {
		for _, key := range []string{"message", "error", "errors"} {
			if v, ok := body[key]; ok {
				if s, ok := v.(string); ok && s != "" {
					return s
				}
				return fmt.Sprintf("%v", v)
			}
		}
	}
	return fmt.Sprintf("Paper.id menolak permintaan (HTTP %d)", status)
}

// normalizeURL prepends https:// when Paper.id returns a bare host/path like
// "stg-v2.paper.id/8sGbmBn", so the link is clickable.
func normalizeURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" || strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return "https://" + u
}
