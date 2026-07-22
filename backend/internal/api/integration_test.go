package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

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

// The tests above run against in-memory fakes, which by construction cannot
// catch a malformed query: SQL is a string until Postgres parses it. Two real
// bugs slipped through exactly that way — a FILTER attached to coalesce() instead
// of to the aggregate, and a parameter Postgres inferred as TEXT because it was
// concatenated into an interval. Both were syntactically fine Go.
//
// So this file talks to a real database. It is skipped unless TEST_DATABASE_URL
// is set:
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
//
// It TRUNCATEs the tables it uses, so it refuses any database whose name doesn't
// look disposable.

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL tidak diset — integration test dilewati")
	}

	// This test wipes tables. Never let it point at something that isn't
	// obviously a throwaway database.
	name := databaseName(url)
	if !strings.Contains(name, "test") && !strings.Contains(name, "dev") {
		t.Fatalf("TEST_DATABASE_URL menunjuk ke %q — test ini menghapus data, "+
			"jadi nama database wajib mengandung 'test' atau 'dev'", name)
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("sambung ke database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	t.Cleanup(pool.Close)

	_, err = pool.Exec(context.Background(),
		"TRUNCATE invoice_audit_log, payments, invoices, members, chapters RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("bersihkan tabel: %v", err)
	}
	return pool
}

func databaseName(url string) string {
	if i := strings.LastIndex(url, "/"); i >= 0 {
		name := url[i+1:]
		if q := strings.Index(name, "?"); q >= 0 {
			name = name[:q]
		}
		return name
	}
	return url
}

type liveStack struct {
	srv *httptest.Server
}

func newLiveServer(t *testing.T) *liveStack {
	t.Helper()
	pool := integrationPool(t)

	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	h := api.NewHandler(log, config.Config{AllowedOrigins: []string{"*"}}, api.Services{
		Invoice:   invoice.NewService(invoice.NewRepository(pool)),
		Payment:   payment.NewService(payment.NewRepository(pool)),
		Member:    member.NewService(member.NewRepository(pool)),
		Chapter:   chapter.NewService(chapter.NewRepository(pool)),
		Settings:  settings.NewService(settings.NewRepository(pool)),
		Audit:     audit.NewService(audit.NewRepository(pool)),
		Dashboard: dashboard.NewService(dashboard.NewRepository(pool)),
	}, pool.Ping)

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &liveStack{srv: srv}
}

func (s *liveStack) req(t *testing.T, method, path, body string, want int) []byte {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, s.srv.URL+path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: dapat HTTP %d, harusnya %d — %s", method, path, resp.StatusCode, want, payload)
	}
	return payload
}

