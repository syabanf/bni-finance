package config

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// DAFTAR VARIABEL DI secrets.sh HARUS IKUT BERUBAH SAAT KODE BERUBAH.
//
// `./secrets.sh check` memeriksa berkas env sebelum server dinyalakan: mana
// yang wajib, mana yang penting, dan mana yang tidak dikenal siapa pun. Nilai
// pemeriksaan itu sepenuhnya bergantung pada daftarnya tetap mutakhir.
//
// Tanpa penjaga ini, variabel baru cuma tidak diperiksa — dan yang lebih buruk,
// ia dilaporkan "ASING" pada berkas yang sebenarnya BENAR. Pemeriksa yang
// menuduh keliru akan dimatikan orang, dan setelah itu ia tidak menjaga apa pun
// sambil tetap terlihat hijau.
func TestDaftarSecretsShTidakBasi(t *testing.T) {
	sh, err := os.ReadFile("../../secrets.sh")
	if err != nil {
		t.Fatalf("baca secrets.sh: %v", err)
	}
	src, err := os.ReadFile("config.go")
	if err != nil {
		t.Fatalf("baca config.go: %v", err)
	}

	didaftar := map[string]bool{}
	for _, nama := range []string{"WAJIB", "PENTING", "SANTAI"} {
		for _, v := range isiVariabelSh(t, string(sh), nama) {
			didaftar[v] = true
		}
	}

	// Nama lama sengaja dikenali secrets.sh supaya dilaporkan sebagai BASI,
	// bukan ASING. Ia tidak dibaca config.go untuk konfigurasi — hanya untuk
	// mendeteksi berkas yang belum ikut berganti nama.
	usang := map[string]bool{
		"SMTP_HOST": true, "SMTP_PORT": true, "SMTP_USER": true,
		"SMTP_PASSWORD": true, "SMTP_FROM": true,
	}

	dibaca := map[string]bool{}
	re := regexp.MustCompile(`(?:os\.Getenv|envOr|durationOr|bytesOr)\("([A-Z_0-9]+)"`)
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		dibaca[m[1]] = true
	}
	if len(dibaca) < 10 {
		t.Fatalf("hanya %d variabel terbaca dari config.go — polanya yang rusak, bukan daftarnya", len(dibaca))
	}

	var hilang, hantu []string
	for v := range dibaca {
		if !didaftar[v] && !usang[v] {
			hilang = append(hilang, v)
		}
	}
	for v := range didaftar {
		if !dibaca[v] {
			hantu = append(hantu, v)
		}
	}
	sort.Strings(hilang)
	sort.Strings(hantu)

	if len(hilang) > 0 {
		t.Errorf("dibaca config.go tapi tidak ada di daftar secrets.sh: %v\n"+
			"tambahkan ke WAJIB, PENTING, atau SANTAI — kalau tidak, `check` akan\n"+
			"melaporkannya ASING pada berkas env yang sebenarnya benar", hilang)
	}
	if len(hantu) > 0 {
		t.Errorf("ada di daftar secrets.sh tapi tidak dibaca config.go: %v\n"+
			"variabel itu sudah mati; `check` menuntut orang mengisi sesuatu yang\n"+
			"tidak berpengaruh apa-apa", hantu)
	}
}

// isiVariabelSh mengambil isi penetapan seperti  NAMA="a b c"  dari skrip.
func isiVariabelSh(t *testing.T, sh, nama string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^` + nama + `="([^"]*)"`)
	m := re.FindStringSubmatch(sh)
	if m == nil {
		t.Fatalf("secrets.sh tidak lagi punya daftar %s — penjaga ini ikut mati bersamanya", nama)
	}
	return strings.Fields(m[1])
}
