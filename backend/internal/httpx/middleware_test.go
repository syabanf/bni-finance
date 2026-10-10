package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAPIKeyGuard(t *testing.T) {
	h := APIKey("rahasia")(okHandler())

	t.Run("tanpa header ditolak", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("dapat %d, harusnya 401", rec.Code)
		}
	})

	t.Run("key salah ditolak", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
		req.Header.Set("Authorization", "Bearer salah")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("dapat %d, harusnya 401", rec.Code)
		}
	})

	t.Run("key benar diteruskan", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
		req.Header.Set("Authorization", "Bearer rahasia")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("dapat %d, harusnya 200", rec.Code)
		}
	})
}

func TestAPIKeyDisabledWhenUnset(t *testing.T) {
	h := APIKey("")(okHandler())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("tanpa API_KEY harus terbuka, dapat %d", rec.Code)
	}
}

func TestCORSPreflightAndAllowlist(t *testing.T) {
	h := CORS([]string{"https://bni-finance.vercel.app"})(okHandler())

	t.Run("preflight dijawab 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/invoices", nil)
		req.Header.Set("Origin", "https://bni-finance.vercel.app")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("dapat %d, harusnya 204", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://bni-finance.vercel.app" {
			t.Errorf("allow-origin: dapat %q", got)
		}
	})

	t.Run("origin asing tidak diberi header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
		req.Header.Set("Origin", "https://jahat.example")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("origin asing tidak boleh diizinkan, dapat %q", got)
		}
	})
}

func TestFailMapsNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	Fail(rec, ErrNotFound)
	if rec.Code != http.StatusNotFound {
		t.Errorf("dapat %d, harusnya 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	Fail(rec, Conflict("bentrok"))
	if rec.Code != http.StatusConflict {
		t.Errorf("dapat %d, harusnya 409", rec.Code)
	}
}

// "//" DI PATH TIDAK BOLEH BERUJUNG REDIRECT UNTUK POST.
//
// http.ServeMux menjawab 307 untuk POST ke path yang tidak rapi (301 untuk
// GET). Klien webhook yang tidak mengikuti redirect untuk POST menganggap
// callback-nya gagal. Diuji dua arah: tanpa RapikanPath mux memang redirect,
// dengan RapikanPath permintaannya sampai utuh berikut method dan body-nya.
func TestRapikanPathMencegah301UntukWebhook(t *testing.T) {
	var diterima struct {
		method, path, body string
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/webhooks/x", func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 64)
		n, _ := r.Body.Read(b)
		diterima.method, diterima.path, diterima.body = r.Method, r.URL.Path, string(b[:n])
		w.WriteHeader(http.StatusOK)
	})

	kirim := func(h http.Handler) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		// Path diisi langsung: target "//api/..." pada NewRequest dibaca
		// sebagai URL relatif-skema ("//host/path"), bukan sebagai path
		// bergaris miring ganda seperti yang dikirim klien sungguhan.
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ref_id":"r1"}`))
		req.URL.Path = "//api/v1/webhooks/x"
		req.URL.RawQuery = "token=t"
		req.RequestURI = "//api/v1/webhooks/x?token=t"
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := kirim(mux); rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("tanpa RapikanPath mux seharusnya redirect 307 untuk POST (itulah masalahnya), dapat %d", rec.Code)
	}

	diterima = struct{ method, path, body string }{}
	rec := kirim(RapikanPath(mux))
	if rec.Code != http.StatusOK {
		t.Fatalf("dengan RapikanPath harus 200, dapat %d", rec.Code)
	}
	if diterima.method != http.MethodPost || diterima.path != "/api/v1/webhooks/x" || diterima.body != `{"ref_id":"r1"}` {
		t.Errorf("permintaan tidak sampai utuh: %+v", diterima)
	}
}

// Path yang sudah rapi tidak disentuh sama sekali.
func TestRapikanPathMembiarkanPathRapi(t *testing.T) {
	var dapat string
	h := RapikanPath(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { dapat = r.URL.Path }))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/invoices/", nil))
	if dapat != "/api/v1/invoices/" {
		t.Errorf("path berubah jadi %q", dapat)
	}
}
