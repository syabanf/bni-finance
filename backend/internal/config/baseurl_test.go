package config

import "testing"

// APP_BASE_URL yang menunjuk localhost sementara SMTP menyala adalah kombinasi
// yang gagal di tempat yang tidak terlihat dari server: emailnya TERKIRIM,
// lognya bersih, dan yang menerima menekan tautan mati di perangkatnya.
//
// Fungsi ini yang memutuskan kapan peringatan itu muncul, jadi ia perlu benar
// untuk bentuk-bentuk yang sungguhan ditulis orang di .env.
func TestDeteksiURLLokal(t *testing.T) {
	lokal := []string{
		"http://localhost:5173",
		"http://localhost",
		"https://localhost:8443",
		"http://127.0.0.1:5173",
		"http://[::1]:5173",
		"", // belum diisi sama sekali
		"   ",
		"localhost:5173", // tanpa skema — Parse menaruhnya sebagai path, host kosong
	}
	for _, u := range lokal {
		if !Lokal(u) {
			t.Errorf("Lokal(%q) = false, seharusnya true", u)
		}
	}

	jauh := []string{
		"https://bni-finance.reddie.id",
		"https://bni-finance.reddie.id/",
		"http://192.168.1.10:5173", // LAN: bukan localhost, dan memang bisa dibuka perangkat lain
		"https://staging.contoh.id",
	}
	for _, u := range jauh {
		if Lokal(u) {
			t.Errorf("Lokal(%q) = true, seharusnya false", u)
		}
	}
}
