package paperid

import (
	"context"
	"testing"
)

// KREDENSIAL KOSONG TIDAK BOLEH LOLOS, TERMASUK SAAT KONFIGURASINYA JUGA KOSONG.
//
// Endpoint ini duduk di luar middleware autentikasi, karena Paper.id memanggil
// tanpa login. Header Paper-Company-Id satu-satunya yang membedakan callback
// sungguhan dari siapa pun yang tahu alamatnya.
//
// Bahayanya ada pada subtle.ConstantTimeCompare: membandingkan dua string
// kosong menghasilkan 1, yaitu COCOK. Jadi tanpa penjaga eksplisit, instalasi
// yang PAPER_ID_COMPANY_ID-nya belum diisi akan menerima callback tanpa
// header apa pun dan menandai invoice lunas. Dibuktikan dengan menghapus
// penjaganya: settle dipanggil, galat nil.
//
// Tes yang sudah ada mengirim kredensial "apa pun", yang tidak pernah menyentuh
// kombinasi itu. Menghapus penjaganya membuat seluruh paket tetap hijau.
func TestWebhookKredensialKosongDitolak(t *testing.T) {
	kasus := []struct {
		nama        string
		terkonfigur string
		dikirim     string
	}{
		{"konfigurasi kosong, header kosong", "", ""},
		{"konfigurasi kosong, header diisi", "", "apa pun"},
		{"konfigurasi diisi, header kosong", "rahasia", ""},
		{"konfigurasi diisi, header salah", "rahasia", "salah"},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			store := &stubStore{}
			svc := newService(store, &stubGateway{}, k.terkonfigur)
			_, err := svc.HandleWebhook(context.Background(),
				"/api/v1/webhooks/paperid", k.dikirim, mustJSONBytes(paidWebhook()))
			if statusOf(err) != 401 {
				t.Errorf("harus 401, dapat %v", err)
			}
			// Yang menentukan bukan kode statusnya, melainkan ini: tidak boleh
			// ada invoice yang berpindah status.
			if store.settleRef.called {
				t.Error("settle dipanggil padahal kredensialnya tidak sah")
			}
		})
	}
}

// Jalur acknowledge memeriksa kredensial dengan aturan yang sama.
//
// Ia tidak menyentuh invoice, jadi godaan untuk membiarkannya terbuka besar.
// Tapi endpoint terbuka yang menerima apa saja adalah tempat menumpuknya
// sampah, dan rekaman yang tidak bisa dipercaya asalnya justru tidak berguna
// saat dipakai menelusuri masalah. Sebelum ini ia tidak punya satu pun tes.
func TestAcknowledgeMemeriksaKredensial(t *testing.T) {
	kasus := []struct {
		nama        string
		terkonfigur string
		dikirim     string
		mau401      bool
	}{
		{"konfigurasi kosong, header kosong", "", "", true},
		{"konfigurasi kosong, header diisi", "", "apa pun", true},
		{"konfigurasi diisi, header kosong", "rahasia", "", true},
		{"konfigurasi diisi, header salah", "rahasia", "salah", true},
		{"header benar", "rahasia", "rahasia", false},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			svc := newService(&stubStore{}, &stubGateway{}, k.terkonfigur)
			err := svc.AcknowledgeWebhook(context.Background(),
				"/api/v1/webhooks/paperid/payment-out", k.dikirim, mustJSONBytes(paidWebhook()))
			if k.mau401 {
				if statusOf(err) != 401 {
					t.Errorf("harus 401, dapat %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("header benar harus diterima, dapat %v", err)
			}
		})
	}
}

// SAKELAR TERBUKA MELEWATI PEMERIKSAAN, DAN BAWAANNYA MATI.
//
// Keduanya diuji bersama karena yang berbahaya bukan modenya, melainkan
// kemungkinan ia menyala tanpa seorang pun meminta. Service yang dibangun
// tanpa menyentuh sakelar ini harus tetap menolak callback tanpa kredensial.
func TestCallbackTerbukaMelewatiKredensial(t *testing.T) {
	t.Run("bawaan: tertutup", func(t *testing.T) {
		store := &stubStore{}
		svc := newService(store, &stubGateway{}, "rahasia") // sakelar tidak disentuh
		if _, err := svc.HandleWebhook(context.Background(),
			"/api/v1/webhooks/paperid", "", mustJSONBytes(paidWebhook())); statusOf(err) != 401 {
			t.Fatalf("tanpa sakelar harus tetap 401, dapat %v", err)
		}
		if store.settleRef.called {
			t.Error("settle dipanggil padahal sakelarnya tidak dinyalakan")
		}
	})

	t.Run("terbuka: header kosong diterima", func(t *testing.T) {
		// settleReturns menentukan apakah ADA invoice yang cocok dengan
		// referensinya. Tanpa itu, settle tetap dipanggil tapi melaporkan nol
		// baris berubah, dan tesnya memerah pada hal yang bukan intinya.
		store := &stubStore{settleReturns: true}
		svc := newService(store, &stubGateway{}, "rahasia").IzinkanCallbackTanpaToken(true)
		settled, err := svc.HandleWebhook(context.Background(),
			"/api/v1/webhooks/paperid", "", mustJSONBytes(paidWebhook()))
		if err != nil {
			t.Fatalf("mode terbuka harus menerima, dapat %v", err)
		}
		if !settled || !store.settleRef.called {
			t.Errorf("invoice harus dilunasi: settled=%v settleDipanggil=%v", settled, store.settleRef.called)
		}
	})

	t.Run("terbuka: tanpa company id terkonfigurasi sama sekali", func(t *testing.T) {
		store := &stubStore{}
		svc := newService(store, &stubGateway{}, "").IzinkanCallbackTanpaToken(true)
		if _, err := svc.HandleWebhook(context.Background(),
			"/api/v1/webhooks/paperid", "", mustJSONBytes(paidWebhook())); err != nil {
			t.Fatalf("mode terbuka tidak boleh menuntut company id terkonfigurasi, dapat %v", err)
		}
	})

	t.Run("terbuka: jalur acknowledge ikut terbuka", func(t *testing.T) {
		svc := newService(&stubStore{}, &stubGateway{}, "rahasia").IzinkanCallbackTanpaToken(true)
		if err := svc.AcknowledgeWebhook(context.Background(),
			"/api/v1/webhooks/paperid/payment-out", "", mustJSONBytes(paidWebhook())); err != nil {
			t.Errorf("mode terbuka harus menerima, dapat %v", err)
		}
	})

	t.Run("terbuka lalu dimatikan lagi", func(t *testing.T) {
		store := &stubStore{}
		svc := newService(store, &stubGateway{}, "rahasia").
			IzinkanCallbackTanpaToken(true).
			IzinkanCallbackTanpaToken(false)
		if _, err := svc.HandleWebhook(context.Background(),
			"/api/v1/webhooks/paperid", "", mustJSONBytes(paidWebhook())); statusOf(err) != 401 {
			t.Fatalf("sakelar harus bisa dimatikan lagi, dapat %v", err)
		}
	})
}
