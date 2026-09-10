package auth

import (
	"strings"
	"testing"
	"time"
)

// NOL DI DEPAN HARUS BERTAHAN.
//
// "042317" adalah kode yang sah. Memangkasnya jadi "42317" membuat kode itu
// tidak akan pernah cocok — dan kegagalannya menimpa sekitar satu dari sepuluh
// pengguna, secara acak, tanpa pola yang bisa ditebak dari laporan mereka.
func TestOTPSelaluEnamDigit(t *testing.T) {
	for i := 0; i < 500; i++ {
		kode, hash, _, err := BuatOTP(time.Now())
		if err != nil {
			t.Fatalf("buat OTP: %v", err)
		}
		if len(kode) != 6 {
			t.Fatalf("kode = %q, panjangnya %d — seharusnya 6", kode, len(kode))
		}
		if strings.Trim(kode, "0123456789") != "" {
			t.Fatalf("kode = %q mengandung karakter bukan angka", kode)
		}
		if hash != HashToken(kode) {
			t.Fatalf("hash tidak cocok dengan kodenya")
		}
	}
}

// Kodenya harus berbeda-beda.
//
// Bukan uji keacakan sungguhan — hanya jaring untuk kekeliruan yang paling
// merusak dan paling mudah tak terlihat: seluruh pengguna menerima kode yang
// sama, atau kode yang bisa ditebak dari waktu pembuatannya.
func TestOTPTidakBerulang(t *testing.T) {
	seen := map[string]bool{}
	const n = 300
	for i := 0; i < n; i++ {
		kode, _, _, err := BuatOTP(time.Now())
		if err != nil {
			t.Fatal(err)
		}
		seen[kode] = true
	}
	// Dari sejuta kemungkinan, 300 pengambilan hampir pasti unik. Ambang 290
	// memberi ruang untuk tabrakan yang wajar tanpa meloloskan generator yang
	// macet di segelintir nilai.
	if len(seen) < 290 {
		t.Errorf("hanya %d kode unik dari %d — generatornya tidak acak?", len(seen), n)
	}
}

// Kode yang disimpan tidak boleh bisa dibaca balik jadi kodenya.
func TestOTPDisimpanSebagaiHash(t *testing.T) {
	kode, hash, _, err := BuatOTP(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(hash, kode) {
		t.Errorf("hash %q memuat kodenya sendiri (%q)", hash, kode)
	}
	if hash == kode {
		t.Error("kode disimpan apa adanya, bukan hash-nya")
	}
}

func TestUmurOTPLebihPendekDaripadaTokenReset(t *testing.T) {
	// Bukan angka ajaib: tautan reset dibuka sekali lalu selesai, sedangkan OTP
	// diketik segera setelah emailnya masuk. Kalau suatu saat keduanya jadi
	// sama, itu keputusan yang perlu disengaja — bukan pergeseran diam-diam.
	if UmurOTP >= UmurTokenReset {
		t.Errorf("UmurOTP (%v) >= UmurTokenReset (%v)", UmurOTP, UmurTokenReset)
	}
}
