package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/api"
	"github.com/syabanf/bni-finance/backend/internal/audit"
	"github.com/syabanf/bni-finance/backend/internal/chapter"
	"github.com/syabanf/bni-finance/backend/internal/config"
	"github.com/syabanf/bni-finance/backend/internal/dashboard"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/invoice"
	"github.com/syabanf/bni-finance/backend/internal/member"
	"github.com/syabanf/bni-finance/backend/internal/payment"
	"github.com/syabanf/bni-finance/backend/internal/settings"
)

type fullStack struct {
	srv      *httptest.Server
	chapters *fakeChapterStore
	members  *fakeMemberStore
	settings *fakeSettingsStore
	audits   *fakeAuditStore
}

// newFullServer registers EVERY resource. Building it at all is part of the
// test: http.ServeMux panics on conflicting patterns, so a clashing route would
// fail here rather than in production.
func newFullServer(t *testing.T) *fullStack {
	t.Helper()

	invStore := newFakeInvoiceStore()
	stack := &fullStack{
		chapters: newFakeChapterStore(),
		members:  newFakeMemberStore(),
		settings: newFakeSettingsStore(),
		audits:   newFakeAuditStore(),
	}

	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := config.Config{AllowedOrigins: []string{"*"}}

	h := api.NewHandler(log, cfg, api.Services{
		Invoice:   invoice.NewService(invStore),
		Payment:   payment.NewService(newFakePaymentStore(invStore)),
		Chapter:   chapter.NewService(stack.chapters),
		Member:    member.NewService(stack.members),
		Settings:  settings.NewService(stack.settings),
		Audit:     audit.NewService(stack.audits),
		Dashboard: dashboard.NewService(fakeDashboardStore{}),
	}, nil)

	stack.srv = httptest.NewServer(h)
	t.Cleanup(stack.srv.Close)
	return stack
}

func (s *fullStack) do(t *testing.T, method, path, body string) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequest(method, s.srv.URL+path, reader)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, payload
}

func TestChapterAndMemberCRUD(t *testing.T) {
	s := newFullServer(t)

	// Chapter lifecycle.
	code, body := s.do(t, http.MethodPost, "/api/v1/chapters",
		`{"id":"ch-garuda","name":"Garuda","cityName":"Jakarta"}`)
	if code != http.StatusCreated {
		t.Fatalf("buat chapter: dapat %d — %s", code, body)
	}
	var created domain.Chapter
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode chapter: %v", err)
	}
	// displayName defaults to name when omitted.
	if created.DisplayName != "Garuda" {
		t.Errorf("displayName harusnya jatuh ke name, dapat %q", created.DisplayName)
	}

	if code, body = s.do(t, http.MethodPost, "/api/v1/chapters", `{"name":""}`); code != http.StatusBadRequest {
		t.Errorf("nama kosong harus 400, dapat %d — %s", code, body)
	}
	if code, _ = s.do(t, http.MethodGet, "/api/v1/chapters?cityName=Jakarta", ""); code != http.StatusOK {
		t.Errorf("daftar chapter: dapat %d", code)
	}
	if code, _ = s.do(t, http.MethodPatch, "/api/v1/chapters/ch-garuda", `{"cityName":"Bandung"}`); code != http.StatusOK {
		t.Errorf("ubah chapter: dapat %d", code)
	}
	if code, _ = s.do(t, http.MethodGet, "/api/v1/chapters/tidak-ada", ""); code != http.StatusNotFound {
		t.Errorf("chapter hilang harus 404, dapat %d", code)
	}

	// Member lifecycle.
	code, body = s.do(t, http.MethodPost, "/api/v1/members",
		`{"chapterId":"ch-garuda","name":"Budi","email":"budi@example.com","renewalDate":"2026-09-01"}`)
	if code != http.StatusCreated {
		t.Fatalf("buat member: dapat %d — %s", code, body)
	}
	var m domain.Member
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decode member: %v", err)
	}
	if m.Status != domain.MemberActive {
		t.Errorf("status default harus active, dapat %q", m.Status)
	}
	// A date-only column must stay YYYY-MM-DD on the wire.
	if !strings.Contains(string(body), `"renewalDate":"2026-09-01"`) {
		t.Errorf("renewalDate harus YYYY-MM-DD, dapat %s", body)
	}

	if code, body = s.do(t, http.MethodPost, "/api/v1/members", `{"name":"Tanpa Chapter"}`); code != http.StatusBadRequest {
		t.Errorf("chapterId kosong harus 400, dapat %d — %s", code, body)
	}
	if code, body = s.do(t, http.MethodGet, "/api/v1/members?status=ngawur", ""); code != http.StatusBadRequest {
		t.Errorf("status tak dikenal harus 400, dapat %d — %s", code, body)
	}
	if code, _ = s.do(t, http.MethodGet, "/api/v1/members/"+m.ID, ""); code != http.StatusOK {
		t.Errorf("ambil member: dapat %d", code)
	}
}

