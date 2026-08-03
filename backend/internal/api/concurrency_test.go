package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// scale menaikkan setiap tes concurrency serentak.
//
//	CONCURRENCY=1000 make test-integration TEST_DATABASE_URL=…
//
// Bawaannya kecil supaya CI cepat; angka besar dipakai saat sengaja mencari
// batas. Yang diuji tetap invarian yang sama — hanya jumlah penekannya berubah.
func scale(t *testing.T, base int) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("CONCURRENCY"))
	if raw == "" {
		return base
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		t.Fatalf("CONCURRENCY=%q bukan angka positif", raw)
	}
	return n
}

// Concurrency against a REAL Postgres.
//
// stress_test.go already hammers the API, but through in-memory fakes — and a
// fake cannot race the way SQL does. Read-then-write against a database is
// exactly where correctness quietly breaks under load, and a Go map guarded by
// a mutex will never reproduce it.
//
// Gated like the other integration tests:
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
//
// These assert INVARIANTS, not throughput. "Survived 10k requests" says little;
// "never issued the same invoice number twice" is the property that matters,
// because violating it corrupts data rather than merely slowing down.

// concurrentStack brings up the real HTTP stack plus one chapter/member to
// bill. Paper.id is left unconfigured on purpose: these tests must not depend
// on a third party being reachable, and none of them exercise the send path.
func concurrentStack(t *testing.T) *e2eStack {
	t.Helper()
	s := newE2EStack(t)

	s.do(t, "POST", "/api/v1/chapters", s.adminToken,
		`{"id":"ch-conc","name":"conc","displayName":"BNI Concurrency"}`, http.StatusCreated, nil)
	s.do(t, "POST", "/api/v1/members", s.adminToken,
		`{"id":"mem-conc","chapterId":"ch-conc","name":"Peserta Concurrency",`+
			`"email":"fahmi@wit.id","phone":"082240274833","status":"active"}`,
		http.StatusCreated, nil)
	return s
}

// invoiceBody builds a create payload. Passing number="" lets the server
// generate one, which is the case under test.
func invoiceBody(number string, amount int) string {
	num := ""
	if number != "" {
		num = fmt.Sprintf(`"number":%q,`, number)
	}
	return fmt.Sprintf(`{%s"memberId":"mem-conc","chapterId":"ch-conc","type":"renewal",`+
		`"amount":%d,"dueDate":"2026-12-31","periodStart":"2026-07-28","periodEnd":"2027-07-28"}`,
		num, amount)
}

// post fires one request and returns status + body, without the assertion that
// s.do makes — concurrency tests need to COUNT failures, not stop at the first.
func (s *e2eStack) post(path, token, body string) (int, string) {
	req, err := http.NewRequest("POST", s.srv.URL+path, strings.NewReader(body))
	if err != nil {
		return 0, err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := s.client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(raw)
}

// --- invoice numbering ---------------------------------------------------------

// Auto-generated invoice numbers must stay unique when several admins bill at
// once — and the app does exactly that: "bulk-generate invoice renewal" fires a
// batch from one click.
//
// NextNumber does `SELECT COUNT(*)` and then INSERTs, with no transaction and
// no lock between the two. Two callers that read the same count both compute
// the same number; the unique index then rejects one of them. Nothing is
// corrupted — the constraint holds — but the request fails, and a bulk run
// loses invoices for reasons the operator cannot see.
func TestConcurrentInvoiceCreationProducesUniqueNumbers(t *testing.T) {
	s := concurrentStack(t)

	n := scale(t, 24)
	var (
		mu      sync.Mutex
		numbers = map[string]int{}
		fails   []string
		wg      sync.WaitGroup
	)

	// A barrier so every goroutine reaches the request at the same moment;
	// staggered starts would hide the race behind scheduling luck.
	gate := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-gate
			status, body := s.post("/api/v1/invoices", s.adminToken, invoiceBody("", 100_000+i))
			mu.Lock()
			defer mu.Unlock()
			if status != http.StatusCreated {
				fails = append(fails, fmt.Sprintf("HTTP %d: %s", status, strings.TrimSpace(body)))
				return
			}
			var inv struct {
				Number string `json:"number"`
			}
			json.Unmarshal([]byte(body), &inv)
			numbers[inv.Number]++
		}(i)
	}
	close(gate)
	wg.Wait()

	if len(fails) > 0 {
		t.Errorf("%d dari %d pembuatan gagal — penomoran balapan:\n  %s",
			len(fails), n, strings.Join(uniqueStrings(fails, 3), "\n  "))
	}
	for number, count := range numbers {
		if count > 1 {
			t.Errorf("nomor %s dipakai %d kali — indeks unik seharusnya mustahil ditembus", number, count)
		}
	}
	if len(numbers) != n && len(fails) == 0 {
		t.Errorf("harus %d nomor berbeda, dapat %d", n, len(numbers))
	}
}

