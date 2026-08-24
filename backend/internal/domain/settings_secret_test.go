package domain_test

import (
	"strings"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
)

// Nama-nama ini pernah BOCOR apa adanya ke setiap pengguna yang sudah masuk,
// termasuk yang bukan admin. Dibuktikan dengan menulisnya lewat API lalu
// membacanya kembali memakai token pengguna biasa: lima dari enam kembali
// sebagai "RAHASIA-BOCOR-12345", bukan "••••••".
//
// Daftar ini adalah pagarnya. Menyempitkan secretKeyParts lagi akan langsung
// memerahkan tes ini, bukan diam-diam membocorkan kredensial berikutnya.
func TestKunciBerbentukKredensialSelaluDisamarkan(t *testing.T) {
	rahasia := []string{
		// yang dulu lolos
		"supabase_service_role_key", "signing_key", "encryption_key",
		"db_pass", "connection_string",
		// yang sejak awal tertutup — jangan sampai ikut hilang
		"paperid_client_secret", "bni_vm_token", "admin_password",
		"x_api_key", "google_apikey", "service_credential", "private_pem",
		// ragam lain yang lazim
		"jwt_secret", "bearer_token", "auth_header", "webhook_signature",
		"password_salt", "cert_chain", "dsn", "database_url",
	}
	for _, k := range rahasia {
		if !domain.IsSecretKey(k) {
			t.Errorf("%q TIDAK disamarkan — nilainya terbaca apa adanya oleh pengguna biasa", k)
		}
		// Huruf besar dan spasi tidak boleh jadi jalan pintas.
		if !domain.IsSecretKey(strings.ToUpper(" " + k + " ")) {
			t.Errorf("%q lolos saat ditulis dengan huruf besar/spasi", k)
		}
	}
}

// Sebaliknya: setting yang memang bukan kredensial harus tetap terbaca, kalau
// tidak admin kehilangan kemampuan melihat konfigurasinya sendiri.
func TestSettingBiasaTidakIkutDisamarkan(t *testing.T) {
	biasa := []string{
		"invoice_draft_days_before", "invoice_due_days_after",
		"paperid_send_email", "paperid_send_whatsapp", "self_payment_mode",
	}
	for _, k := range biasa {
		if domain.IsSecretKey(k) {
			t.Errorf("%q ikut disamarkan padahal hanya konfigurasi biasa", k)
		}
	}
}

// Penyamaran hanya berlaku saat DIBACA. Menulis tetap harus bisa, kalau tidak
// kredensialnya tidak akan pernah bisa dipasang sejak awal.
func TestRedactHanyaMengubahNilaiYangTerisi(t *testing.T) {
	terisi := domain.AppSetting{Key: "bni_vm_token", Value: "asli"}.Redact()
	if terisi.Value != domain.MaskedValue || !terisi.Masked {
		t.Errorf("nilai rahasia tidak disamarkan: %+v", terisi)
	}
	// Yang kosong dibiarkan kosong — "••••••" atas nilai kosong membuat orang
	// mengira kredensialnya sudah terpasang padahal belum.
	kosong := domain.AppSetting{Key: "bni_vm_token", Value: ""}.Redact()
	if kosong.Value != "" || kosong.Masked {
		t.Errorf("setting kosong ikut disamarkan: %+v", kosong)
	}
}
