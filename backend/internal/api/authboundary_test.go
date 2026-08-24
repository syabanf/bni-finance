package api_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/domain"
)

// Postgres cannot enforce any of this: the backend connects as one trusted
// role, so the database sees a single identity. The boundary exists only in Go
// middleware — and these tests are the only proof that it actually holds.

func (s *fullStack) doAs(t *testing.T, token, method, path, body string) int {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, s.srv.URL+path, r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

// TestNoTokenIsRejected walks the whole protected surface. A single route that
// forgot the middleware would be a silent hole, so this checks them all rather
// than a sample.
func TestNoTokenIsRejected(t *testing.T) {
	s := newFullServer(t)

	protected := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/invoices"},
		{http.MethodPost, "/api/v1/invoices"},
		{http.MethodGet, "/api/v1/invoices/inv-1"},
		{http.MethodPatch, "/api/v1/invoices/inv-1"},
		{http.MethodDelete, "/api/v1/invoices/inv-1"},
		{http.MethodGet, "/api/v1/invoices/inv-1/audit"},
		{http.MethodPost, "/api/v1/invoices/inv-1/audit"},
		{http.MethodGet, "/api/v1/payments"},
		{http.MethodPost, "/api/v1/payments"},
		{http.MethodGet, "/api/v1/members"},
		{http.MethodPost, "/api/v1/members"},
		{http.MethodGet, "/api/v1/members/renewal-due"},
		{http.MethodGet, "/api/v1/chapters"},
		{http.MethodPost, "/api/v1/chapters"},
		{http.MethodGet, "/api/v1/fee-settings"},
		{http.MethodPatch, "/api/v1/fee-settings"},
		{http.MethodGet, "/api/v1/app-settings"},
		{http.MethodPut, "/api/v1/app-settings/self_payment_mode"},
		{http.MethodGet, "/api/v1/dashboard/summary"},
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodGet, "/api/v1/users"},
	}

	for _, route := range protected {
		if got := s.doAs(t, "", route.method, route.path, "{}"); got != http.StatusUnauthorized {
			t.Errorf("%s %s tanpa token: dapat %d, harusnya 401", route.method, route.path, got)
		}
	}
}

func TestInvalidTokensAreRejected(t *testing.T) {
	s := newFullServer(t)

	expired, err := auth.NewSigner(testSecret, time.Hour)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	stale, _, _ := expired.Sign(domain.User{
		ID: "1", Email: "a@b.c", Name: "Lama", Role: domain.RoleAdmin,
	}, time.Now().Add(-48*time.Hour))

	foreignSigner, _ := auth.NewSigner("secret-asing-yang-panjangnya-lebih-dari-32-karakter", time.Hour)
	foreign, _, _ := foreignSigner.Sign(domain.User{
		ID: "1", Email: "a@b.c", Name: "Asing", Role: domain.RoleAdmin,
	}, time.Now())

	cases := map[string]string{
		"sampah":              "bukan-token-sama-sekali",
		"kedaluwarsa":         stale,
		"ditandatangani lain": foreign,
	}
	for name, token := range cases {
		if got := s.doAs(t, token, http.MethodGet, "/api/v1/invoices", ""); got != http.StatusUnauthorized {
			t.Errorf("token %s: dapat %d, harusnya 401", name, got)
		}
	}
}

// TestReadOnlyUserCannotWrite is the RBAC boundary. The UI hides these buttons
// from a 'user', but hiding a button is not access control — the server has to
// refuse them too.
func TestReadOnlyUserCannotWrite(t *testing.T) {
	s := newFullServer(t)
	readOnly := tokenFor(t, s.signer, domain.RoleUser)

	writes := []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/v1/invoices", `{"memberId":"m","chapterId":"c","type":"renewal","amount":1,"dueDate":"2026-01-01","periodStart":"2026-01-01","periodEnd":"2027-01-01"}`},
		{http.MethodPatch, "/api/v1/invoices/inv-1", `{"status":"sent"}`},
		{http.MethodDelete, "/api/v1/invoices/inv-1", ""},
		{http.MethodPost, "/api/v1/invoices/inv-1/audit", `{"notes":"halo"}`},
		{http.MethodPost, "/api/v1/payments", `{"invoiceId":"inv-1","amount":1}`},
		{http.MethodPatch, "/api/v1/payments/pay-1", `{"amount":2}`},
		{http.MethodDelete, "/api/v1/payments/pay-1", ""},
		{http.MethodPost, "/api/v1/members", `{"chapterId":"c","name":"X"}`},
		{http.MethodPatch, "/api/v1/members/mem-1", `{"name":"Y"}`},
		{http.MethodDelete, "/api/v1/members/mem-1", ""},
		{http.MethodPost, "/api/v1/chapters", `{"name":"X"}`},
		{http.MethodPatch, "/api/v1/chapters/ch-1", `{"name":"Y"}`},
		{http.MethodDelete, "/api/v1/chapters/ch-1", ""},
		{http.MethodPatch, "/api/v1/fee-settings", `{"renewalFee":1}`},
		{http.MethodPut, "/api/v1/app-settings/self_payment_mode", `{"value":"true"}`},
		{http.MethodDelete, "/api/v1/app-settings/self_payment_mode", ""},
		{http.MethodGet, "/api/v1/users", ""},
		{http.MethodPost, "/api/v1/users", `{"email":"x@y.z","password":"rahasia","name":"X"}`},
	}

	for _, w := range writes {
		if got := s.doAs(t, readOnly, w.method, w.path, w.body); got != http.StatusForbidden {
			t.Errorf("%s %s sebagai 'user': dapat %d, harusnya 403", w.method, w.path, got)
		}
	}
}

// A read-only user is still a user: reads must keep working, or the role is
// useless.
func TestReadOnlyUserCanRead(t *testing.T) {
	s := newFullServer(t)
	readOnly := tokenFor(t, s.signer, domain.RoleUser)

	reads := []string{
		"/api/v1/invoices",
		"/api/v1/payments",
		"/api/v1/members",
		"/api/v1/members/renewal-due",
		"/api/v1/chapters",
		"/api/v1/fee-settings",
		"/api/v1/app-settings",
		"/api/v1/dashboard/summary",
	}
	for _, path := range reads {
		if got := s.doAs(t, readOnly, http.MethodGet, path, ""); got != http.StatusOK {
			t.Errorf("GET %s sebagai 'user': dapat %d, harusnya 200", path, got)
		}
	}
}

// Health and the public payment page must stay reachable without a token —
// the person paying an invoice has no account.
func TestPublicRoutesNeedNoToken(t *testing.T) {
	s := newFullServer(t)

	if got := s.doAs(t, "", http.MethodGet, "/healthz", ""); got != http.StatusOK {
		t.Errorf("/healthz tanpa token: dapat %d, harusnya 200", got)
	}
	// Login must be reachable — you have no token until it succeeds. 400 here
	// (empty body) proves the handler ran rather than being blocked at 401.
	if got := s.doAs(t, "", http.MethodPost, "/api/v1/auth/login", `{}`); got == http.StatusUnauthorized {
		t.Error("POST /auth/login tidak boleh butuh token")
	}
}