// --- pembayaran ------------------------------------------------------------------

// Paper.id retries a callback it thinks failed, and retries can overlap. Two
// callbacks landing together must still settle the invoice once: a second
// payment row would overstate cash received in every report that sums them.
func TestConcurrentWebhooksSettleExactlyOnce(t *testing.T) {
	s := concurrentStack(t)

	number := fmt.Sprintf("CONC-%d", time.Now().UnixNano())
	var inv e2eInvoice
	s.do(t, "POST", "/api/v1/invoices", s.adminToken, invoiceBody(number, 500_000),
		http.StatusCreated, &inv)
	// Mark it sent so the settle path accepts it, without needing Paper.id.
	s.do(t, "PATCH", "/api/v1/invoices/"+inv.ID, s.adminToken,
		fmt.Sprintf(`{"status":"sent","paperIdInvoiceId":%q}`, "pp-"+inv.ID), http.StatusOK, nil)

	callback := fmt.Sprintf(`{
		"payment_date": "2026-07-28",
		"payment_info": {"method":"bank_transfer","channel":"bni","amount":500000,
		                 "paid_amount":500000,"paid_at":"2026-07-28 10:00:00","status":"PAID"},
		"additional_info": {"invoices":[{"uuid":%q,"number":%q}]}
	}`, "pp-"+inv.ID, number)

	n := scale(t, 16)
	var settled atomic.Int64
	var wg sync.WaitGroup
	gate := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			status, body := s.post("/api/v1/webhooks/paperid?token="+e2eCallbackToken, "", callback)
			if status != http.StatusOK {
				return
			}
			var res struct {
				Settled bool `json:"settled"`
			}
			json.Unmarshal([]byte(body), &res)
			if res.Settled {
				settled.Add(1)
			}
		}()
	}
	close(gate)
	wg.Wait()

	if got := settled.Load(); got != 1 {
		t.Errorf("%d callback melaporkan melunasi; harus tepat 1", got)
	}

	var pays struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	s.do(t, "GET", "/api/v1/payments?invoiceId="+inv.ID, s.adminToken, "", http.StatusOK, &pays)
	if pays.Meta.Total != 1 {
		t.Errorf("harus tepat 1 baris pembayaran, dapat %d — pelunasan ganda", pays.Meta.Total)
	}
}

// Recording a payment settles the invoice in the same transaction. Firing
// several at once must not produce several settlements of the same invoice.
func TestConcurrentManualPaymentsOnSameInvoice(t *testing.T) {
	s := concurrentStack(t)

	number := fmt.Sprintf("CONC-%d-M", time.Now().UnixNano())
	var inv e2eInvoice
	s.do(t, "POST", "/api/v1/invoices", s.adminToken, invoiceBody(number, 750_000),
		http.StatusCreated, &inv)
	s.do(t, "PATCH", "/api/v1/invoices/"+inv.ID, s.adminToken, `{"status":"sent"}`, http.StatusOK, nil)

	n := scale(t, 16)
	var created atomic.Int64
	var serverErr atomic.Int64
	var wg sync.WaitGroup
	gate := make(chan struct{})
	body := fmt.Sprintf(`{"invoiceId":%q,"amount":750000,"paymentMethod":"bank_transfer"}`, inv.ID)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			status, _ := s.post("/api/v1/payments", s.adminToken, body)
			switch {
			case status == http.StatusCreated:
				created.Add(1)
			case status >= 500:
				serverErr.Add(1)
			}
		}()
	}
	close(gate)
	wg.Wait()

	// A 500 means the race escaped as an unhandled error rather than a decision.
	if got := serverErr.Load(); got != 0 {
		t.Errorf("%d permintaan dijawab 5xx — balapan bocor sebagai error, bukan keputusan", got)
	}
	// Several payments on one invoice are allowed by design (cicilan), but the
	// invoice must not end up settled more times than it was paid.
	var after e2eInvoice
	s.do(t, "GET", "/api/v1/invoices/"+inv.ID, s.adminToken, "", http.StatusOK, &after)
	if after.Status != "paid" {
		t.Errorf("invoice harus lunas setelah %d pembayaran, dapat %q", created.Load(), after.Status)
	}
}

// --- beban campuran --------------------------------------------------------------

