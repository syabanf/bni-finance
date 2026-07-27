package paperid

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateInvoiceSendsContractAndParsesResponse(t *testing.T) {
	var gotHeaders http.Header
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/store-invoice" {
			t.Errorf("dipanggil ke %s %s", r.Method, r.URL.Path)
		}
		gotHeaders = r.Header
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &gotBody)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data":{
			"id":"d46442ea-fb67-4e6e-8e0b-01d259f64a4a",
			"number":"INV-2026-001",
			"payper_url":"stg-v2.paper.id/8sGbmBn",
			"pdf_url":"https://storage.googleapis.com/x/INV.pdf",
			"pdf_url_short":"stg-v2.paper.id/8ck6W44"
		},"status_code":201}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-id", "my-secret")
	res, err := c.CreateInvoice(context.Background(), CreateInput{
		Number:        "INV-2026-001",
		InvoiceDate:   time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC),
		Amount:        1_500_000,
		ItemName:      "Renewal",
		ItemDesc:      "Perpanjangan",
		CustomerID:    "mem-001",
		CustomerName:  "Budi",
		CustomerEmail: "budi@example.com",
		CustomerPhone: "081200000001",
	})
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	// Auth is on custom headers, not Authorization.
	if gotHeaders.Get("client_id") != "my-id" || gotHeaders.Get("client_secret") != "my-secret" {
		t.Errorf("header auth salah: %q / %q",
			gotHeaders.Get("client_id"), gotHeaders.Get("client_secret"))
	}

	// Dates must be DD-MM-YYYY.
	if gotBody["invoice_date"] != "27-07-2026" || gotBody["due_date"] != "26-08-2026" {
		t.Errorf("format tanggal salah: %v / %v", gotBody["invoice_date"], gotBody["due_date"])
	}

	items := gotBody["items"].([]any)
	first := items[0].(map[string]any)
	if first["price"].(float64) != 1_500_000 || first["quantity"].(float64) != 1 {
		t.Errorf("item salah: %v", first)
	}

	// Bare host in payper_url must come back as a usable https URL.
	if res.PaymentURL != "https://stg-v2.paper.id/8sGbmBn" {
		t.Errorf("payment URL tidak dinormalkan: %q", res.PaymentURL)
	}
	if res.PaperInvoiceID != "d46442ea-fb67-4e6e-8e0b-01d259f64a4a" {
		t.Errorf("paper invoice id salah: %q", res.PaperInvoiceID)
	}
	if res.InvoicePDFURL != "https://storage.googleapis.com/x/INV.pdf" {
		t.Errorf("pdf url salah: %q", res.InvoicePDFURL)
	}
}

func TestCreateInvoiceSurfacesUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"customer phone is required","status_code":400}`))
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, "id", "secret").CreateInvoice(context.Background(), CreateInput{})
	if err == nil {
		t.Fatal("error upstream seharusnya diteruskan")
	}
	var ae *apiError
	if !asAPIError(err, &ae) {
		t.Fatalf("harusnya *apiError, dapat %T", err)
	}
	if ae.Status != 400 || ae.Message != "customer phone is required" {
		t.Errorf("detail error hilang: %+v", ae)
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"stg-v2.paper.id/abc":      "https://stg-v2.paper.id/abc",
		"https://paper.id/x":       "https://paper.id/x",
		"http://paper.id/x":        "http://paper.id/x",
		"":                         "",
		"  stg-v2.paper.id/trim  ": "https://stg-v2.paper.id/trim",
	}
	for in, want := range cases {
		if got := normalizeURL(in); got != want {
			t.Errorf("normalizeURL(%q) = %q, mau %q", in, got, want)
		}
	}
}

func asAPIError(err error, target **apiError) bool {
	for err != nil {
		if ae, ok := err.(*apiError); ok {
			*target = ae
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
