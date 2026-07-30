package metrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// --- cardinality --------------------------------------------------------------

// The single most damaging mistake a metrics endpoint can make.
//
// Prometheus keeps every series it has ever scraped in memory. A `path` label
// taken from r.URL.Path means one series per invoice id — normal traffic then
// grows the scraper's heap without bound until it dies, taking the monitoring
// down with it. The route PATTERN keeps it at one series per endpoint.
func TestPathLabelUsesRoutePatternNotURL(t *testing.T) {
	reg := NewRegistry()
	h := NewHTTP(reg)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/invoices/{id}", func(w http.ResponseWriter, r *http.Request) {})
	srv := httptest.NewServer(h.Middleware(mux)(mux))
	defer srv.Close()

	// 200 different invoice ids — the shape of ordinary traffic.
	for i := 0; i < 200; i++ {
		res, err := http.Get(fmt.Sprintf("%s/api/v1/invoices/inv-%d", srv.URL, i))
		if err != nil {
			t.Fatalf("permintaan %d: %v", i, err)
		}
		res.Body.Close()
	}

	if got := h.requests.Series(); got != 1 {
		t.Errorf("200 id berbeda harus jadi 1 series, dapat %d — label memakai URL, bukan pola route", got)
	}
	if got := h.duration.Series(); got != 1 {
		t.Errorf("histogram: harus 1 series, dapat %d", got)
	}

	body := render(reg)
	if !strings.Contains(body, `route="/api/v1/invoices/{id}"`) {
		t.Errorf("label route bukan pola:\n%s", firstLines(body, 8))
	}
	if strings.Contains(body, "inv-7") {
		t.Error("id invoice bocor ke label")
	}
}

// A 404 is attacker-controlled: any path they invent would otherwise mint a
// series. They all collapse into one bucket.
func TestUnmatchedPathsCollapseToOneSeries(t *testing.T) {
	reg := NewRegistry()
	h := NewHTTP(reg)
	empty := http.NewServeMux()
	srv := httptest.NewServer(h.Middleware(empty)(empty))
	defer srv.Close()

	for i := 0; i < 50; i++ {
		res, _ := http.Get(fmt.Sprintf("%s/tidak-ada-%d", srv.URL, i))
		res.Body.Close()
	}

	if got := h.requests.Series(); got != 1 {
		t.Errorf("50 path tak dikenal harus jadi 1 series, dapat %d", got)
	}
	if body := render(reg); !strings.Contains(body, `route="other"`) {
		t.Errorf("path tak cocok harus berlabel other:\n%s", firstLines(body, 8))
	}
}

// --- what must never appear ---------------------------------------------------

// The blackbox has the same rule for headers. Here it is labels: metrics are
// scraped by systems with far looser access control than the API, and a label
// containing an email or an amount turns the monitoring stack into a data
// export. Nothing in this package accepts business data, and this pins it.
func TestNoBusinessDataInLabels(t *testing.T) {
	reg := NewRegistry()
	h := NewHTTP(reg)
	ints := NewIntegrations(reg)
	RegisterRuntime(reg)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/members/{id}", func(w http.ResponseWriter, r *http.Request) {})
	srv := httptest.NewServer(h.Middleware(mux)(mux))
	defer srv.Close()

	res, _ := http.Get(srv.URL + "/api/v1/members/mem-003?email=fahmi@wit.id&phone=082240274833")
	res.Body.Close()
	ints.Record("paper_id", "outbound", true, 1.2)

	body := render(reg)
	for _, secret := range []string{"fahmi@wit.id", "082240274833", "mem-003", "1500000"} {
		if strings.Contains(body, secret) {
			t.Errorf("data bisnis %q muncul di metrics:\n%s", secret, body)
		}
	}
}

// --- exposition format ---------------------------------------------------------

func TestHistogramBucketsAreCumulative(t *testing.T) {
	reg := NewRegistry()
	h := reg.NewHistogram("t_seconds", "uji", []float64{1, 2, 3})

	for _, v := range []float64{0.5, 1.5, 2.5, 99} {
		h.Observe(nil, v)
	}

	body := render(reg)
	// Cumulative: le=1 → 1, le=2 → 2, le=3 → 3, +Inf → 4. A non-cumulative
	// histogram parses fine and produces silently wrong quantiles.
	for _, want := range []string{
		`t_seconds_bucket{le="1"} 1`,
		`t_seconds_bucket{le="2"} 2`,
		`t_seconds_bucket{le="3"} 3`,
		`t_seconds_bucket{le="+Inf"} 4`,
		`t_seconds_count 4`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("hilang %q dari:\n%s", want, body)
		}
	}
	if !strings.Contains(body, "t_seconds_sum 103.5") {
		t.Errorf("sum salah:\n%s", body)
	}
}