// TestLiveSQLPaths walks every query the fakes cannot reach. Its job is to make
// Postgres parse and execute each statement at least once.
func TestLiveSQLPaths(t *testing.T) {
	s := newLiveServer(t)

	s.req(t, http.MethodPost, "/api/v1/chapters",
		`{"id":"ch-it","name":"Integrasi","displayName":"BNI Integrasi","cityName":"Jakarta"}`,
		http.StatusCreated)

	// renewal_date must land inside the 30-day window the query asks for.
	memberBody := s.req(t, http.MethodPost, "/api/v1/members",
		`{"chapterId":"ch-it","name":"Uji Integrasi","email":"uji@example.com","renewalDate":"`+
			inDays(14)+`"}`, http.StatusCreated)

	var m domain.Member
	if err := json.Unmarshal(memberBody, &m); err != nil {
		t.Fatalf("decode member: %v", err)
	}
	// The LEFT JOIN must actually hydrate the chapter.
	if m.Chapter == nil || m.Chapter.DisplayName != "BNI Integrasi" {
		t.Errorf("chapter harus ikut ter-join, dapat %+v", m.Chapter)
	}

	// A parameter concatenated into an interval is inferred as TEXT — the bug
	// this call exists to catch.
	renewalBody := s.req(t, http.MethodGet, "/api/v1/members/renewal-due?days=30", "", http.StatusOK)
	var renewal struct {
		Data []domain.RenewalDueMember `json:"data"`
	}
	json.Unmarshal(renewalBody, &renewal)
	if len(renewal.Data) != 1 {
		t.Fatalf("harusnya 1 member jatuh tempo, dapat %d — %s", len(renewal.Data), renewalBody)
	}
	if renewal.Data[0].DaysUntilDue <= 0 || renewal.Data[0].DaysUntilDue > 30 {
		t.Errorf("daysUntilDue di luar jangkauan: %d", renewal.Data[0].DaysUntilDue)
	}

	// Every member filter, so each dynamic WHERE branch gets parsed.
	s.req(t, http.MethodGet,
		"/api/v1/members?chapterId=ch-it&status=active&q=Uji&renewalFrom=2020-01-01&renewalTo=2030-01-01",
		"", http.StatusOK)
	s.req(t, http.MethodGet, "/api/v1/chapters?q=Integrasi&cityName=Jakarta&areaName=", "", http.StatusOK)

	// Invoice lifecycle — Create and Update both run inside a transaction that
	// also writes the audit row.
	invoiceBody := s.req(t, http.MethodPost, "/api/v1/invoices",
		`{"memberId":"`+m.ID+`","chapterId":"ch-it","type":"renewal","amount":1500000,`+
			`"dueDate":"2026-08-01","periodStart":"2026-08-01","periodEnd":"2027-08-01","createdBy":"uji"}`,
		http.StatusCreated)
	var inv domain.Invoice
	json.Unmarshal(invoiceBody, &inv)
	if inv.Number == "" {
		t.Error("nomor invoice harus dibuat otomatis")
	}

	assertAudit(t, s, inv.ID, 1, domain.AuditCreated)

	s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID,
		`{"status":"sent","actorName":"Admin","actorId":"adm-1"}`, http.StatusOK)
	assertAudit(t, s, inv.ID, 2, domain.AuditSent)

	// Illegal transitions are refused before any SQL runs.
	s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID, `{"status":"draft"}`, http.StatusConflict)

	// Every invoice filter branch.
	s.req(t, http.MethodGet,
		"/api/v1/invoices?status=outstanding&type=renewal&chapterId=ch-it&memberId="+m.ID+
			"&q=INV&dueFrom=2020-01-01&dueTo=2030-01-01&issuedFrom=2020-01-01&issuedTo=2030-01-01",
		"", http.StatusOK)

	// Settling writes the payment, flips the invoice and appends to the
	// timeline — all in one transaction, across two packages.
	s.req(t, http.MethodPost, "/api/v1/payments",
		`{"invoiceId":"`+inv.ID+`","amount":1500000,"paymentMethod":"bank_transfer","note":"uji"}`,
		http.StatusCreated)

	paid := s.req(t, http.MethodGet, "/api/v1/invoices/"+inv.ID, "", http.StatusOK)
	json.Unmarshal(paid, &inv)
	if inv.Status != domain.StatusPaid {
		t.Errorf("invoice harus lunas setelah pembayaran, dapat %s", inv.Status)
	}
	assertAudit(t, s, inv.ID, 3, domain.AuditPaid)

	s.req(t, http.MethodGet,
		"/api/v1/payments?invoiceId="+inv.ID+"&method=bank_transfer&paidFrom=2020-01-01&paidTo=2030-01-01",
		"", http.StatusOK)

	// An invoice with payments can't be deleted, and neither can the chapter or
	// member behind it — 409, not a foreign-key 500.
	s.req(t, http.MethodDelete, "/api/v1/invoices/"+inv.ID, "", http.StatusConflict)
	s.req(t, http.MethodDelete, "/api/v1/chapters/ch-it", "", http.StatusConflict)
	s.req(t, http.MethodDelete, "/api/v1/members/"+m.ID, "", http.StatusConflict)

	// Malformed uuid must not surface as a 500.
	s.req(t, http.MethodGet, "/api/v1/invoices/bukan-uuid", "", http.StatusNotFound)

	// Settings: singleton update plus the key/value upsert.
	s.req(t, http.MethodGet, "/api/v1/fee-settings", "", http.StatusOK)
	s.req(t, http.MethodPatch, "/api/v1/fee-settings",
		`{"registrationFee":1750000,"renewalFee":1500000,"currency":"IDR","notes":"uji","updatedBy":"adm"}`,
		http.StatusOK)
	s.req(t, http.MethodPut, "/api/v1/app-settings/self_payment_mode", `{"value":"true"}`, http.StatusOK)
	s.req(t, http.MethodGet, "/api/v1/app-settings", "", http.StatusOK)

	// The five aggregate queries — the other bug lived here.
	summaryBody := s.req(t, http.MethodGet, "/api/v1/dashboard/summary?months=6", "", http.StatusOK)
	var sum domain.DashboardSummary
	if err := json.Unmarshal(summaryBody, &sum); err != nil {
		t.Fatalf("decode ringkasan: %v", err)
	}
	if len(sum.Monthly) != 6 {
		t.Errorf("months=6 harus menghasilkan 6 titik, dapat %d", len(sum.Monthly))
	}
	if sum.Paid.Count != 1 || sum.Paid.Amount != 1_500_000 {
		t.Errorf("bucket lunas salah: %+v", sum.Paid)
	}
	if sum.Total.Count != 1 || sum.Total.Amount != 1_500_000 {
		t.Errorf("bucket total salah: %+v", sum.Total)
	}
	if sum.RenewalDue.Count != 1 {
		t.Errorf("renewalDue harusnya 1, dapat %d", sum.RenewalDue.Count)
	}
	if len(sum.ChapterStats) != 1 || sum.ChapterStats[0].TotalAmount != 1_500_000 {
		t.Errorf("statistik chapter salah: %+v", sum.ChapterStats)
	}
}

