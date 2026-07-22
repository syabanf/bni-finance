package httpx

import (
	"net/http"
	"net/http/httptest"
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
