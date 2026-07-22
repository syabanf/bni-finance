package publicpay

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Xendit's REST API is two POSTs. Wrapping it here keeps the secret key on the
// server — the browser never sees it, which was the whole point of the edge
// function this replaces.

const (
	xenditBaseURL = "https://api.xendit.co"
	vaExpiryHours = 24
	// QrisMaxAmount is Xendit's per-transaction QRIS ceiling.
	QrisMaxAmount = 10_000_000
)

type xenditClient struct {
	secretKey string
	http      *http.Client
	baseURL   string
}

func newXenditClient(secretKey string) *xenditClient {
	return &xenditClient{
		secretKey: secretKey,
		http:      &http.Client{Timeout: 20 * time.Second},
		baseURL:   xenditBaseURL,
	}
}

func (c *xenditClient) authHeader() string {
	// Xendit uses HTTP Basic with the secret key as the username.
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.secretKey+":"))
}

func (c *xenditClient) post(ctx context.Context, path string, body any, extraHeaders map[string]string, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode permintaan Xendit: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("buat permintaan Xendit: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("hubungi Xendit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var detail map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&detail)
		msg, _ := detail["message"].(string)
		if msg == "" {
			msg = fmt.Sprintf("Xendit menolak permintaan (HTTP %d)", resp.StatusCode)
		}
		return &xenditError{Status: resp.StatusCode, Message: msg}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("baca balasan Xendit: %w", err)
	}
	return nil
}

type xenditError struct {
	Status  int
	Message string
}

func (e *xenditError) Error() string { return e.Message }

type virtualAccount struct {
	ID            string `json:"id"`
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
}

func (c *xenditClient) createVirtualAccount(ctx context.Context, externalID, bank, customerName string, amount int64, expiry time.Time) (*virtualAccount, error) {
	var out virtualAccount
	err := c.post(ctx, "/callback_virtual_accounts", map[string]any{
		"external_id":     externalID,
		"bank_code":       bank,
		"name":            customerName,
		"is_closed":       true,
		"is_single_use":   true,
		"expected_amount": amount,
		"expiration_date": expiry.UTC().Format(time.RFC3339),
	}, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type qrCode struct {
	ID        string  `json:"id"`
	QRString  string  `json:"qr_string"`
	ExpiresAt *string `json:"expires_at"`
}

func (c *xenditClient) createQris(ctx context.Context, referenceID string, amount int64) (*qrCode, error) {
	var out qrCode
	err := c.post(ctx, "/qr_codes", map[string]any{
		"reference_id": referenceID,
		"type":         "DYNAMIC",
		"currency":     "IDR",
		"amount":       amount,
	}, map[string]string{"api-version": "2022-07-31"}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
