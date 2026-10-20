package paperid

import (
	"strings"
	"testing"
)

// Riwayat invoice mencatat ke mana Paper.id diminta mengirim. Paper.id tidak
// melaporkan status dibaca, jadi inilah satu-satunya jejak untuk menelusuri
// "member tidak menerima invoice": tujuan yang salah, atau kanal yang dilewati.
func TestKanalKirim(t *testing.T) {
	kasus := []struct {
		nama               string
		email, whatsapp    bool
		alamat, telepon    string
		harusAda, dilarang []string
	}{
		{"dua kanal", true, true, "a@contoh.invalid", "0812-3456-7890",
			[]string{"lewat WhatsApp 6281234567890", "email a@contoh.invalid"}, []string{"dilewati"}},
		{"tanpa email", false, true, "", "081234567890",
			[]string{"lewat WhatsApp 6281234567890", "email dilewati"}, []string{"email a@"}},
		{"semua kanal mati", false, false, "a@contoh.invalid", "081234567890",
			[]string{"tanpa notifikasi"}, []string{"lewat"}},
	}
	for _, k := range kasus {
		got := kanalKirim(k.email, k.whatsapp, k.alamat, k.telepon)
		for _, s := range k.harusAda {
			if !strings.Contains(got, s) {
				t.Errorf("%s: %q tidak memuat %q", k.nama, got, s)
			}
		}
		for _, s := range k.dilarang {
			if strings.Contains(got, s) {
				t.Errorf("%s: %q tidak boleh memuat %q", k.nama, got, s)
			}
		}
	}
}
