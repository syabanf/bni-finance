package paperid

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// KREDENSIAL CALLBACK HANYA DIBACA DARI HEADER Paper-Company-Id.
//
// Dashboard Paper.id mengirim header itu bila opsi "Kirim paper company id"
// dicentang. Jalur lama (?token= dan header x-paper-callback-token) sudah
// dicabut, dan tes ini memastikan keduanya tidak diam-diam tetap diterima:
// kredensial yang masih dibaca dari tempat yang tidak lagi didokumentasikan
// adalah kredensial yang tidak ada yang menjaganya.
func TestWebhookMembacaHeaderPaperCompanyId(t *testing.T) {
	const companyID = "f91a7877-9d3b-4416-8725-ecf5b9bb6223"
	kasus := []struct {
		nama   string
		header map[string]string
		query  string
		mau    int
	}{
		{"Paper-Company-Id benar", map[string]string{"Paper-Company-Id": companyID}, "", 200},
		{"huruf kecil tetap dikenali (header tidak peka huruf)", map[string]string{"paper-company-id": companyID}, "", 200},
		{"tanpa header", nil, "", 401},
		{"Paper-Company-Id salah", map[string]string{"Paper-Company-Id": "perusahaan-lain"}, "", 401},
		{"?token= lama tidak lagi diterima", nil, "?token=" + companyID, 401},
		{"x-paper-callback-token lama tidak lagi diterima", map[string]string{"x-paper-callback-token": companyID}, "", 401},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			store := &stubStore{settleReturns: true}
			mux := http.NewServeMux()
			NewHandler(newService(store, &stubGateway{}, companyID)).RegisterPublic(mux)

			req := httptest.NewRequest(http.MethodPost,
				"/api/v1/webhooks/paperid/payment-in"+k.query, bytes.NewReader(mustJSONBytes(paidWebhook())))
			for nama, nilai := range k.header {
				req.Header.Set(nama, nilai)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != k.mau {
				t.Fatalf("status %d, harusnya %d: %s", rec.Code, k.mau, rec.Body.String())
			}
			if (k.mau == 200) != store.settleRef.called {
				t.Errorf("settle dipanggil=%v, tidak sesuai dengan status %d", store.settleRef.called, k.mau)
			}
		})
	}
}