// TestLiveConcurrentSettle is the transaction test that matters: many settles on
// one invoice, against a database that really does row locking.
func TestLiveConcurrentSettle(t *testing.T) {
	s := newLiveServer(t)

	s.req(t, http.MethodPost, "/api/v1/chapters", `{"id":"ch-race","name":"Balapan"}`, http.StatusCreated)
	memberBody := s.req(t, http.MethodPost, "/api/v1/members",
		`{"chapterId":"ch-race","name":"Rekan Balap"}`, http.StatusCreated)
	var m domain.Member
	json.Unmarshal(memberBody, &m)

	invoiceBody := s.req(t, http.MethodPost, "/api/v1/invoices",
		`{"memberId":"`+m.ID+`","chapterId":"ch-race","type":"registration","amount":2000000,`+
			`"dueDate":"2026-09-01","periodStart":"2026-09-01","periodEnd":"2027-09-01"}`,
		http.StatusCreated)
	var inv domain.Invoice
	json.Unmarshal(invoiceBody, &inv)

	const concurrency = 32
	errs := make(chan error, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			resp, err := http.Post(s.srv.URL+"/api/v1/payments", "application/json",
				strings.NewReader(`{"invoiceId":"`+inv.ID+`","amount":2000000,"paymentMethod":"qris"}`))
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode >= 500 {
				errs <- &httpError{resp.StatusCode, string(body)}
				return
			}
			errs <- nil
		}()
	}
	for i := 0; i < concurrency; i++ {
		if err := <-errs; err != nil {
			t.Errorf("pelunasan paralel gagal: %v", err)
		}
	}

	final := s.req(t, http.MethodGet, "/api/v1/invoices/"+inv.ID, "", http.StatusOK)
	json.Unmarshal(final, &inv)
	if inv.Status != domain.StatusPaid {
		t.Errorf("invoice harus lunas, dapat %s", inv.Status)
	}
	// COALESCE keeps the first settlement's amount — 32 settles must not stack.
	if inv.PaidAmount == nil || *inv.PaidAmount != 2_000_000 {
		t.Errorf("paidAmount harus 2000000, dapat %v", inv.PaidAmount)
	}

	// The row lock means exactly one settle wins, so exactly one 'paid' entry.
	entries := auditEntries(t, s, inv.ID)
	var paidCount int
	for _, e := range entries {
		if e.Action == domain.AuditPaid {
			paidCount++
		}
	}
	if paidCount != 1 {
		t.Errorf("harus tepat 1 entri audit 'paid' walau %d pelunasan paralel, dapat %d",
			concurrency, paidCount)
	}
}

// inDays is a date N days from today, in the wire format the API expects.
// Relative rather than hardcoded, so the renewal window stays valid over time.
func inDays(n int) string {
	return time.Now().AddDate(0, 0, n).Format(domain.DateLayout)
}

type httpError struct {
	status int
	body   string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.status, e.body)
}

func auditEntries(t *testing.T, s *liveStack, invoiceID string) []domain.AuditEntry {
	t.Helper()
	body := s.req(t, http.MethodGet, "/api/v1/invoices/"+invoiceID+"/audit?limit=200", "", http.StatusOK)
	var out struct {
		Data []domain.AuditEntry `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode audit: %v", err)
	}
	return out.Data
}

func assertAudit(t *testing.T, s *liveStack, invoiceID string, wantCount int, wantNewest domain.AuditAction) {
	t.Helper()
	entries := auditEntries(t, s, invoiceID)
	if len(entries) != wantCount {
		t.Fatalf("jumlah entri audit: dapat %d, harusnya %d — %+v", len(entries), wantCount, entries)
	}
	if entries[0].Action != wantNewest {
		t.Errorf("entri terbaru harusnya %q, dapat %q", wantNewest, entries[0].Action)
	}
}