// TestRenewalDueNotShadowed guards the one genuinely fragile route: a literal
// path segment sitting next to /members/{id}.
func TestRenewalDueNotShadowed(t *testing.T) {
	s := newFullServer(t)

	s.do(t, http.MethodPost, "/api/v1/chapters", `{"id":"ch-1","name":"Satu"}`)
	s.do(t, http.MethodPost, "/api/v1/members",
		`{"chapterId":"ch-1","name":"Siti","renewalDate":"2026-08-01"}`)

	code, body := s.do(t, http.MethodGet, "/api/v1/members/renewal-due?days=30", "")
	if code != http.StatusOK {
		t.Fatalf("renewal-due dibayangi /members/{id}: dapat %d — %s", code, body)
	}

	var out struct {
		Data []domain.RenewalDueMember `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode renewal-due: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].DaysUntilDue == 0 {
		t.Errorf("harusnya 1 member jatuh tempo dengan daysUntilDue, dapat %+v", out.Data)
	}
	// The embedded Member must flatten, not nest under "member".
	if out.Data[0].Name != "Siti" {
		t.Errorf("field member harus ikut ter-flatten, dapat %s", body)
	}
}

// TestDeleteBlockedByDependents proves the 409 guards fire instead of letting a
// foreign key violation reach the client as a 500.
func TestDeleteBlockedByDependents(t *testing.T) {
	s := newFullServer(t)

	s.do(t, http.MethodPost, "/api/v1/chapters", `{"id":"ch-x","name":"X"}`)
	s.chapters.members = 4

	code, body := s.do(t, http.MethodDelete, "/api/v1/chapters/ch-x", "")
	if code != http.StatusConflict {
		t.Errorf("hapus chapter terpakai harus 409, dapat %d — %s", code, body)
	}

	s.chapters.members = 0
	if code, _ = s.do(t, http.MethodDelete, "/api/v1/chapters/ch-x", ""); code != http.StatusNoContent {
		t.Errorf("hapus chapter bebas harus 204, dapat %d", code)
	}

	code, body = s.do(t, http.MethodPost, "/api/v1/members", `{"chapterId":"ch-x","name":"Ani"}`)
	if code != http.StatusCreated {
		t.Fatalf("buat member: %d — %s", code, body)
	}
	var m domain.Member
	json.Unmarshal(body, &m)

	s.members.invoices = 2
	if code, body = s.do(t, http.MethodDelete, "/api/v1/members/"+m.ID, ""); code != http.StatusConflict {
		t.Errorf("hapus member berinvoice harus 409, dapat %d — %s", code, body)
	}
}

// TestSecretSettingsAreRedacted is the important one: app_settings holds the
// BNI VM token, so a credential must never travel back out of the API.
func TestSecretSettingsAreRedacted(t *testing.T) {
	s := newFullServer(t)

	const token = "rahasia-token-bni-vm"

	if code, body := s.do(t, http.MethodPut, "/api/v1/app-settings/bni_vm_token",
		`{"value":"`+token+`"}`); code != http.StatusOK {
		t.Fatalf("simpan token: dapat %d — %s", code, body)
	} else if strings.Contains(string(body), token) {
		t.Errorf("respons tulis membocorkan token: %s", body)
	}

	_, body := s.do(t, http.MethodGet, "/api/v1/app-settings/bni_vm_token", "")
	if strings.Contains(string(body), token) {
		t.Errorf("GET membocorkan token: %s", body)
	}
	if !strings.Contains(string(body), domain.MaskedValue) {
		t.Errorf("token harusnya tersamar, dapat %s", body)
	}

	_, body = s.do(t, http.MethodGet, "/api/v1/app-settings", "")
	if strings.Contains(string(body), token) {
		t.Errorf("daftar membocorkan token: %s", body)
	}

	// The stored value must still be the real one — masking is presentation.
	stored, err := s.settings.GetApp(t.Context(), "bni_vm_token")
	if err != nil {
		t.Fatalf("ambil dari store: %v", err)
	}
	if stored.Value != token {
		t.Errorf("nilai tersimpan berubah: %q", stored.Value)
	}

	// A non-secret flag stays readable.
	s.do(t, http.MethodPut, "/api/v1/app-settings/self_payment_mode", `{"value":"true"}`)
	_, body = s.do(t, http.MethodGet, "/api/v1/app-settings/self_payment_mode", "")
	if !strings.Contains(string(body), `"value":"true"`) {
		t.Errorf("flag biasa harus terbaca apa adanya, dapat %s", body)
	}

	// Writing back the mask would silently destroy the real token.
	if code, _ := s.do(t, http.MethodPut, "/api/v1/app-settings/bni_vm_token",
		`{"value":"`+domain.MaskedValue+`"}`); code != http.StatusBadRequest {
		t.Errorf("menulis ulang nilai tersamar harus 400, dapat %d", code)
	}
}

func TestFeeSettingsAndDashboard(t *testing.T) {
	s := newFullServer(t)

	if code, body := s.do(t, http.MethodGet, "/api/v1/fee-settings", ""); code != http.StatusOK {
		t.Fatalf("ambil fee settings: %d — %s", code, body)
	}
	if code, body := s.do(t, http.MethodPatch, "/api/v1/fee-settings",
		`{"registrationFee":-1}`); code != http.StatusBadRequest {
		t.Errorf("biaya negatif harus 400, dapat %d — %s", code, body)
	}
	if code, body := s.do(t, http.MethodPatch, "/api/v1/fee-settings", `{}`); code != http.StatusBadRequest {
		t.Errorf("patch kosong harus 400, dapat %d — %s", code, body)
	}

	code, body := s.do(t, http.MethodPatch, "/api/v1/fee-settings", `{"renewalFee":2000000}`)
	if code != http.StatusOK {
		t.Fatalf("ubah fee: %d — %s", code, body)
	}
	var fees domain.FeeSettings
	json.Unmarshal(body, &fees)
	if fees.RenewalFee != 2_000_000 {
		t.Errorf("renewalFee harusnya 2000000, dapat %d", fees.RenewalFee)
	}

	code, body = s.do(t, http.MethodGet, "/api/v1/dashboard/summary?months=3", "")
	if code != http.StatusOK {
		t.Fatalf("ringkasan dashboard: %d — %s", code, body)
	}
	var sum domain.DashboardSummary
	if err := json.Unmarshal(body, &sum); err != nil {
		t.Fatalf("decode ringkasan: %v", err)
	}
	if len(sum.Monthly) != 3 {
		t.Errorf("months=3 harus menghasilkan 3 titik, dapat %d", len(sum.Monthly))
	}
	if sum.Total.Count != 3 || sum.RenewalDue.Count != 4 {
		t.Errorf("ringkasan tidak sesuai: %+v", sum)
	}
}

func TestAuditTimeline(t *testing.T) {
	s := newFullServer(t)
	s.audits.known["inv-1"] = true

	// A note on an unknown invoice is a 404, not a foreign-key 500.
	if code, _ := s.do(t, http.MethodPost, "/api/v1/invoices/inv-hantu/audit",
		`{"notes":"halo"}`); code != http.StatusNotFound {
		t.Errorf("audit invoice tak dikenal harus 404, dapat %d", code)
	}
	if code, body := s.do(t, http.MethodPost, "/api/v1/invoices/inv-1/audit",
		`{"actorName":"Admin"}`); code != http.StatusBadRequest {
		t.Errorf("catatan kosong harus 400, dapat %d — %s", code, body)
	}

	code, body := s.do(t, http.MethodPost, "/api/v1/invoices/inv-1/audit",
		`{"notes":"ditelepon, janji bayar Jumat","actorName":"Admin"}`)
	if code != http.StatusCreated {
		t.Fatalf("tambah catatan: %d — %s", code, body)
	}

	code, body = s.do(t, http.MethodGet, "/api/v1/invoices/inv-1/audit", "")
	if code != http.StatusOK {
		t.Fatalf("baca timeline: %d — %s", code, body)
	}
	var out struct {
		Data []domain.AuditEntry `json:"data"`
	}
	json.Unmarshal(body, &out)
	if len(out.Data) != 1 || out.Data[0].Action != domain.AuditUpdated {
		t.Errorf("timeline tidak sesuai: %+v", out.Data)
	}
}

// TestEmptyListIsArray keeps clients from having to handle `"data": null`.
func TestEmptyListIsArray(t *testing.T) {
	s := newFullServer(t)
	for _, path := range []string{
		"/api/v1/chapters", "/api/v1/members", "/api/v1/app-settings",
		"/api/v1/members/renewal-due",
	} {
		_, body := s.do(t, http.MethodGet, path, "")
		if !strings.Contains(string(body), `"data":[]`) {
			t.Errorf("%s: daftar kosong harus [] bukan null — %s", path, body)
		}
	}
}
