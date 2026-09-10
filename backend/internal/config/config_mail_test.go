package config

import "testing"

// PERGANTIAN NAMA VARIABEL HARUS TERDETEKSI, BUKAN GAGAL DIAM-DIAM.
//
// Pengiriman email pindah dari SMTP ke REST API Resend, dan nama variabelnya
// ikut berubah. Lingkungan yang belum ikut diubah — .env di mesin orang lain,
// secret di panel hosting — kehilangan kemampuan mengirim email tanpa satu pun
// galat: kredensialnya masih di sana, isinya masih benar, hanya namanya yang
// tidak dibaca lagi.
//
// Yang muncul cuma "email belum dikonfigurasi", yang menyuruh orang mengisi
// sesuatu yang menurut mereka SUDAH diisi. Penanda ini yang membuat
// peringatannya menyebutkan penggantian namanya.
func TestSMTPUsangTerdeteksi(t *testing.T) {
	kasus := []struct {
		nama string
		env  map[string]string
		mau  bool
	}{
		{"hanya password lama", map[string]string{"SMTP_PASSWORD": "re_lama"}, true},
		{"hanya host lama", map[string]string{"SMTP_HOST": "smtp.resend.com"}, true},
		{"keduanya", map[string]string{"SMTP_HOST": "smtp.resend.com", "SMTP_PASSWORD": "re_lama"}, true},
		// Nama baru terpasang: tidak ada yang perlu diperingatkan.
		{"sudah pakai nama baru", map[string]string{"RESEND_API_KEY": "re_baru"}, false},
		// Sisa komentar atau baris kosong tidak boleh dihitung sebagai
		// "terpasang" — memperingatkan orang yang sudah beres itu melelahkan,
		// dan peringatan yang sering keliru berhenti dibaca.
		{"baris lama kosong", map[string]string{"SMTP_HOST": "", "SMTP_PASSWORD": "   "}, false},
		{"tidak ada apa-apa", map[string]string{}, false},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			for _, v := range []string{"SMTP_HOST", "SMTP_PASSWORD", "RESEND_API_KEY", "MAIL_FROM"} {
				t.Setenv(v, "")
			}
			for nama, isi := range k.env {
				t.Setenv(nama, isi)
			}
			// DATABASE_URL wajib; Load() gagal tanpa itu dan tidak akan sampai
			// ke bagian yang diuji di sini.
			t.Setenv("DATABASE_URL", "postgres://x/y")
			t.Setenv("JWT_SECRET", "cukup-panjang-untuk-lolos-pemeriksaan-minimal")

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.SMTPUsang != k.mau {
				t.Errorf("SMTPUsang = %v, seharusnya %v", cfg.SMTPUsang, k.mau)
			}
		})
	}
}
