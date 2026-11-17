package config

import "testing"

// APP_BASE_URL yang menunjuk localhost sementara pengiriman email menyala
// adalah kombinasi
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

// Bawaan DOCS_PUBLIC mengikuti APP_BASE_URL, supaya produksi tertutup tanpa
// perlu mengingat satu variabel lagi di .env.production.
func TestDocsPublicMengikutiBaseURL(t *testing.T) {
	kasus := []struct {
		baseURL, docsPublic string
		ingin               bool
	}{
		{"http://localhost:5173", "", true},
		{"https://bnisystem.id", "", false},
		{"https://bnisystem.id", "true", true},
		{"http://localhost:5173", "false", false},
	}
	for _, k := range kasus {
		t.Setenv("DATABASE_URL", "postgres://x/y")
		t.Setenv("JWT_SECRET", "rahasia-uji")
		t.Setenv("APP_BASE_URL", k.baseURL)
		t.Setenv("DOCS_PUBLIC", k.docsPublic)
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DocsPublic != k.ingin {
			t.Errorf("APP_BASE_URL=%s DOCS_PUBLIC=%q: DocsPublic=%v, ingin %v", k.baseURL, k.docsPublic, cfg.DocsPublic, k.ingin)
		}
	}
}