// Sustained mixed load against Postgres. The property under test is not speed
// but that nothing returns 5xx and the connection pool never deadlocks —
// pool exhaustion shows up here long before it shows up in production.
func TestMixedLoadAgainstPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("dilewati pada -short")
	}
	s := concurrentStack(t)

	// Seed a little data so reads have something to page through.
	for i := 0; i < 10; i++ {
		s.do(t, "POST", "/api/v1/invoices", s.adminToken,
			invoiceBody(fmt.Sprintf("SEED-%d-%d", time.Now().UnixNano(), i), 250_000),
			http.StatusCreated, nil)
	}

	workers, perWorker := scale(t, 24), 40
	var ok, clientErr, serverErr atomic.Int64
	var wg sync.WaitGroup

	reads := []string{
		"/api/v1/invoices?limit=20",
		"/api/v1/invoices?status=outstanding",
		"/api/v1/members",
		"/api/v1/chapters",
		"/api/v1/payments",
		"/api/v1/dashboard/summary",
	}

	started := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				var status int
				if i%5 == 0 {
					// One write in five, each with its own number so the
					// numbering race is not what this test measures.
					status, _ = s.post("/api/v1/invoices", s.adminToken,
						invoiceBody(fmt.Sprintf("LOAD-%d-%d-%d", time.Now().UnixNano(), w, i), 300_000))
				} else {
					req, _ := http.NewRequest("GET", s.srv.URL+reads[(w+i)%len(reads)], nil)
					req.Header.Set("Authorization", "Bearer "+s.adminToken)
					res, err := s.client.Do(req)
					if err != nil {
						serverErr.Add(1)
						continue
					}
					io.Copy(io.Discard, res.Body)
					res.Body.Close()
					status = res.StatusCode
				}
				switch {
				case status >= 500:
					serverErr.Add(1)
				case status >= 400:
					clientErr.Add(1)
				default:
					ok.Add(1)
				}
			}
		}(w)
	}
	wg.Wait()

	total := workers * perWorker
	t.Logf("%d permintaan / %v — ok=%d 4xx=%d 5xx=%d",
		total, time.Since(started).Round(time.Millisecond), ok.Load(), clientErr.Load(), serverErr.Load())

	if serverErr.Load() > 0 {
		t.Errorf("%d permintaan dijawab 5xx di bawah beban", serverErr.Load())
	}
	if clientErr.Load() > 0 {
		t.Errorf("%d permintaan dijawab 4xx — beban tidak boleh mengubah keabsahan permintaan", clientErr.Load())
	}
}

func uniqueStrings(in []string, max int) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
		if len(out) == max {
			break
		}
	}
	return out
}

// --- penjaga admin terakhir --------------------------------------------------

