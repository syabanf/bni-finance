package paperid

import (
	"context"
	"testing"
)

// CALLBACK TANPA IDENTITAS DIAKUI, BUKAN DITOLAK, DAN TIDAK MELUNASI APA PUN.
//
// Setiap bentuk callback yang didokumentasikan Paper.id membawa identitas
// invoice. Payload tanpa satu pun identitas (uji dari dashboard mereka, atau
// bentuk yang belum kita kenal) bukan pembayaran yang bisa dicocokkan, dan
// mengulanginya tidak akan pernah berhasil. Menjawab 400 hanya membuat
// Paper.id mengirim ulang tanpa henti.
//
// Dua hal yang dijaga sekaligus: jawabannya 200, dan settle TIDAK dipanggil.
// Yang kedua lebih penting dari yang pertama.
func TestWebhookTanpaIdentitasDiakuiTanpaMelunasi(t *testing.T) {
	kasus := []struct {
		nama string
		body string
	}{
		{"kosong sama sekali", `{}`},
		{"status PAID tanpa identitas maupun payment_info", `{"status":"PAID","message":"test"}`},
		{"payment_info PAID tanpa identitas",
			`{"payment_info":{"method":"bank_transfer","status":"PAID","bank_transfer":{"amount":1000,"paid_amount":1000,"status":"PAID"}}}`},
		{"uji dari dashboard", `{"event":"test","message":"this is a test callback"}`},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			store := &stubStore{settleReturns: true}
			svc := newService(store, &stubGateway{}, "rahasia")
			settled, err := svc.HandleWebhook(context.Background(),
				"/api/v1/webhooks/paperid/payment-in", "rahasia", []byte(k.body))
			if err != nil {
				t.Fatalf("harus diakui tanpa galat, dapat %v", err)
			}
			if settled {
				t.Error("tidak boleh melaporkan lunas")
			}
			if store.settleRef.called {
				t.Error("settle dipanggil padahal tidak ada invoice yang bisa dicocokkan")
			}
		})
	}
}

// Yang BUKAN JSON tetap 400: itu bukan callback Paper.id dalam bentuk apa pun,
// dan menerimanya diam-diam menyembunyikan proxy atau konfigurasi yang salah.
func TestWebhookBukanJSONTetapDitolak(t *testing.T) {
	svc := newService(&stubStore{}, &stubGateway{}, "rahasia")
	_, err := svc.HandleWebhook(context.Background(),
		"/api/v1/webhooks/paperid", "rahasia", []byte("<html>502 Bad Gateway</html>"))
	if statusOf(err) != 400 {
		t.Fatalf("bukan JSON harus 400, dapat %v", err)
	}
}
