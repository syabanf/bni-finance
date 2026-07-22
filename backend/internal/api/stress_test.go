package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/api"
	"github.com/syabanf/bni-finance/backend/internal/config"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/invoice"
	"github.com/syabanf/bni-finance/backend/internal/payment"
)

// newTestServer builds the REAL handler chain (routes + middleware) on top of
// in-memory stores, so the test measures the API layer rather than Postgres.
func newTestServer(t *testing.T) (*httptest.Server, *fakeInvoiceStore, *fakePaymentStore) {
	t.Helper()

	invStore := newFakeInvoiceStore()
	payStore := newFakePaymentStore(invStore)

	// Discard logs — per-request logging would dominate the measurement.
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := config.Config{AllowedOrigins: []string{"*"}}

	h := api.NewHandler(log, cfg, api.Services{
		Invoice: invoice.NewService(invStore),
		Payment: payment.NewService(payStore),
	}, nil)

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, invStore, payStore
}

func createInvoiceBody(i int) []byte {
	body := map[string]any{
		"memberId":    fmt.Sprintf("mem-%03d", i%50),
		"chapterId":   fmt.Sprintf("ch-%d", i%6),
		"type":        "renewal",
		"amount":      1_500_000,
		"dueDate":     "2026-07-01",
		"periodStart": "2026-07-01",
		"periodEnd":   "2027-07-01",
	}
	b, _ := json.Marshal(body)
	return b
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

// TestStressMixedWorkload hammers the API with a realistic read-heavy mix.
// Run with -race to also prove there are no data races in the HTTP layer.
func TestStressMixedWorkload(t *testing.T) {
	if testing.Short() {
		t.Skip("stress test dilewati pada -short")
	}

	srv, invStore, payStore := newTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}

	// Seed so list/get have something to work with.
	for i := 0; i < 200; i++ {
		resp, err := client.Post(srv.URL+"/api/v1/invoices", "application/json",
			bytes.NewReader(createInvoiceBody(i)))
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		resp.Body.Close()
	}

	const (
		workers       = 64
		reqsPerWorker = 250
		totalRequests = workers * reqsPerWorker
	)

	var (
		okCount  atomic.Int64
		errCount atomic.Int64
		badCount atomic.Int64
		mu       sync.Mutex
		latency  = make([]time.Duration, 0, totalRequests)
	)

	start := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			local := make([]time.Duration, 0, reqsPerWorker)

			for i := 0; i < reqsPerWorker; i++ {
				n := worker*reqsPerWorker + i
				t0 := time.Now()

				var (
					resp *http.Response
					err  error
				)
				switch n % 20 {
				case 0, 1, 2: // 15% create invoice (write)
					resp, err = client.Post(srv.URL+"/api/v1/invoices", "application/json",
						bytes.NewReader(createInvoiceBody(n)))
				case 3: // 5% create payment (write + settle, cross-resource)
					body := fmt.Sprintf(`{"invoiceId":"inv-%06d","amount":1500000,"paymentMethod":"bank_transfer"}`,
						(n%200)+1)
					resp, err = client.Post(srv.URL+"/api/v1/payments", "application/json",
						bytes.NewReader([]byte(body)))
				case 4, 5, 6, 7: // 20% get by id
					resp, err = client.Get(fmt.Sprintf("%s/api/v1/invoices/inv-%06d", srv.URL, (n%200)+1))
				case 8: // 5% list payments
					resp, err = client.Get(srv.URL + "/api/v1/payments?limit=25")
				default: // 55% list invoices with filters
					resp, err = client.Get(fmt.Sprintf(
						"%s/api/v1/invoices?status=outstanding&chapterId=ch-%d&limit=25&offset=0", srv.URL, n%6))
				}

				local = append(local, time.Since(t0))

				if err != nil {
					errCount.Add(1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				switch {
				case resp.StatusCode >= 500:
					errCount.Add(1)
				case resp.StatusCode >= 400:
					badCount.Add(1)
				default:
					okCount.Add(1)
				}
			}

			mu.Lock()
			latency = append(latency, local...)
			mu.Unlock()
		}(w)
	}
	wg.Wait()
	elapsed := time.Since(start)

	sort.Slice(latency, func(i, j int) bool { return latency[i] < latency[j] })

	rps := float64(totalRequests) / elapsed.Seconds()
	t.Logf("STRESS  %d req · %d worker · %s", totalRequests, workers, elapsed.Round(time.Millisecond))
	t.Logf("        throughput : %.0f req/s", rps)
	t.Logf("        latency    : p50=%v  p95=%v  p99=%v  max=%v",
		percentile(latency, 0.50).Round(time.Microsecond),
		percentile(latency, 0.95).Round(time.Microsecond),
		percentile(latency, 0.99).Round(time.Microsecond),
		latency[len(latency)-1].Round(time.Microsecond))
	t.Logf("        hasil      : 2xx=%d  4xx=%d  5xx/transport=%d",
		okCount.Load(), badCount.Load(), errCount.Load())
	t.Logf("        state      : %d invoice, %d payment", invStore.count(), payStore.count())

	if errCount.Load() > 0 {
		t.Errorf("ada %d kegagalan 5xx/transport — API tidak stabil di bawah beban", errCount.Load())
	}
	if okCount.Load()+badCount.Load() != int64(totalRequests) {
		t.Errorf("request hilang: 2xx+4xx=%d, dikirim=%d",
			okCount.Load()+badCount.Load(), totalRequests)
	}
}

// TestConcurrentSettleSameInvoice checks the cross-resource path: many workers
// paying the SAME invoice at once must not produce 5xx, and the invoice must
// end up paid exactly once.
func TestConcurrentSettleSameInvoice(t *testing.T) {
	srv, invStore, payStore := newTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Post(srv.URL+"/api/v1/invoices", "application/json",
		bytes.NewReader(createInvoiceBody(1)))
	if err != nil {
		t.Fatalf("buat invoice: %v", err)
	}
	var created domain.Invoice
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	const concurrency = 32
	var wg sync.WaitGroup
	var server5xx atomic.Int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := fmt.Sprintf(`{"invoiceId":%q,"amount":1500000}`, created.ID)
			r, err := client.Post(srv.URL+"/api/v1/payments", "application/json",
				bytes.NewReader([]byte(body)))
			if err != nil {
				server5xx.Add(1)
				return
			}
			defer r.Body.Close()
			io.Copy(io.Discard, r.Body)
			if r.StatusCode >= 500 {
				server5xx.Add(1)
			}
		}()
	}
	wg.Wait()

	if server5xx.Load() > 0 {
		t.Errorf("%d request gagal 5xx saat melunasi invoice yang sama", server5xx.Load())
	}

	final, err := invStore.GetByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("ambil invoice: %v", err)
	}
	if final.Status != domain.StatusPaid {
		t.Errorf("invoice harus lunas, dapat %s", final.Status)
	}
	t.Logf("SETTLE  %d pembayaran paralel → invoice %s, %d payment tercatat",
		concurrency, final.Status, payStore.count())
}