// A quote in a label value ends the string early and makes the whole payload
// unparseable — the scraper then drops EVERY metric, not just this one.
func TestLabelValuesAreEscaped(t *testing.T) {
	reg := NewRegistry()
	c := reg.NewCounter("t_total", "uji")
	c.Inc(Labels{"route": `/a"b\c`})

	body := render(reg)
	if !strings.Contains(body, `route="/a\"b\\c"`) {
		t.Errorf("label tidak di-escape: %s", body)
	}
}

func TestExpositionHasHelpAndTypeForEveryMetric(t *testing.T) {
	reg := NewRegistry()
	NewHTTP(reg)
	NewIntegrations(reg)
	RegisterRuntime(reg)

	body := render(reg)
	var names []string
	for _, line := range strings.Split(body, "\n") {
		if after, ok := strings.CutPrefix(line, "# TYPE "); ok {
			names = append(names, strings.Fields(after)[0])
		}
	}
	if len(names) < 8 {
		t.Fatalf("terlalu sedikit metrik terdaftar: %v", names)
	}
	for _, n := range names {
		if !strings.Contains(body, "# HELP "+n+" ") {
			t.Errorf("%s tidak punya HELP — scraper menampilkannya tanpa keterangan", n)
		}
	}
}

// --- akses --------------------------------------------------------------------

func TestHandlerOpenWithoutToken(t *testing.T) {
	reg := NewRegistry()
	NewHTTP(reg)

	rec := httptest.NewRecorder()
	Handler(reg, "").ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("tanpa token dikonfigurasi harus terbuka, dapat %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content-type harus text/plain untuk format eksposisi, dapat %q", ct)
	}
}

func TestHandlerRequiresTokenWhenConfigured(t *testing.T) {
	reg := NewRegistry()
	NewHTTP(reg)
	h := Handler(reg, "rahasia-scrape")

	cases := []struct {
		name, header string
		want         int
	}{
		{"tanpa header", "", http.StatusUnauthorized},
		{"token salah", "Bearer salah", http.StatusUnauthorized},
		{"skema salah", "Basic rahasia-scrape", http.StatusUnauthorized},
		{"benar", "Bearer rahasia-scrape", http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/metrics", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Errorf("status %d, diharapkan %d", rec.Code, c.want)
			}
		})
	}
}

// --- concurrency ---------------------------------------------------------------

// Handlers run concurrently, so every counter is touched from many goroutines.
// Run with -race; a lost update here would quietly under-report traffic.
func TestConcurrentWritesAreSafe(t *testing.T) {
	reg := NewRegistry()
	c := reg.NewCounter("t_total", "uji")
	h := reg.NewHistogram("t_seconds", "uji", nil)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Inc(Labels{"route": fmt.Sprintf("/r%d", i%5)})
				h.Observe(Labels{"route": fmt.Sprintf("/r%d", i%5)}, 0.01)
			}
		}(i)
	}
	wg.Wait()

	if got := c.Series(); got != 5 {
		t.Errorf("harus 5 series, dapat %d", got)
	}
	if got := c.total(); got != 5000 {
		t.Errorf("hitungan hilang: %v, diharapkan 5000", got)
	}
}

// --- helpers -------------------------------------------------------------------

func render(reg *Registry) string {
	var b strings.Builder
	reg.Write(&b)
	return b.String()
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// The API mounts a second mux at the /api/ prefix, so asking only the root mux
// answers "/api/" for every endpoint under it — safe cardinality, useless data,
// and nothing fails. Found by scraping a running server and seeing one line
// where there should have been dozens.
func TestMountedSubMuxResolvesToInnerPattern(t *testing.T) {
	reg := NewRegistry()
	h := NewHTTP(reg)

	inner := http.NewServeMux()
	inner.HandleFunc("GET /api/v1/invoices", func(w http.ResponseWriter, r *http.Request) {})
	inner.HandleFunc("GET /api/v1/members/{id}", func(w http.ResponseWriter, r *http.Request) {})

	root := http.NewServeMux()
	root.Handle("/api/", inner)

	srv := httptest.NewServer(h.Middleware(root, inner)(root))
	defer srv.Close()

	for _, p := range []string{"/api/v1/invoices", "/api/v1/members/m1", "/api/v1/members/m2"} {
		res, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		res.Body.Close()
	}

	body := render(reg)
	for _, want := range []string{
		`route="/api/v1/invoices"`,
		`route="/api/v1/members/{id}"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("hilang %s — endpoint runtuh ke prefix mount:\n%s", want, firstLines(body, 6))
		}
	}
	if strings.Contains(body, `route="/api/"`) {
		t.Error("masih memakai prefix mount sebagai label")
	}
	// Dua id member berbeda tetap satu series.
	if got := h.requests.Series(); got != 2 {
		t.Errorf("harus 2 series (invoices + members/{id}), dapat %d", got)
	}
}
