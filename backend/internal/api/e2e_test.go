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
	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/blackbox"
	"github.com/syabanf/bni-finance/backend/internal/chapter"
	"github.com/syabanf/bni-finance/backend/internal/config"
	"github.com/syabanf/bni-finance/backend/internal/dashboard"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/invoice"
	"github.com/syabanf/bni-finance/backend/internal/member"
	"github.com/syabanf/bni-finance/backend/internal/paperid"
	"github.com/syabanf/bni-finance/backend/internal/payment"
	"github.com/syabanf/bni-finance/backend/internal/publicpay"
	"github.com/syabanf/bni-finance/backend/internal/settings"
)

// One test that walks the whole business journey over HTTP: sign in, register a
// member, bill them, push the invoice to Paper.id, receive the payment
// callback, and watch it land in the dashboard.
//
// The other tests here each prove one layer. This one proves they compose — the
// failures it is meant to catch live BETWEEN layers and are invisible to any of
// them alone. Two from this project:
//
//   - PUT was missing from the CORS allow-list, so every browser call to a PUT
//     route failed while curl and every handler test passed.
//   - customer.id was derived from the member's contact details, so a member
//     whose phone changed could never be billed again — the send path was
//     correct, the client was correct, and the pair was broken.
//
// Gated the same way as the other integration tests:
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
//
// The Paper.id leg additionally needs PAPER_ID_CLIENT_ID / _SECRET. Without
// them the journey still runs; the send step is skipped rather than faked,
// because a fake there would prove nothing about the one integration that
// actually breaks.

const e2eCallbackToken = "token-callback-e2e"

type e2eStack struct {
	srv        *httptest.Server
	client     *http.Client
	pool       *pgxpool.Pool
	adminToken string
	userToken  string
	paperLive  bool

	// invoicesMade menghitung invoice yang dibuat sepanjang perjalanan.
	// Dashboard menegaskan Total.Count sama persis dengan ini — dan karena
	// retry transien Paper.id membuat invoice segar, angka tetapnya "2" akan
	// bohong tepat pada run yang paling perlu dipercaya.
	invoicesMade int
}

func newE2EStack(t *testing.T) *e2eStack {
	t.Helper()
	pool := integrationPool(t)

	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	signer := testSigner(t)
	recorder := blackbox.New(50)

	authSvc := auth.NewService(auth.NewRepository(pool), signer)

	clientID := os.Getenv("PAPER_ID_CLIENT_ID")
	clientSecret := os.Getenv("PAPER_ID_CLIENT_SECRET")
	baseURL := os.Getenv("PAPER_ID_BASE_URL")
	paperSvc := paperid.NewService(paperid.NewRepository(pool),
		baseURL, clientID, clientSecret, e2eCallbackToken, recorder)

	h := api.NewHandler(log, config.Config{AllowedOrigins: []string{"*"}}, signer, api.Services{
		Auth:      authSvc,
		Invoice:   invoice.NewService(invoice.NewRepository(pool)),
		Payment:   payment.NewService(payment.NewRepository(pool)),
		Member:    member.NewService(member.NewRepository(pool)),
		Chapter:   chapter.NewService(chapter.NewRepository(pool)),
		Settings:  settings.NewService(settings.NewRepository(pool)),
		Audit:     audit.NewService(audit.NewRepository(pool)),
		Dashboard: dashboard.NewService(dashboard.NewRepository(pool)),
		Public:    publicpay.NewService(publicpay.NewRepository(pool), "", "", recorder),
		PaperID:   paperSvc,
		Blackbox:  recorder,
	}, pool.Ping)

	st := &e2eStack{
		srv: httptest.NewServer(h),
		// 90 detik: lebih longgar daripada timeout 60 detik yang dipakai
		// backend ke Paper.id, supaya yang terlihat adalah error backend,
		// bukan timeout klien tes.
		client:    loadClient(t, 1024, 90*time.Second),
		pool:      pool,
		paperLive: clientID != "" && clientSecret != "",
	}
	t.Cleanup(st.srv.Close)

	// Dedicated accounts, removed afterwards: the shared dev database may
	// already hold real users, and this test must not disturb them.
	st.adminToken = st.makeUser(t, authSvc, "e2e-admin@contoh.local", domain.RoleAdmin)
	st.userToken = st.makeUser(t, authSvc, "e2e-user@contoh.local", domain.RoleUser)
	return st
}