// Sistem harus selalu menyisakan minimal satu admin. guardLastAdmin menghitung
// admin lalu menolak bila tinggal satu — tetapi menghitung dan menurunkan
// peran adalah dua pernyataan terpisah tanpa kunci di antaranya.
//
// Dua penurunan yang berjalan bersamaan sama-sama membaca "masih ada 2", dua-
// duanya lolos, dan keduanya menulis. Hasilnya nol admin. Kode itu sendiri
// menyebutnya "keadaan yang tak bisa dipulihkan lewat API" — dan memang: tanpa
// admin, tidak ada yang bisa mengangkat admin baru.
func TestLastAdminGuardHoldsUnderConcurrency(t *testing.T) {
	s := newE2EStack(t)

	// Dua admin tambahan, sehingga totalnya persis di ambang penjaga: keduanya
	// boleh turun satu per satu, tetapi tidak boleh turun bersamaan.
	ids := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		email := fmt.Sprintf("adm-race-%d@contoh.local", i)
		s.pool.Exec(t.Context(), "DELETE FROM users WHERE email = $1", email)
		var created struct {
			ID string `json:"id"`
		}
		s.do(t, "POST", "/api/v1/users", s.adminToken,
			fmt.Sprintf(`{"email":%q,"password":"kata-sandi-uji-panjang","name":"Admin Race","role":"admin"}`, email),
			http.StatusCreated, &created)
		ids = append(ids, created.ID)
		// context.Background(), BUKAN t.Context(): sejak Go 1.24 context milik
		// tes dibatalkan tepat SEBELUM fungsi Cleanup dijalankan, jadi DELETE
		// dengan t.Context() tidak pernah sampai ke database. Errornya pun
		// dibuang karena nilai baliknya tidak diperiksa — dan akibatnya tiap
		// run meninggalkan dua akun ADMIN berkata sandi tetap di database
		// bersama. Ketahuan saat mendaftar isi tabel users: adm-race-0 dan
		// adm-race-1 masih di sana, padahal tesnya lulus.
		t.Cleanup(func() {
			if _, err := s.pool.Exec(context.Background(),
				"DELETE FROM users WHERE email = $1", email); err != nil {
				t.Errorf("bersihkan %s: %v", email, err)
			}
		})
	}

	// Kembalikan peran semula setelah tes. Tanpa ini, tes ini meninggalkan
	// database bersama TANPA ADMIN — persis keadaan yang ia jaga agar tidak
	// terjadi — dan menjalankannya dua kali membuat preconditionnya sendiri
	// tidak terpenuhi. Ketahuan setelah memeriksa tabel users usai tes lulus.
	before := map[string]string{}
	if rows, err := s.pool.Query(t.Context(), "SELECT id, role FROM users"); err == nil {
		for rows.Next() {
			var id, role string
			rows.Scan(&id, &role)
			before[id] = role
		}
		rows.Close()
	}
	t.Cleanup(func() {
		for id, role := range before {
			s.pool.Exec(context.Background(), "UPDATE users SET role = $2 WHERE id = $1", id, role)
		}
	})

	// Turunkan SEMUA admin serentak, termasuk yang dibuat newE2EStack.
	var all []string
	rows, err := s.pool.Query(t.Context(), "SELECT id FROM users WHERE role = 'admin'")
	if err != nil {
		t.Fatalf("baca admin: %v", err)
	}
	for rows.Next() {
		var id string
		rows.Scan(&id)
		all = append(all, id)
	}
	rows.Close()
	if len(all) < 3 {
		t.Fatalf("butuh minimal 3 admin untuk menguji, ada %d", len(all))
	}

	var wg sync.WaitGroup
	gate := make(chan struct{})
	for _, id := range all {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			<-gate
			req, _ := http.NewRequest("PATCH", s.srv.URL+"/api/v1/users/"+id+"/role",
				strings.NewReader(`{"role":"user"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+s.adminToken)
			res, err := s.client.Do(req)
			if err == nil {
				io.Copy(io.Discard, res.Body)
				res.Body.Close()
			}
		}(id)
	}
	close(gate)
	wg.Wait()

	var remaining int
	if err := s.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&remaining); err != nil {
		t.Fatalf("hitung admin: %v", err)
	}
	if remaining < 1 {
		t.Errorf("tersisa %d admin — sistem tidak bisa dipulihkan lewat API", remaining)
	}
	_ = ids
}

// --- transisi status invoice -------------------------------------------------

// Transisi status divalidasi terhadap status yang BARU SAJA dibaca, lalu ditulis
// lewat pernyataan terpisah. Dua permintaan bersamaan pada invoice `sent` — satu
// melunasi, satu membatalkan — sama-sama membaca `sent`, sama-sama lolos
// validasi, dan sama-sama menulis.
//
// Hasilnya invoice lunas yang berstatus dibatalkan, atau sebaliknya: uang
// diterima atas invoice yang tercatat batal. Jalur webhook aman karena
// SettleByRef mengunci barisnya FOR UPDATE dan memeriksa ulang; jalur PATCH
// manual inilah yang tidak.
//
// Tepat satu dari keduanya harus berhasil; yang kalah harus menerima 409.
func TestConflictingStatusTransitionsResolveToOne(t *testing.T) {
	s := concurrentStack(t)

	rounds := scale(t, 12)
	var bothWon, neitherWon int

	for i := 0; i < rounds; i++ {
		var inv e2eInvoice
		s.do(t, "POST", "/api/v1/invoices", s.adminToken,
			invoiceBody(fmt.Sprintf("TRANS-%d-%d", time.Now().UnixNano(), i), 400_000),
			http.StatusCreated, &inv)
		s.do(t, "PATCH", "/api/v1/invoices/"+inv.ID, s.adminToken,
			`{"status":"sent"}`, http.StatusOK, nil)

		var okCount atomic.Int64
		var wg sync.WaitGroup
		gate := make(chan struct{})
		for _, target := range []string{`{"status":"paid"}`, `{"status":"cancelled","cancelReason":"uji"}`} {
			wg.Add(1)
			go func(body string) {
				defer wg.Done()
				<-gate
				req, _ := http.NewRequest("PATCH", s.srv.URL+"/api/v1/invoices/"+inv.ID,
					strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+s.adminToken)
				res, err := s.client.Do(req)
				if err != nil {
					return
				}
				io.Copy(io.Discard, res.Body)
				res.Body.Close()
				if res.StatusCode == http.StatusOK {
					okCount.Add(1)
				}
			}(target)
		}
		close(gate)
		wg.Wait()

		switch okCount.Load() {
		case 2:
			bothWon++
		case 0:
			neitherWon++
		}
	}

	if bothWon > 0 {
		t.Errorf("%d dari %d ronde: KEDUA transisi berhasil — status akhirnya sekadar siapa menulis terakhir",
			bothWon, rounds)
	}
	if neitherWon > 0 {
		t.Errorf("%d dari %d ronde: tidak ada yang berhasil", neitherWon, rounds)
	}
}
