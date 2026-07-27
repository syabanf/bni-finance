package paperid

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type stubStore struct {
	sendable *Sendable
	// setting is the value returned for any key; settings overrides it per key
	// for the tests that care which key was read.
	setting        string
	settings       map[string]string
	paperInvoiceID string

	sentWith  *CreateResult
	settleRef struct {
		paperID, number, method, status string
		amount                          int64
		called                          bool
	}
	settleReturns bool
}

func (s *stubStore) GetSendable(context.Context, string) (*Sendable, error) {
	if s.sendable == nil {
		return nil, httpx.NotFound("invoice tidak ditemukan")
	}
	return s.sendable, nil
}

func (s *stubStore) MarkSent(_ context.Context, id string, res CreateResult, _, _ time.Time, _ string) (*domain.Invoice, error) {
	s.sentWith = &res
	return &domain.Invoice{ID: id, Status: domain.StatusSent}, nil
}

func (s *stubStore) SettleByRef(_ context.Context, paperID, number, method, status string, amount int64, _ time.Time) (bool, error) {
	s.settleRef.paperID = paperID
	s.settleRef.number = number
	s.settleRef.method = method
	s.settleRef.status = status
	s.settleRef.amount = amount
	s.settleRef.called = true
	return s.settleReturns, nil
}

func (s *stubStore) GetSetting(_ context.Context, key string) (string, error) {
	if v, ok := s.settings[key]; ok {
		return v, nil
	}
	return s.setting, nil
}

func (s *stubStore) PaperInvoiceID(context.Context, string) (string, error) {
	return s.paperInvoiceID, nil
}

type stubGateway struct {
	res    *CreateResult
	err    error
	gotIn  CreateInput
	called bool
}

func (g *stubGateway) CreateInvoice(_ context.Context, in CreateInput) (*CreateResult, error) {
	g.called = true
	g.gotIn = in
	if g.err != nil {
		return nil, g.err
	}
	return g.res, nil
}

func newService(store Store, gw Gateway, token string) *Service {
	return &Service{repo: store, gateway: gw, baseURL: DefaultBaseURL, callbackToken: token,
		now: func() time.Time {
			return time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
		}}
}

func draftSendable() *Sendable {
	return &Sendable{
		ID: "inv-1", Number: "INV-2026-001", Amount: 1_500_000,
		Type: domain.TypeRenewal, Status: domain.StatusDraft,
		MemberID: "mem-1", Name: "Budi", Email: "budi@example.com", Phone: "0812",
	}
}

func statusOf(err error) int {
	var he *httpx.Error
	if errors.As(err, &he) {
		return he.Status
	}
	return 0
}

func TestSendUnconfiguredIs503(t *testing.T) {
	svc := newService(&stubStore{sendable: draftSendable()}, nil, "tok")
	_, err := svc.Send(context.Background(), "inv-1", SendOptions{})
	if statusOf(err) != 503 {
		t.Fatalf("tanpa gateway harus 503, dapat %v", err)
	}
}

func TestSendRejectsNonDraft(t *testing.T) {
	s := draftSendable()
	s.Status = domain.StatusSent
	svc := newService(&stubStore{sendable: s}, &stubGateway{}, "tok")

	_, err := svc.Send(context.Background(), "inv-1", SendOptions{})
	if statusOf(err) != 409 {
		t.Fatalf("invoice non-draft harus 409, dapat %v", err)
	}
}

func TestSendRequiresPhone(t *testing.T) {
	s := draftSendable()
	s.Phone = "   "
	gw := &stubGateway{}
	svc := newService(&stubStore{sendable: s}, gw, "tok")

	_, err := svc.Send(context.Background(), "inv-1", SendOptions{})
	if statusOf(err) != 400 {
		t.Fatalf("tanpa telepon harus 400, dapat %v", err)
	}
	if gw.called {
		t.Error("Paper.id tidak boleh dipanggil kalau telepon kosong")
	}
}

