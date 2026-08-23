package paperid

import (
	"context"
	"crypto/sha256"
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

// dueDaysSettingKey mirrors the frontend: the due date is issue date + N days.
const dueDaysSettingKey = "invoice_due_days_after"
const defaultDueDays = 30

type Store interface {
	GetSendable(ctx context.Context, invoiceID string) (*Sendable, error)
	MarkSent(ctx context.Context, invoiceID string, res CreateResult, dueDate, sentAt time.Time, actor string) (*domain.Invoice, error)
	MarkReminded(ctx context.Context, invoiceID string, res CreateResult, at time.Time, actor string) (*domain.Invoice, error)
	ReserveReminder(ctx context.Context, invoiceID string) (int, error)
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
// recordInbound menyimpan callback APA ADANYA.
//
// Dulu yang direkam adalah json.Marshal atas struct HASIL PARSE — yang kita
// pahami, bukan yang Paper.id kirim. Kalau formatnya berbeda, blackbox
// menampilkan struct nyaris kosong dan justru MENYESATKAN orang yang sedang
// mendiagnosis: alat yang dipakai untuk menemukan ketidakcocokan format buta
// terhadap ketidakcocokan format.
//
// Sekarang body mentah yang disimpan, beserta catatan selisihnya terhadap
// bentuk yang kita harapkan.
func (s *Service) recordInbound(raw []byte, status int, success bool, err error) {
	if s.rec == nil {
		return
	}
	notes := formatNotes(inspectPayload(raw))
	var resp []byte
	if notes != "" {
		resp, _ = json.Marshal(struct {
			Catatan string `json:"catatanFormat"`
		}{notes})
	}
	s.rec.Record(blackbox.Call{
		Integration: "paper_id", Direction: blackbox.Inbound,
		Method: http.MethodPost, URL: "/api/v1/webhooks/paperid",
		Request: raw, Response: resp, Status: status, Success: success, Err: err,
	})
}

// Keys controlling how a sent invoice reaches the member.
//
// Absent means ON. Delivering the invoice is the point of issuing one — an
// invoice the member never receives is not a milder outcome, it is a silent
// failure that still reports success. Only an explicit "false" turns a channel
// off, so switching one off is a decision someone made, never something that
// happened because a settings row was missing.
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
	// Only a literal "false" disables a channel. A missing row, an empty value,
	// or a failed read all mean "nobody turned this off", and the invoice still
	// has to reach the member.
	enabled := func(key string) bool {
		v, err := s.repo.GetSetting(ctx, key)
		if err != nil {
			return true
		}
		return strings.TrimSpace(v) != "false"
	}
	if opts.Email != nil {
		email = *opts.Email
	} else {
		email = enabled(sendEmailSettingKey)
	}
	if opts.WhatsApp != nil {
		whatsapp = *opts.WhatsApp
	} else {
		whatsapp = enabled(sendWhatsAppSettingKey)
	}
	return email, whatsapp
}

// recordSend mencatat HASIL OPERASI pengiriman — bukan panggilan HTTP-nya.
//
// Keduanya perlu, dan bedanya penting. client.go merekam percakapan dengan
// Paper.id; entri ini merekam apa yang terjadi pada permintaan bisnisnya. Tanpa
// entri ini, setiap kegagalan yang terjadi SEBELUM panggilan keluar tidak
// meninggalkan jejak apa pun di blackbox:
//
//   - gateway belum dikonfigurasi          -> 503, tidak pernah menyentuh HTTP
//   - invoice tidak ditemukan / galat DB   -> tidak pernah menyentuh HTTP
//   - invoice bukan draft                  -> 409, tidak pernah menyentuh HTTP
//
// Dari dashboard, ketiganya tampak sebagai "kirim gagal" tetapi halaman
// blackbox kosong — persis keluhan yang membuat perbaikan ini ada. Blackbox
// dipakai untuk menjawab "tadi kenapa gagal", jadi diam bukan jawaban.
func (s *Service) recordSend(invoiceID string, opts SendOptions, started time.Time,
	out *domain.Invoice, err error) {
	if s.rec == nil {
		return
	}
	req, _ := json.Marshal(struct {
		InvoiceID string `json:"invoiceId"`
		Email     *bool  `json:"sendEmail"`
		WhatsApp  *bool  `json:"sendWhatsApp"`
	}{invoiceID, opts.Email, opts.WhatsApp})

	// Entri tanpa response setengah berguna: halaman blackbox menampilkan
	// permintaannya lalu berhenti, dan pertanyaan "hasilnya apa" tetap tak
	// terjawab — padahal itu justru alasan orang membukanya.
	//
	// Pada kegagalan, isinya pesan yang benar-benar diterima pemanggil. Pada
	// keberhasilan, ringkasan hasil yang bisa ditindaklanjuti: nomor, status,
	// dan tautan Paper.id — bukan seluruh invoice, karena entri ini tentang
	// operasinya, bukan salinan basis data.
	var resp []byte
	switch {
	case err != nil:
		resp, _ = json.Marshal(struct {
			Error string `json:"error"`
		}{err.Error()})
	case out != nil:
		resp, _ = json.Marshal(struct {
			ID         string               `json:"id"`
			Number     string               `json:"number"`
			Status     domain.InvoiceStatus `json:"status"`
			PaperID    *string              `json:"paperIdInvoiceId,omitempty"`
			PaymentURL *string              `json:"paperIdPaymentUrl,omitempty"`
		}{out.ID, out.Number, out.Status, out.PaperIDInvoiceID, out.PaperIDPaymentURL})
	}

	s.rec.Record(blackbox.Call{
		Integration: "paper_id", Direction: blackbox.Outbound,
		Method: http.MethodPost, URL: "/api/v1/invoices/" + invoiceID + "/send",
		Request: req, Response: resp, Status: httpx.StatusOf(err), Success: err == nil,
		Duration: time.Since(started), Err: err,
	})
}

// Send pushes a draft invoice to Paper.id and records the result.
//
// Nilai balik diberi nama supaya defer di bawah melihat error dari JALUR MANA
// PUN — termasuk return awal yang tidak pernah mencapai gateway. Menaruh
// pemanggilan Record di setiap titik return akan bekerja hari ini dan bocor
// lagi pada return berikutnya yang ditambahkan orang.
func (s *Service) Send(ctx context.Context, invoiceID string, opts SendOptions) (out *domain.Invoice, err error) {
	started := s.now()
	defer func() { s.recordSend(invoiceID, opts, started, out, err) }()

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
		CustomerID:    customerRef(inv),
		CustomerName:  inv.Name,
		CustomerEmail: inv.Email,
		CustomerPhone: normalizePhone(inv.Phone),
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
func (s *Service) HandleWebhook(ctx context.Context, token string, raw []byte) (settled bool, err error) {
	if s.callbackToken == "" {
		err := httpx.Unauthorized("callback Paper.id belum dikonfigurasi")
		s.recordInbound(raw, http.StatusUnauthorized, false, err)
		return false, err
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.callbackToken)) != 1 {
		err := httpx.Unauthorized("token callback tidak valid")
		s.recordInbound(raw, http.StatusUnauthorized, false, err)
		return false, err
	}

	var in WebhookInput
	if err := json.Unmarshal(raw, &in); err != nil {
		e := httpx.BadRequest("payload callback bukan JSON yang sah")
		s.recordInbound(raw, http.StatusBadRequest, false, e)
		return false, e
	}

	// Only a completed payment settles; other events are acknowledged (200) but
	// change nothing.
	status := strings.ToUpper(strings.TrimSpace(in.PaymentInfo.Status))
	if status != "PAID" && status != "SETTLED" && status != "SUCCESS" && status != "SUCCEEDED" {
		// Acknowledged but not acted on — still worth recording.
		s.recordInbound(raw, http.StatusOK, true, nil)
		return false, nil
	}

	var uuid, number string
	if len(in.AdditionalInfo.Invoices) > 0 {
		uuid = in.AdditionalInfo.Invoices[0].UUID
		number = in.AdditionalInfo.Invoices[0].Number
	}
	if uuid == "" && number == "" {
		err := httpx.BadRequest("callback tidak memuat invoice yang bisa dicocokkan")
		s.recordInbound(raw, http.StatusBadRequest, false, err)
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
		s.recordInbound(raw, http.StatusInternalServerError, false, err)
		return false, err
	}
	s.recordInbound(raw, http.StatusOK, true, nil)
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
		if isPartnerMismatch(ae) {
			// Should be unreachable now that customerRef changes with the
			// details, but the message is worth keeping: on its own the
			// upstream text gives no hint that contact data is the cause.
			return httpx.Conflict(
				"Paper.id sudah menyimpan member ini dengan nama/email/telepon yang berbeda — " +
					"samakan datanya, atau hubungi Paper.id untuk memperbarui kontaknya")
		}
		return httpx.NewError(http.StatusBadGateway, "Paper.id menolak: "+ae.Message, err)
	}
	return httpx.NewError(http.StatusBadGateway, "tidak bisa menghubungi Paper.id", err)
}

