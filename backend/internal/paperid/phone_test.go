package paperid

import "testing"

// Nomor berawalan 0 yang lolos ke Paper.id berarti WhatsApp tidak sampai ke
// siapa pun, sementara sistem melaporkan sukses. Tabel ini mengunci konversinya.
func TestNormalizePhone(t *testing.T) {
	kasus := []struct{ masuk, mau string }{
		{"082240274833", "6282240274833"},      // format lokal — yang tersimpan di data member
		{"6282240274833", "6282240274833"},     // sudah internasional, jangan diubah
		{"+62 822-4027-4833", "6282240274833"}, // dengan tanda baca dan plus
		{"0822 4027 4833", "6282240274833"},    // dengan spasi
		{"(0822) 4027-4833", "6282240274833"},
		{"82240274833", "6282240274833"}, // tanpa awalan apa pun
		{"", ""},                         // kosong tetap kosong
		{"   ", ""},                      // hanya spasi
		{"abc", ""},                      // tanpa angka sama sekali
	}
	for _, k := range kasus {
		if got := normalizePhone(k.masuk); got != k.mau {
			t.Errorf("normalizePhone(%q) = %q, mau %q", k.masuk, got, k.mau)
		}
	}
}