func TestSendHappyPath(t *testing.T) {
	store := &stubStore{sendable: draftSendable(), setting: "45"}
	gw := &stubGateway{res: &CreateResult{
		PaperInvoiceID: "pp-1", PaymentURL: "https://stg-v2.paper.id/x", InvoicePDFURL: "https://x/INV.pdf",
	}}
	svc := newService(store, gw, "tok")

	on := true
	inv, err := svc.Send(context.Background(), "inv-1", SendOptions{Email: &on})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if inv.Status != domain.StatusSent {
		t.Errorf("status harus sent, dapat %s", inv.Status)
	}
	// Customer id must be the member id, so repeat invoices reuse the customer.
	if gw.gotIn.CustomerID != "mem-1" {
		t.Errorf("customer id harus member id, dapat %q", gw.gotIn.CustomerID)
	}
	// Due date follows the app_settings value (45 days), not the default.
	wantDue := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	if !gw.gotIn.DueDate.Equal(wantDue) {
		t.Errorf("due date harus mengikuti setting 45 hari, dapat %v", gw.gotIn.DueDate)
	}
	if !gw.gotIn.SendEmail || gw.gotIn.SendWhatsApp {
		t.Errorf("opsi kirim tidak diteruskan benar: %+v", gw.gotIn)
	}
	if store.sentWith == nil || store.sentWith.PaperInvoiceID != "pp-1" {
		t.Errorf("hasil Paper.id tidak diteruskan ke MarkSent: %+v", store.sentWith)
	}
}

func TestSendMapsUpstreamErrorTo502(t *testing.T) {
	gw := &stubGateway{err: &apiError{Status: 400, Message: "customer phone is required"}}
	svc := newService(&stubStore{sendable: draftSendable()}, gw, "tok")

	_, err := svc.Send(context.Background(), "inv-1", SendOptions{})
	if statusOf(err) != 502 {
		t.Fatalf("error upstream harus jadi 502, dapat %v", err)
	}
}

// A duplicate number means the invoice already exists on Paper.id (a prior
// attempt that timed out on our side). That's a 409 the operator can act on,
// not an opaque 502.
func TestSendDuplicateNumberIs409(t *testing.T) {
	gw := &stubGateway{err: &apiError{Status: 403, Message: "invoice number sudah dipakai"}}
	svc := newService(&stubStore{sendable: draftSendable()}, gw, "tok")

	_, err := svc.Send(context.Background(), "inv-1", SendOptions{})
	if statusOf(err) != 409 {
		t.Fatalf("nomor duplikat harus 409, dapat %v", err)
	}
}

// --- webhook ----------------------------------------------------------------

func paidWebhook() WebhookInput {
	var in WebhookInput
	in.PaymentInfo.Status = "PAID"
	in.PaymentInfo.Method = "bank_transfer"
	in.PaymentInfo.Channel = "bni"
	in.PaymentInfo.PaidAmount = 1_500_000
	in.PaymentInfo.PaidAt = "2026-07-27 10:00:00"
	in.AdditionalInfo.Invoices = []struct {
		UUID   string `json:"uuid"`
		Number string `json:"number"`
	}{{UUID: "pp-1", Number: "INV-2026-001"}}
	return in
}

func TestWebhookRejectsBadToken(t *testing.T) {
	store := &stubStore{}
	svc := newService(store, &stubGateway{}, "rahasia")

	if _, err := svc.HandleWebhook(context.Background(), "salah", paidWebhook()); statusOf(err) != 401 {
		t.Fatalf("token salah harus 401, dapat %v", err)
	}
	if store.settleRef.called {
		t.Error("settle tidak boleh dipanggil dengan token salah")
	}
}

func TestWebhookUnconfiguredRejects(t *testing.T) {
	svc := newService(&stubStore{}, &stubGateway{}, "")
	if _, err := svc.HandleWebhook(context.Background(), "apa pun", paidWebhook()); statusOf(err) != 401 {
		t.Fatalf("token belum dikonfigurasi harus menolak, dapat %v", err)
	}
}

func TestWebhookIgnoresNonPaid(t *testing.T) {
	store := &stubStore{}
	svc := newService(store, &stubGateway{}, "rahasia")

	in := paidWebhook()
	in.PaymentInfo.Status = "PENDING"
	settled, err := svc.HandleWebhook(context.Background(), "rahasia", in)
	if err != nil || settled {
		t.Fatalf("event non-PAID harus diabaikan, dapat settled=%v err=%v", settled, err)
	}
	if store.settleRef.called {
		t.Error("settle tidak boleh dipanggil untuk event non-PAID")
	}
}

