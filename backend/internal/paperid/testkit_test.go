package paperid

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/apidocs"
	"github.com/syabanf/bni-finance/backend/internal/domain"
)

// The whole point of dry-run is that nothing leaves the building. If this ever
// regresses, the console would create real invoices on Paper.id by default.
func TestTestInvoiceDryRunDoesNotCallUpstream(t *testing.T) {
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1"}}
	svc := newService(&stubStore{}, gw, "tok")

	res, err := svc.TestInvoice(context.Background(), TestInvoiceInput{})
	if err != nil {
		t.Fatalf("TestInvoice: %v", err)
	}
	if gw.called {
		t.Fatal("dry-run tidak boleh memanggil Paper.id")
	}
	if !res.DryRun || !res.Success {
		t.Errorf("dry-run harus sukses menyusun payload: %+v", res)
	}
	if res.Response != nil {
		t.Errorf("dry-run tidak boleh punya response: %s", res.Response)
	}
	if !strings.HasSuffix(res.URL, "/api/v1/store-invoice") {
		t.Errorf("URL target salah: %s", res.URL)
	}
}

// Defaults must produce a payload Paper.id would accept, so an empty form is
// still a useful connectivity test.
func TestTestInvoiceFillsDefaults(t *testing.T) {
	svc := newService(&stubStore{}, &stubGateway{}, "tok")

	res, err := svc.TestInvoice(context.Background(), TestInvoiceInput{})
	if err != nil {
		t.Fatalf("TestInvoice: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Request, &body); err != nil {
		t.Fatalf("request bukan JSON valid: %v", err)
	}
	number, _ := body["number"].(string)
	if !strings.HasPrefix(number, testNumberPrefix+"-") {
		t.Errorf("nomor uji harus ber-prefix %s, dapat %q", testNumberPrefix, number)
	}
	customer, _ := body["customer"].(map[string]any)
	if customer["phone"] == "" || customer["name"] == "" {
		t.Errorf("default customer kosong: %v", customer)
	}
	// Dates must already be in Paper.id's DD-MM-YYYY form.
	if d, _ := body["invoice_date"].(string); d != "27-07-2026" {
		t.Errorf("format invoice_date salah: %q", d)
	}
	// Nothing may be delivered unless explicitly asked.
	send, _ := body["send"].(map[string]any)
	if send["email"] != false || send["whatsapp"] != false {
		t.Errorf("default kirim harus mati semua: %v", send)
	}
}

// Two rapid tests must not share a number — Paper.id refuses a repeat forever.
func TestTestInvoiceNumbersAreUnique(t *testing.T) {
	svc := newService(&stubStore{}, &stubGateway{}, "tok")
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		res, err := svc.TestInvoice(context.Background(), TestInvoiceInput{})
		if err != nil {
			t.Fatalf("TestInvoice: %v", err)
		}
		var body map[string]any
		json.Unmarshal(res.Request, &body)
		number := body["number"].(string)
		if seen[number] {
			t.Fatalf("nomor uji berulang: %s", number)
		}
		seen[number] = true
	}
}

func TestTestInvoiceLiveCallsGateway(t *testing.T) {
	gw := &stubGateway{res: &CreateResult{
		PaperInvoiceID: "pp-9", Number: "TEST-1", PaymentURL: "https://stg-v2.paper.id/z",
	}}
	svc := newService(&stubStore{}, gw, "tok")

	live := false
	res, err := svc.TestInvoice(context.Background(), TestInvoiceInput{
		DryRun: &live, Amount: 250_000, CustomerName: "Siti",
	})
	if err != nil {
		t.Fatalf("TestInvoice: %v", err)
	}
	if !gw.called {
		t.Fatal("mode kirim harus memanggil Paper.id")
	}
	if gw.gotIn.Amount != 250_000 || gw.gotIn.CustomerName != "Siti" {
		t.Errorf("input form tidak diteruskan: %+v", gw.gotIn)
	}
	if !res.Success || res.PaperInvoiceID != "pp-9" || res.PaymentURL == "" {
		t.Errorf("hasil tidak lengkap: %+v", res)
	}
}

// An upstream rejection is reported in the result, not as a transport error —
// the console needs to render the reason rather than blow up.
func TestTestInvoiceReportsUpstreamFailure(t *testing.T) {
	gw := &stubGateway{err: &apiError{Status: 403, Message: "invoice number sudah dipakai"}}
	svc := newService(&stubStore{}, gw, "tok")

	live := false
	res, err := svc.TestInvoice(context.Background(), TestInvoiceInput{DryRun: &live})
	if err != nil {
		t.Fatalf("kegagalan upstream tidak boleh jadi error transport: %v", err)
	}
	if res.Success || res.Error == "" {
		t.Errorf("kegagalan harus terlaporkan di hasil: %+v", res)
	}
}

func TestTestInvoiceUnconfiguredIs503(t *testing.T) {
	svc := newService(&stubStore{}, nil, "tok")
	live := false
	if _, err := svc.TestInvoice(context.Background(), TestInvoiceInput{DryRun: &live}); statusOf(err) != 503 {
		t.Fatalf("tanpa kredensial harus 503, dapat %v", err)
	}
	// Dry-run must still work without credentials — it never calls out.
	if _, err := svc.TestInvoice(context.Background(), TestInvoiceInput{}); err != nil {
		t.Errorf("dry-run harus tetap jalan tanpa kredensial: %v", err)
	}
}

// --- simulated callback -----------------------------------------------------

func sentSendable() *Sendable {
	s := draftSendable()
	s.Status = domain.StatusSent
	return s
}