// isPartnerMismatch spots Paper.id refusing a customer.id whose stored details
// differ from the ones we just sent.
func isPartnerMismatch(ae *apiError) bool {
	return strings.Contains(strings.ToLower(ae.Message), "partner doesn't match")
}

func isDuplicateNumber(ae *apiError) bool {
	m := strings.ToLower(ae.Message)
	return strings.Contains(m, "number sudah dipakai") ||
		strings.Contains(m, "number already") ||
		strings.Contains(m, "already used") ||
		strings.Contains(m, "duplicate")
}

// customerRef builds the `customer.id` Paper.id stores the member under.
//
// Paper.id binds that id to the customer's details the first time it sees it.
// Sending the SAME id later with a different name, email, or phone is rejected
// with "Failed partner doesn't match" — a 400 that says nothing about contact
// details being the cause.
//
// The member id alone therefore cannot be the reference: BNI VM sync updates
// names and phone numbers routinely, and the first such update would break
// every future invoice for that member, permanently and opaquely. Proven
// against staging: the same member id succeeds with its original phone and
// fails with a new one.
//
// So the reference is the member id plus a short digest of the details Paper.id
// binds. Unchanged details reuse the existing customer, exactly as before; the
// moment they change, the id changes too and Paper.id registers a new customer
// instead of refusing the invoice. A member who changes phone ends up with a
// second contact upstream — cheaper than an invoice that can never be sent.
func customerRef(inv *Sendable) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(inv.Name)) + "\x00" +
		strings.ToLower(strings.TrimSpace(inv.Email)) + "\x00" +
		strings.TrimSpace(inv.Phone)))
	return inv.MemberID + "-" + hex.EncodeToString(sum[:4])
}