func (s *e2eStack) makeUser(t *testing.T, svc *auth.Service, email string, role domain.UserRole) string {
	t.Helper()
	ctx := context.Background()
	// A leftover from a failed run would make Create fail on the unique index.
	_, _ = s.pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)

	const password = "kata-sandi-e2e-yang-panjang"
	r := role
	if _, err := svc.Create(ctx, domain.CreateUserInput{
		Email: email, Password: password, Name: "E2E " + string(role), Role: &r,
	}); err != nil {
		t.Fatalf("buat pengguna %s: %v", email, err)
	}
	t.Cleanup(func() {
		_, _ = s.pool.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	})

	var out struct {
		Token string `json:"token"`
	}
	s.do(t, "POST", "/api/v1/auth/login", "",
		fmt.Sprintf(`{"email":%q,"password":%q}`, email, password), http.StatusOK, &out)
	if out.Token == "" {
		t.Fatalf("login %s tidak mengembalikan token", email)
	}
	return out.Token
}

// do performs one request and asserts the status, decoding the body when out is
// non-nil. Every step of the journey goes through here, so a failure reports the
// endpoint and the server's own error message rather than a bare status code.
func (s *e2eStack) do(t *testing.T, method, path, token, body string, wantStatus int, out any) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, s.srv.URL+path, reader)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	if res.StatusCode != wantStatus {
		t.Fatalf("%s %s: status %d, diharapkan %d — %s", method, path, res.StatusCode, wantStatus, raw)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s %s: body bukan JSON — %s", method, path, raw)
		}
	}
}

// freshDraft membuat satu invoice draft bernomor segar dan mencatatnya.
func (s *e2eStack) freshDraft(t *testing.T) e2eInvoice {
	t.Helper()
	var inv e2eInvoice
	s.do(t, "POST", "/api/v1/invoices", s.adminToken,
		fmt.Sprintf(`{"number":"E2E-%d","memberId":"mem-e2e","chapterId":"ch-e2e","type":"renewal",`+
			`"amount":250000,"dueDate":"2026-12-31","periodStart":"2026-07-27",`+
			`"periodEnd":"2027-07-27"}`, time.Now().UnixNano()),
		http.StatusCreated, &inv)
	s.invoicesMade++
	return inv
}

// transientPaperFailure memutuskan apakah kegagalan kirim layak diulang dengan
// invoice SEGAR — bukan invoice yang sama, karena timeout klien yang sebenarnya
// berhasil di hulu membakar nomornya secara permanen.
//
// Paper.id staging bukan lawan yang stabil: pada lima run hari ini, langkah
// yang sama memakan 5 dtk empat kali lalu 52 dtk sekali; run sebelumnya pernah
// 45 dtk, dan timeout klien 60 dtk. Tanpa retry, satu ayunan menjatuhkan
// seluruh paket — kandidat terkuat untuk kegagalan tunggal yang tidak pernah
// bisa direproduksi itu.
//
// 409 hanya diulang bila pesannya menyebut nomor terpakai. 409 partner
// mismatch TIDAK diulang: itu persis bug customerRef yang dijaga langkah
// "ganti nomor", dan mengulangnya hanya mengaburkan kegagalan yang sah.
func transientPaperFailure(status int, body string) bool {
	switch status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	case http.StatusConflict:
		return strings.Contains(body, "nomor sudah dipakai")
	}
	return false
}

// sendRetrying mengirim inv ke Paper.id; kegagalan transien diulang dengan
// invoice segar, maksimal tiga percobaan. Setiap pengulangan dicatat KERAS —
// flake yang bersembunyi adalah flake yang tidak pernah diperbaiki.
func (s *e2eStack) sendRetrying(t *testing.T, inv e2eInvoice) e2eInvoice {
	t.Helper()
	const attempts = 3
	for i := 1; ; i++ {
		status, body := s.post("/api/v1/invoices/"+inv.ID+"/send", s.adminToken,
			`{"sendEmail":false,"sendWhatsApp":false}`)
		if status == http.StatusOK {
			var sent e2eInvoice
			if err := json.Unmarshal([]byte(body), &sent); err != nil {
				t.Fatalf("respons kirim bukan JSON: %s", body)
			}
			return sent
		}
		if i >= attempts || !transientPaperFailure(status, body) {
			t.Fatalf("kirim ke Paper.id gagal (HTTP %d) pada percobaan %d: %s",
				status, i, strings.TrimSpace(body))
		}
		t.Logf("PERCOBAAN %d GAGAL TRANSIEN (HTTP %d): %s — ulang dengan invoice segar",
			i, status, strings.TrimSpace(body))
		inv = s.freshDraft(t)
	}
}

