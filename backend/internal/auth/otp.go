package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// OTP login lewat email.
//
// KENAPA ENAM DIGIT, DAN KENAPA ITU CUKUP.
//
// Enam digit hanya sejuta kemungkinan — angka yang kecil untuk sebuah rahasia.
// Yang membuatnya aman bukan panjangnya melainkan tiga batas di sekelilingnya:
// umurnya pendek, sekali pakai, dan percobaannya dibatasi. Tanpa batas
// percobaan, menebaknya selesai dalam hitungan detik; dengan batas lima,
// peluang menebak benar adalah lima per sejuta.
//
// Panjangnya tetap enam karena kode ini diketik ulang orang dari kotak masuk.
// Kode dua belas karakter memang lebih sulit ditebak, tapi juga lebih sering
// salah ketik — dan setiap salah ketik memakan jatah percobaan yang justru
// menjadi pertahanan utamanya.

const (
	// UmurOTP sengaja jauh lebih pendek daripada token reset.
	//
	// Tautan reset dibuka sekali lalu selesai; OTP diketik segera setelah
	// emailnya masuk. Sepuluh menit menampung email yang telat sampai tanpa
	// meninggalkan kode hidup di kotak masuk lebih lama dari perlunya.
	UmurOTP = 10 * time.Minute

	// MaksPercobaanOTP adalah batas tebakan per kode.
	//
	// Lima, bukan tiga: orang salah ketik, dan mengunci terlalu cepat membuat
	// mereka meminta kode baru berulang kali — yang justru menghasilkan lebih
	// banyak email dan lebih banyak kode hidup sekaligus.
	MaksPercobaanOTP = 5

	digitOTP = 6
)

// BuatOTP membangkitkan kode numerik beserta hash-nya.
//
// crypto/rand, bukan math/rand: kode yang bisa diramalkan dari waktu
// pembuatannya tidak menambah pertahanan apa pun terhadap orang yang tahu
// kapan seseorang menekan tombol masuk.
func BuatOTP(sekarang time.Time) (kode, hash string, berlaku time.Time, err error) {
	maks := big.NewInt(1)
	for i := 0; i < digitOTP; i++ {
		maks.Mul(maks, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, maks)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("bangkitkan OTP: %w", err)
	}
	// Nol di depan dipertahankan: "042317" adalah kode yang sah, dan
	// memangkasnya jadi "42317" membuat kode itu tidak akan pernah cocok.
	kode = fmt.Sprintf("%0*d", digitOTP, n)
	return kode, HashToken(kode), sekarang.Add(UmurOTP), nil
}
