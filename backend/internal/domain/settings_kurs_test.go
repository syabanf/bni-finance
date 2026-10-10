package domain

import "testing"

// Rupiah yang tercetak di invoice member lahir dari fungsi ini, jadi
// pembulatannya harus bisa diramalkan.
func TestTurunkanRupiah(t *testing.T) {
	kasus := []struct {
		usd  float64
		rate int64
		mau  int64
	}{
		// Nilai bawaan di db/init.sql harus menghasilkan Rupiah bawaannya
		// persis, kalau tidak halaman Pengaturan akan menampilkan angka
		// yang berbeda dari yang disimpan.
		{1125.00, 16000, 18_000_000},
		{793.75, 16000, 12_700_000},
		// Dibulatkan ke terdekat, bukan dipotong.
		{1.005, 1000, 1005},
		{0.0004, 1000, 0},
		{0.0005, 1000, 1},
		{1234.56, 15987, 19_736_911}, // 19.736.910,72
		{0, 16000, 0},
	}
	for _, k := range kasus {
		if got := TurunkanRupiah(k.usd, k.rate); got != k.mau {
			t.Errorf("TurunkanRupiah(%v, %d) = %d, seharusnya %d", k.usd, k.rate, got, k.mau)
		}
	}
}

// Validasi menolak kurs nol: membagi dengan nol tidak terjadi di sini, tapi
// kurs nol membuat setiap harga Rupiah jadi nol, dan invoice Rp 0 tetap
// terkirim ke member sebagai dokumen yang sah.
func TestUpdateFeeSettingsInputMenolakKursNol(t *testing.T) {
	nol := int64(0)
	if err := (UpdateFeeSettingsInput{UsdRate: &nol}).Validate(); err == nil {
		t.Error("kurs nol harus ditolak")
	}
	negatif := -1.0
	if err := (UpdateFeeSettingsInput{RenewalFeeUSD: &negatif}).Validate(); err == nil {
		t.Error("USD negatif harus ditolak")
	}
	satu := int64(1)
	if err := (UpdateFeeSettingsInput{UsdRate: &satu}).Validate(); err != nil {
		t.Errorf("kurs 1 sah, dapat %v", err)
	}
}