type e2eInvoice struct {
	ID                string `json:"id"`
	Number            string `json:"number"`
	Status            string `json:"status"`
	Amount            int64  `json:"amount"`
	PaperIDInvoiceID  string `json:"paperIdInvoiceId"`
	PaperIDPaymentURL string `json:"paperIdPaymentUrl"`
	PaidAmount        *int64 `json:"paidAmount"`
}

func TestEndToEndInvoiceJourney(t *testing.T) {
	s := newE2EStack(t)

	t.Run("server hidup tanpa token", func(t *testing.T) {
		s.do(t, "GET", "/healthz", "", "", http.StatusOK, nil)
	})

	t.Run("tanpa token ditolak", func(t *testing.T) {
		s.do(t, "GET", "/api/v1/invoices", "", "", http.StatusUnauthorized, nil)
	})

	t.Run("profil terbaca dengan token", func(t *testing.T) {
		var me struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		s.do(t, "GET", "/api/v1/auth/me", s.adminToken, "", http.StatusOK, &me)
		if me.Email != "e2e-admin@contoh.local" || me.Role != "admin" {
			t.Fatalf("profil salah: %+v", me)
		}
	})

	t.Run("peran user tidak boleh menulis", func(t *testing.T) {
		s.do(t, "POST", "/api/v1/chapters", s.userToken,
			`{"id":"ch-e2e-x","name":"x","displayName":"X"}`, http.StatusForbidden, nil)
	})

	// --- data induk ---------------------------------------------------------

	s.do(t, "POST", "/api/v1/chapters", s.adminToken,
		`{"id":"ch-e2e","name":"e2e","displayName":"BNI E2E","areaName":"Jakarta","cityName":"Jakarta"}`,
		http.StatusCreated, nil)

	s.do(t, "POST", "/api/v1/members", s.adminToken,
		`{"id":"mem-e2e","chapterId":"ch-e2e","name":"Peserta E2E",`+
			`"email":"fahmi@wit.id","phone":"082240274833","status":"active"}`,
		http.StatusCreated, nil)

	// --- tagihan ------------------------------------------------------------

	// The number is supplied rather than auto-generated, and that is the whole
	// difference between a test that can run twice and one that cannot.
	//
	// This test TRUNCATEs invoices with RESTART IDENTITY, so the generator hands
	// out INV-<year>-001 on every run. Paper.id refuses a repeated number
	// PERMANENTLY, so the first run passed and every run after it failed 403 —
	// discovered by running it a second time.
	number := fmt.Sprintf("E2E-%d", time.Now().UnixNano())

	var inv e2eInvoice
	t.Run("invoice dibuat sebagai draft", func(t *testing.T) {
		s.do(t, "POST", "/api/v1/invoices", s.adminToken,
			fmt.Sprintf(`{"number":%q,"memberId":"mem-e2e","chapterId":"ch-e2e","type":"renewal",`+
				`"amount":250000,"dueDate":"2026-12-31","periodStart":"2026-07-27",`+
				`"periodEnd":"2027-07-27"}`, number),
			http.StatusCreated, &inv)

		if inv.Status != "draft" {
			t.Errorf("status awal harus draft, dapat %q", inv.Status)
		}
		if inv.Number != number {
			t.Errorf("nomor tidak dipakai apa adanya: %q", inv.Number)
		}
		s.invoicesMade++
	})

	t.Run("audit mencatat pembuatan", func(t *testing.T) {
		var log struct {
			Data []struct {
				Action string `json:"action"`
			} `json:"data"`
		}
		s.do(t, "GET", "/api/v1/invoices/"+inv.ID+"/audit", s.adminToken, "", http.StatusOK, &log)
		// Invoice dan baris auditnya ditulis dalam satu transaksi; tidak ada
		// invoice tanpa jejak.
		if len(log.Data) == 0 || log.Data[len(log.Data)-1].Action != "created" {
			t.Fatalf("entri audit `created` tidak ada: %+v", log.Data)
		}
	})

	// --- Paper.id -----------------------------------------------------------

	if !s.paperLive {
		t.Skip("PAPER_ID_CLIENT_ID/_SECRET tidak diset — sisa perjalanan dilewati " +
			"(memalsukannya tidak membuktikan apa pun tentang integrasi yang justru sering patah)")
	}

	t.Run("invoice terkirim ke Paper.id", func(t *testing.T) {
		// Kanal dimatikan SECARA EKSPLISIT, tidak dibiarkan mengikuti
		// app_settings — tes otomatis tidak boleh mengirim email/WhatsApp ke
		// orang sungguhan. Kegagalan transien staging diulang dengan invoice
		// segar oleh sendRetrying; lihat transientPaperFailure untuk batasnya.
		sent := s.sendRetrying(t, inv)

		if sent.Status != "sent" {
			t.Errorf("status harus sent, dapat %q", sent.Status)
		}
		if sent.PaperIDInvoiceID == "" || sent.PaperIDPaymentURL == "" {
			t.Errorf("field Paper.id tidak tersimpan: %+v", sent)
		}
		inv = sent
	})

	t.Run("member yang ganti nomor masih bisa ditagih", func(t *testing.T) {
		// Regresi yang paling mahal di proyek ini. Paper.id mengikat customer.id
		// ke nama/email/telepon saat pertama melihatnya, lalu MENOLAK id yang
		// sama dengan kontak berbeda — "Failed partner doesn't match", sebuah
		// 400 yang tidak menyebut kontak sama sekali.
		//
		// Sinkronisasi BNI VM memperbarui nomor telepon secara rutin, jadi
		// tanpa penanganan, pembaruan pertama membuat member itu tidak pernah
		// bisa ditagih lagi — selamanya, dan tanpa petunjuk.
		//
		// Langkah ini yang membuat tesnya menangkap bug tersebut: mengirim dua
		// kali dengan kontak yang sama tidak membuktikan apa pun.
		s.do(t, "PATCH", "/api/v1/members/mem-e2e", s.adminToken,
			`{"phone":"081199887766"}`, http.StatusOK, nil)

		var second e2eInvoice
		s.do(t, "POST", "/api/v1/invoices", s.adminToken,
			fmt.Sprintf(`{"number":%q,"memberId":"mem-e2e","chapterId":"ch-e2e","type":"renewal",`+
				`"amount":250000,"dueDate":"2026-12-31","periodStart":"2027-07-28",`+
				`"periodEnd":"2028-07-28"}`, number+"-B"),
			http.StatusCreated, &second)
		s.invoicesMade++

		// sendRetrying TIDAK mengulang 409 partner-mismatch — itu persis bug
		// customerRef yang dijaga langkah ini, dan harus gagal seketika.
		second = s.sendRetrying(t, second)
		if second.PaperIDInvoiceID == "" {
			t.Fatal("invoice kedua tidak tersimpan id Paper.id-nya")
		}
	})

	t.Run("blackbox merekam panggilan keluar tanpa kredensial", func(t *testing.T) {
		var bb struct {
			Data []struct {
				Integration string          `json:"integration"`
				Direction   string          `json:"direction"`
				Status      int             `json:"status"`
				Success     bool            `json:"success"`
				Request     json.RawMessage `json:"request"`
			} `json:"data"`
		}
		s.do(t, "GET", "/api/v1/blackbox?integration=paper_id", s.adminToken, "", http.StatusOK, &bb)
		if len(bb.Data) == 0 {
			t.Fatal("panggilan ke Paper.id tidak terekam")
		}
		// Rekaman terbaru dulu; ambil yang paling akhir dikirim.
		call := bb.Data[0]
		if call.Direction != "outbound" || !call.Success || call.Status != http.StatusCreated {
			t.Errorf("rekaman salah: %+v", call)
		}
		// Recorder hanya menerima body; kredensial hidup di header. Ini
		// menjaga batas itu tetap benar, bukan sekadar niat.
		for _, secret := range []string{os.Getenv("PAPER_ID_CLIENT_ID"), os.Getenv("PAPER_ID_CLIENT_SECRET")} {
			if secret != "" && strings.Contains(string(call.Request), secret) {
				t.Error("blackbox membocorkan kredensial Paper.id")
			}
		}
	})

	// --- pembayaran ---------------------------------------------------------

	callback := fmt.Sprintf(`{
		"payment_date": "2026-07-27",
		"payment_info": {"method":"bank_transfer","channel":"bni","amount":250000,
		                 "paid_amount":250000,"paid_at":"2026-07-27 10:00:00","status":"PAID"},
		"additional_info": {"invoices":[{"uuid":%q,"number":%q}]}
	}`, inv.PaperIDInvoiceID, inv.Number)

	t.Run("callback dengan token salah ditolak", func(t *testing.T) {
		s.do(t, "POST", "/api/v1/webhooks/paperid?token=salah", "", callback, http.StatusUnauthorized, nil)
	})

	t.Run("callback melunasi invoice", func(t *testing.T) {
		var res struct {
			Settled bool `json:"settled"`
		}
		s.do(t, "POST", "/api/v1/webhooks/paperid?token="+e2eCallbackToken, "",
			callback, http.StatusOK, &res)
		if !res.Settled {
			t.Fatal("callback pertama harus melunasi")
		}

		var after e2eInvoice
		s.do(t, "GET", "/api/v1/invoices/"+inv.ID, s.adminToken, "", http.StatusOK, &after)
		if after.Status != "paid" {
			t.Errorf("invoice harus lunas, dapat %q", after.Status)
		}
		if after.PaidAmount == nil || *after.PaidAmount != 250000 {
			t.Errorf("nominal terbayar salah: %v", after.PaidAmount)
		}
	})

	t.Run("callback berulang tidak menggandakan pembayaran", func(t *testing.T) {
		// Paper.id mengirim ulang; tanpa idempotensi, satu pembayaran tercatat
		// dua kali dan laporan keuangannya ikut salah.
		var res struct {
			Settled bool `json:"settled"`
		}
		s.do(t, "POST", "/api/v1/webhooks/paperid?token="+e2eCallbackToken, "",
			callback, http.StatusOK, &res)
		if res.Settled {
			t.Error("callback kedua tidak boleh melunasi ulang")
		}

		var pays struct {
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
		}
		s.do(t, "GET", "/api/v1/payments?invoiceId="+inv.ID, s.adminToken, "", http.StatusOK, &pays)
		if pays.Meta.Total != 1 {
			t.Errorf("harus tepat 1 baris pembayaran, dapat %d", pays.Meta.Total)
		}
	})

	// --- hilir --------------------------------------------------------------

	t.Run("dashboard mencerminkan pelunasan", func(t *testing.T) {
		var sum struct {
			Total struct{ Count int } `json:"total"`
			Paid  struct{ Count int } `json:"paid"`
		}
		s.do(t, "GET", "/api/v1/dashboard/summary", s.adminToken, "", http.StatusOK, &sum)
		// Jumlah totalnya mengikuti hitungan nyata, bukan angka mati: retry
		// transien Paper.id membuat invoice segar, dan "harus 2" akan bohong
		// tepat pada run yang paling perlu dipercaya.
		if sum.Total.Count != s.invoicesMade || sum.Paid.Count != 1 {
			t.Errorf("ringkasan salah: total=%d lunas=%d, diharapkan %d/1",
				sum.Total.Count, sum.Paid.Count, s.invoicesMade)
		}
	})

	t.Run("halaman publik tidak membocorkan kontak member", func(t *testing.T) {
		var raw json.RawMessage
		s.do(t, "GET", "/api/v1/public/invoices/"+inv.ID, "", "", http.StatusOK, &raw)
		// Halaman ini terbuka tanpa login: siapa pun yang punya id invoice bisa
		// membacanya, jadi tidak boleh memuat email atau telepon.
		for _, leak := range []string{"fahmi@wit.id", "082240274833"} {
			if strings.Contains(string(raw), leak) {
				t.Errorf("halaman publik membocorkan %q", leak)
			}
		}
	})
}