func TestWebhookSettlesPaid(t *testing.T) {
	store := &stubStore{settleReturns: true}
	svc := newService(store, &stubGateway{}, "rahasia")

	settled, err := svc.HandleWebhook(context.Background(), "rahasia", paidWebhook())
	if err != nil || !settled {
		t.Fatalf("PAID harus melunasi, dapat settled=%v err=%v", settled, err)
	}
	r := store.settleRef
	if !r.called || r.paperID != "pp-1" || r.number != "INV-2026-001" {
		t.Errorf("SettleByRef dapat argumen salah: %+v", r)
	}
	if r.amount != 1_500_000 {
		t.Errorf("amount harus dari paid_amount, dapat %d", r.amount)
	}
	if r.method != "bank_transfer:bni" {
		t.Errorf("method harus gabung method:channel, dapat %q", r.method)
	}
}

// --- delivery channels ------------------------------------------------------

// The operational default: issuing an invoice must actually reach the member.
// Before this, Send always passed false/false, so Paper.id created the invoice
// and delivered nothing — the member never heard about their bill.
func TestSendUsesDeliverySettings(t *testing.T) {
	store := &stubStore{
		sendable: draftSendable(),
		settings: map[string]string{
			sendEmailSettingKey:    "true",
			sendWhatsAppSettingKey: "true",
		},
	}
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1"}}
	svc := newService(store, gw, "tok")

	if _, err := svc.Send(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !gw.gotIn.SendEmail || !gw.gotIn.SendWhatsApp {
		t.Errorf("kanal pengiriman harus mengikuti app_settings: %+v", gw.gotIn)
	}
}

// Silence is the safe default: an unconfigured or staging environment must not
// message real members just because someone clicked Terbitkan.
func TestSendStaysSilentWithoutSettings(t *testing.T) {
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1"}}
	svc := newService(&stubStore{sendable: draftSendable()}, gw, "tok")

	if _, err := svc.Send(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gw.gotIn.SendEmail || gw.gotIn.SendWhatsApp {
		t.Errorf("tanpa setting harus diam: %+v", gw.gotIn)
	}
}

// An explicit flag beats the setting, so a one-off resend can stay quiet even
// when delivery is switched on globally.
func TestSendExplicitOptionsOverrideSettings(t *testing.T) {
	store := &stubStore{
		sendable: draftSendable(),
		settings: map[string]string{
			sendEmailSettingKey:    "true",
			sendWhatsAppSettingKey: "true",
		},
	}
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1"}}
	svc := newService(store, gw, "tok")

	off := false
	_, err := svc.Send(context.Background(), "inv-1", SendOptions{Email: &off, WhatsApp: &off})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gw.gotIn.SendEmail || gw.gotIn.SendWhatsApp {
		t.Errorf("flag eksplisit harus menang atas setting: %+v", gw.gotIn)
	}
}

// A member with no email address must not abort the send — WhatsApp still
// reaches them, and Paper.id would reject the empty address outright.
func TestSendDropsEmailChannelWhenMemberHasNoEmail(t *testing.T) {
	sendable := draftSendable()
	sendable.Email = ""
	store := &stubStore{
		sendable: sendable,
		settings: map[string]string{
			sendEmailSettingKey:    "true",
			sendWhatsAppSettingKey: "true",
		},
	}
	gw := &stubGateway{res: &CreateResult{PaperInvoiceID: "pp-1"}}
	svc := newService(store, gw, "tok")

	if _, err := svc.Send(context.Background(), "inv-1", SendOptions{}); err != nil {
		t.Fatalf("Send tidak boleh gagal hanya karena email kosong: %v", err)
	}
	if gw.gotIn.SendEmail {
		t.Error("email tidak boleh dinyalakan untuk member tanpa alamat")
	}
	if !gw.gotIn.SendWhatsApp {
		t.Error("WhatsApp harus tetap jalan")
	}
}