func TestTestCallbackDryRunDoesNotSettle(t *testing.T) {
	store := &stubStore{sendable: sentSendable(), paperInvoiceID: "pp-1"}
	svc := newService(store, &stubGateway{}, "rahasia")

	res, err := svc.TestCallback(context.Background(), TestCallbackInput{InvoiceID: "inv-1"})
	if err != nil {
		t.Fatalf("TestCallback: %v", err)
	}
	if store.settleRef.called {
		t.Fatal("dry-run tidak boleh melunasi invoice")
	}
	if !res.DryRun || !res.Success {
		t.Errorf("dry-run harus sukses: %+v", res)
	}

	// The payload must match the real Paper.id callback shape.
	var body map[string]any
	if err := json.Unmarshal(res.Request, &body); err != nil {
		t.Fatalf("payload bukan JSON: %v", err)
	}
	info := body["payment_info"].(map[string]any)
	if info["status"] != "PAID" {
		t.Errorf("status default harus PAID: %v", info["status"])
	}
	invoices := body["additional_info"].(map[string]any)["invoices"].([]any)
	first := invoices[0].(map[string]any)
	if first["uuid"] != "pp-1" || first["number"] != "INV-2026-001" {
		t.Errorf("invoice pada payload salah: %v", first)
	}
}

func TestTestCallbackLiveSettles(t *testing.T) {
	store := &stubStore{sendable: sentSendable(), paperInvoiceID: "pp-1", settleReturns: true}
	svc := newService(store, &stubGateway{}, "rahasia")

	live := false
	res, err := svc.TestCallback(context.Background(), TestCallbackInput{
		InvoiceID: "inv-1", DryRun: &live,
	})
	if err != nil {
		t.Fatalf("TestCallback: %v", err)
	}
	if !store.settleRef.called {
		t.Fatal("mode kirim harus melunasi")
	}
	if !res.Settled || !res.Success {
		t.Errorf("hasil salah: %+v", res)
	}
	// It must go through the real verification path, so the amount defaults to
	// the invoice's own.
	if store.settleRef.amount != 1_500_000 {
		t.Errorf("amount harus jatuh ke nominal invoice, dapat %d", store.settleRef.amount)
	}
}

// Simulating a callback for an invoice that was never pushed makes no sense —
// there is no Paper.id side to reply.
func TestTestCallbackRejectsDraft(t *testing.T) {
	store := &stubStore{sendable: draftSendable()}
	svc := newService(store, &stubGateway{}, "rahasia")

	_, err := svc.TestCallback(context.Background(), TestCallbackInput{InvoiceID: "inv-1"})
	if statusOf(err) != 409 {
		t.Fatalf("invoice draft harus 409, dapat %v", err)
	}
}

func TestTestCallbackRequiresInvoiceID(t *testing.T) {
	svc := newService(&stubStore{}, &stubGateway{}, "rahasia")
	if _, err := svc.TestCallback(context.Background(), TestCallbackInput{}); statusOf(err) != 400 {
		t.Fatalf("tanpa invoiceId harus 400, dapat %v", err)
	}
}

func TestStatusHidesSecrets(t *testing.T) {
	svc := newService(&stubStore{setting: "true"}, &stubGateway{}, "rahasia-sekali")

	st, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.Configured || !st.CallbackConfigured {
		t.Errorf("status salah: %+v", st)
	}
	// Only booleans and the base URL — never the token itself.
	raw, _ := json.Marshal(st)
	if strings.Contains(string(raw), "rahasia-sekali") {
		t.Errorf("status membocorkan token: %s", raw)
	}
}

// The console form is rendered from openapi.yaml, so the defaults documented
// there are what a reader sees pre-filled. If they drift from what the server
// actually substitutes, the form shows one thing and sends another — a lie that
// no other test would catch, because each side is self-consistent.
func TestDocumentedDefaultsMatchServerDefaults(t *testing.T) {
	var spec struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Default any `json:"default"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(apidocs.SpecJSON(), &spec); err != nil {
		t.Fatalf("baca spesifikasi: %v", err)
	}
	props := spec.Components.Schemas["PaperTestInvoiceInput"].Properties
	if len(props) == 0 {
		t.Fatal("PaperTestInvoiceInput tidak ada di spesifikasi")
	}

	svc := newService(&stubStore{}, &stubGateway{}, "tok")
	got := svc.buildTestPayload(TestInvoiceInput{}, svc.now())

	cases := []struct {
		field  string
		actual any
	}{
		{"customerName", got.CustomerName},
		{"customerPhone", got.CustomerPhone},
		{"itemName", got.ItemName},
		{"amount", float64(got.Amount)}, // JSON numbers decode as float64
	}
	for _, c := range cases {
		documented := props[c.field].Default
		if documented == nil {
			t.Errorf("%s: tidak punya `default` di spesifikasi, padahal server mengisinya dengan %v",
				c.field, c.actual)
			continue
		}
		if fmt.Sprint(documented) != fmt.Sprint(c.actual) {
			t.Errorf("%s: spesifikasi bilang %v, server memakai %v", c.field, documented, c.actual)
		}
	}

	// Delivery must stay off in the documented defaults too — a form that
	// arrives with WhatsApp pre-ticked would message someone on first click.
	for _, quiet := range []string{"sendEmail", "sendWhatsApp"} {
		if v := props[quiet].Default; v != false {
			t.Errorf("%s harus berbawaan false di spesifikasi, dapat %v", quiet, v)
		}
	}
	// And dry-run must stay on.
	if v := props["dryRun"].Default; v != true {
		t.Errorf("dryRun harus berbawaan true di spesifikasi, dapat %v", v)
	}
}