// Klasifikasi transien-vs-nyata adalah keputusan paling berbahaya di retry ini:
// salah ke satu arah, flake staging menjatuhkan suite; salah ke arah lain,
// regresi customerRef (partner mismatch, juga 409) ikut diulang-ulang sampai
// kegagalannya kabur. Tabel ini mengunci batasnya.
func TestTransientPaperFailureClassification(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"502 gateway", 502, `{"error":"Paper.id menolak: internal"}`, true},
		{"503 unavailable", 503, `{"error":"Paper.id belum dikonfigurasi"}`, true},
		{"504 timeout", 504, `{"error":"timeout"}`, true},
		{"409 nomor terbakar", 409, `{"error":"invoice ini sudah pernah dibuat di Paper.id (nomor sudah dipakai) — periksa dashboard"}`, true},
		{"409 partner mismatch — bug customerRef, JANGAN diulang", 409,
			`{"error":"Paper.id sudah menyimpan member ini dengan nama/email/telepon yang berbeda"}`, false},
		{"400 telepon kosong — bug nyata", 400, `{"error":"member belum punya nomor telepon"}`, false},
		{"500 kita sendiri", 500, `{"error":"terjadi kesalahan pada server"}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := transientPaperFailure(c.status, c.body); got != c.want {
				t.Errorf("status=%d body=%q → %v, harusnya %v", c.status, c.body, got, c.want)
			}
		})
	}
}
