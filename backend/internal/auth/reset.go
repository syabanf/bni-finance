package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"
)

// Reset kata sandi lewat email.
//
// DUA NILAI YANG BERBEDA, dan bedanya menentukan keamanannya:
//
//	token   dikirim ke email, tidak pernah disimpan
//	hash    disimpan di basis data, tidak pernah dikirim
//
// Token reset setara kata sandi sementara — siapa pun yang memegangnya bisa
// mengambil alih akun. Menyimpannya apa adanya berarti tabelnya sendiri menjadi
// daftar kunci: satu dump basis data, dan setiap token yang masih berlaku bisa
// langsung dipakai. Dengan hash, isi tabel itu tidak berguna bagi siapa pun.

const (
	// UmurTokenReset sengaja pendek.
	//
	// Tautan reset hidup di kotak masuk, dan kotak masuk bertahan lama:
	// perangkat yang dipinjam, email yang diteruskan, akun email lama yang
	// masih terbuka di suatu tempat. Tiga puluh menit cukup untuk membuka
	// email dan mengetik kata sandi baru, dan terlalu singkat untuk berguna
	// bagi siapa pun yang menemukannya belakangan.
	UmurTokenReset = 30 * time.Minute

	// panjangToken dalam byte sebelum dikodekan.
	//
	// 32 byte = 256 bit. Menebaknya bukan sesuatu yang bisa dilakukan, dan itu
	// satu-satunya pertahanan yang tersisa kalau emailnya sampai ke orang lain.
	panjangToken = 32
)

// TokenReset adalah pasangan token-dan-hash sekali pakai.
type TokenReset struct {
	// Token dikirim ke pengguna. JANGAN disimpan.
	Token string
	// Hash disimpan. JANGAN dikirim.
	Hash    string
	Berlaku time.Time
}

// BuatTokenReset membangkitkan token acak beserta hash-nya.
func BuatTokenReset(sekarang time.Time) (TokenReset, error) {
	b := make([]byte, panjangToken)
	if _, err := rand.Read(b); err != nil {
		return TokenReset{}, fmt.Errorf("bangkitkan token reset: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return TokenReset{
		Token:   token,
		Hash:    HashToken(token),
		Berlaku: sekarang.Add(UmurTokenReset),
	}, nil
}

// HashToken mengembalikan hash yang disimpan untuk sebuah token.
//
// SHA-256 polos, bukan PBKDF2 seperti kata sandi — dan itu bukan kelalaian.
// Peregangan kunci melindungi rahasia BERENTROPI RENDAH yang dipilih manusia
// dari serangan tebak. Token ini 256 bit acak: tidak ada yang bisa ditebak,
// jadi biaya peregangan hanya akan memperlambat setiap permintaan reset yang
// sah tanpa menambah satu pun hambatan bagi penyerang.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// TokenCocok membandingkan dua hash tanpa membocorkan lewat waktu eksekusi.
func TokenCocok(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
