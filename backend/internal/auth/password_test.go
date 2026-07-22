package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	const pw = "sandi-rahasia-123"

	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if strings.Contains(hash, pw) {
		t.Fatal("hash memuat kata sandi asli")
	}
	if !VerifyPassword(hash, pw) {
		t.Error("kata sandi benar ditolak")
	}
	if VerifyPassword(hash, "sandi-salah") {
		t.Error("kata sandi salah diterima")
	}
	// Off-by-one on either end must not pass.
	if VerifyPassword(hash, pw+"x") || VerifyPassword(hash, pw[:len(pw)-1]) {
		t.Error("kata sandi mirip diterima")
	}
}

func TestHashIsSalted(t *testing.T) {
	a, _ := HashPassword("sama")
	b, _ := HashPassword("sama")
	if a == b {
		t.Error("dua hash dari kata sandi sama harus berbeda — salt tidak dipakai")
	}
	if !VerifyPassword(a, "sama") || !VerifyPassword(b, "sama") {
		t.Error("kedua hash harus tetap bisa diverifikasi")
	}
}

// A corrupt or truncated row must read as "wrong password", never panic and
// never accidentally match.
func TestVerifyRejectsMalformedHash(t *testing.T) {
	bad := []string{
		"",
		"bukan-hash",
		"pbkdf2_sha256$600000$hanya-tiga-bagian",
		"pbkdf2_sha256$abc$c2FsdA$aGFzaA",          // iterasi bukan angka
		"pbkdf2_sha256$0$c2FsdA$aGFzaA",            // iterasi nol
		"bcrypt$600000$c2FsdA$aGFzaA",              // algoritma lain
		"pbkdf2_sha256$600000$!!!bukan-b64$aGFzaA", // salt rusak
		"pbkdf2_sha256$600000$c2FsdA$!!!bukan-b64", // hash rusak
	}
	for _, h := range bad {
		if VerifyPassword(h, "apa pun") {
			t.Errorf("hash rusak %q diterima", h)
		}
	}
}

func TestVerifyRespectsStoredIterationCount(t *testing.T) {
	// The iteration count lives in the hash, so it can be raised later without
	// invalidating passwords already stored at the old cost.
	hash, err := HashPassword("halo")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	parts := strings.Split(hash, "$")
	if len(parts) != hashFieldCount {
		t.Fatalf("format hash tak terduga: %q", hash)
	}
	// Tampering with the recorded cost must break verification, not silently
	// re-derive at a cheaper setting.
	tampered := strings.Join([]string{parts[0], "1000", parts[2], parts[3]}, "$")
	if VerifyPassword(tampered, "halo") {
		t.Error("hash dengan iterasi yang diubah tidak boleh cocok")
	}
}