// --- pengingat ----------------------------------------------------------------

// Remind mengirim ULANG invoice yang sudah diterbitkan sebagai pengingat.
//
// Paper.id tidak punya endpoint pengingat. Diperiksa langsung terhadap API
// staging, dengan probe yang dikalibrasi lebih dulu (store-invoice menjawab 400
// karena validasi, path ngawur menjawab 404): kedelapan kandidat nama endpoint
// pengingat menjawab 404. Jadi satu-satunya cara membuat Paper.id mengantar
// pesan lagi adalah menerbitkan dokumen lain.
//
// Nomornya harus berbeda. Paper.id membakar nomor secara permanen; mengirim
// ulang dengan nomor yang sama ditolak "nomor sudah dipakai". Pengingat karena
// itu memakai nomor turunan — INV-2026-001-R1, -R2 — sementara nomor kanonik di
// sistem kita tidak berubah, sehingga rekonsiliasi tetap menunjuk satu tagihan.
//
// KONSEKUENSI YANG HARUS DISADARI: setiap pengingat membuat DOKUMEN BARU di
// Paper.id untuk tagihan yang sama. Member akan melihat beberapa invoice dengan
// nominal identik di riwayat Paper.id mereka. Itu harga dari "pengingat lewat
// Paper.id" selama endpoint pengingatnya belum ada, dan bukan sesuatu yang bisa
// disembunyikan di sisi kita.
func (s *Service) Remind(ctx context.Context, invoiceID string, opts SendOptions) (_ *domain.Invoice, err error) {
	started := s.now()
	defer func() { s.recordRemind(invoiceID, opts, started, err) }()

	if s.gateway == nil {
		return nil, httpx.NewError(http.StatusServiceUnavailable,
			"Paper.id belum dikonfigurasi — isi PAPER_ID_CLIENT_ID & PAPER_ID_CLIENT_SECRET", nil)
	}

	inv, err := s.repo.GetSendable(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	// Pengingat hanya masuk akal untuk tagihan yang sudah dikirim dan belum
	// dibayar. Draft belum pernah sampai ke member — itu Send, bukan pengingat.
	// Lunas dan batal adalah catatan tertutup; mengingatkannya berarti menagih
	// orang atas sesuatu yang tidak mereka hutangi.
	switch inv.Status {
	case domain.StatusSent, domain.StatusOverdue:
	case domain.StatusDraft:
		return nil, httpx.Conflict("invoice belum pernah dikirim — pakai Kirim, bukan Pengingat")
	default:
		return nil, httpx.Conflict(
			"invoice berstatus " + string(inv.Status) + " tidak bisa diingatkan")
	}

	if strings.TrimSpace(inv.Phone) == "" {
		return nil, httpx.BadRequest("member belum punya nomor telepon — Paper.id mewajibkannya")
	}

	now := s.now()
	sendEmail, sendWhatsApp := s.resolve(ctx, opts)
	if strings.TrimSpace(inv.Email) == "" {
		sendEmail = false
	}

	// Nomor urut DIPESAN dan diikat sebelum Paper.id dihubungi. Lihat
	// ReserveReminder: menaikkannya sesudah membuat setiap kegagalan meracuni
	// sufiksnya secara permanen.
	seq, err := s.repo.ReserveReminder(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	// Jatuh tempo TIDAK dihitung ulang dari hari ini. Pengingat atas tagihan
	// yang sudah lewat jatuh tempo harus tetap menampilkan tanggal aslinya —
	// memundurkannya akan membuat tunggakan tampak belum jatuh tempo.
	res, err := s.gateway.CreateInvoice(ctx, CreateInput{
		Number:        reminderNumber(inv.Number, seq),
		InvoiceDate:   now,
		DueDate:       inv.DueDate,
		Amount:        inv.Amount,
		ItemName:      itemName(inv.Type),
		ItemDesc:      itemDesc(inv.Type),
		CustomerID:    customerRef(inv),
		CustomerName:  inv.Name,
		CustomerEmail: inv.Email,
		CustomerPhone: normalizePhone(inv.Phone),
		SendEmail:     sendEmail,
		SendWhatsApp:  sendWhatsApp,
		Notes:         "Pengingat pembayaran untuk invoice " + inv.Number,
	})
	if err != nil {
		return nil, gatewayError(err)
	}
	return s.repo.MarkReminded(ctx, invoiceID, *res, now, "sistem")
}

// reminderNumber menurunkan nomor dokumen pengingat dari nomor kanonik.
func reminderNumber(base string, n int) string {
	return fmt.Sprintf("%s-R%d", base, n)
}

// recordRemind mencerminkan recordSend: setiap hasil, berhasil maupun gagal,
// meninggalkan satu entri di blackbox lengkap dengan request dan response.
func (s *Service) recordRemind(invoiceID string, opts SendOptions, started time.Time, err error) {
	if s.rec == nil {
		return
	}
	req, _ := json.Marshal(struct {
		InvoiceID string `json:"invoiceId"`
		Email     *bool  `json:"sendEmail"`
		WhatsApp  *bool  `json:"sendWhatsApp"`
	}{invoiceID, opts.Email, opts.WhatsApp})

	var resp []byte
	if err != nil {
		resp, _ = json.Marshal(struct {
			Error string `json:"error"`
		}{err.Error()})
	} else {
		resp, _ = json.Marshal(struct {
			Status string `json:"status"`
		}{"pengingat dikirim ulang ke Paper.id"})
	}

	s.rec.Record(blackbox.Call{
		Integration: "paper_id", Direction: blackbox.Outbound,
		Method: http.MethodPost, URL: "/api/v1/invoices/" + invoiceID + "/remind",
		Request: req, Response: resp, Status: httpx.StatusOf(err), Success: err == nil,
		Duration: time.Since(started), Err: err,
	})
}
