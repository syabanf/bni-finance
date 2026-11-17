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

// Pengirim di domain lain dari tautannya adalah yang membuat situs ditandai
// phishing; pengirim di subdomain (pola umum penyedia email) harus tetap lolos.
func TestPengirimSedomain(t *testing.T) {
	kasus := []struct {
		dari, baseURL string
		ingin         bool
	}{
		{"no-reply@bnisystem.id", "https://bnisystem.id", true},
		{"Finance Hub <no-reply@mail.bnisystem.id>", "https://bnisystem.id", true},
		{"no-reply@bnisystem.id", "https://app.bnisystem.id", true},
		{"No-Reply@BNISystem.ID", "https://bnisystem.id/", true},
		{"no-reply@reddie.id", "https://bnisystem.id", false},
		{"Finance Hub <no-reply@reddie.id>", "https://bnisystem.id", false},
		{"no-reply@jahatbnisystem.id", "https://bnisystem.id", false},
		{"no-reply@bnisystem.id.jahat.com", "https://bnisystem.id", false},
		{"bukan-alamat", "https://bnisystem.id", false},
		{"no-reply@reddie.id", "http://localhost:5173", true},
	}
	for _, k := range kasus {
		if got := PengirimSedomain(k.dari, k.baseURL); got != k.ingin {
			t.Errorf("PengirimSedomain(%q, %q) = %v, ingin %v", k.dari, k.baseURL, got, k.ingin)
		}
	}
}
