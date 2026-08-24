package domain

import "time"

// Denda keterlambatan — DIHITUNG SAAT DIBACA, tidak pernah disimpan dan tidak
// pernah ditagih otomatis.
//
// Keputusan itu yang membuat fiturnya kecil, dan alasannya bukan kemalasan:
// denda yang menempel dan tumbuh di invoice memaksa nominalnya berubah seiring
// waktu. Setiap perubahan nominal pada invoice yang SUDAH terkirim ke Paper.id
// menghasilkan tagihan yang tidak lagi cocok dengan yang diterima member — dan
// nomor invoice Paper.id tidak bisa diterbitkan ulang.
//
// Karena tidak disimpan, tidak ada pekerjaan berkala, tidak ada baris yang bisa
// basi, dan mematikan fiturnya cukup dengan satu sakelar.

// LateFeeRule adalah pengaturan denda dari app_settings.
type LateFeeRule struct {
	Aktif    bool  `json:"aktif"`
	PerHari  int64 `json:"perHari"`
	MaksHari int   `json:"maksHari"`
}

// LateFee adalah hasil hitungan untuk satu invoice.
type LateFee struct {
	HariTelat int   `json:"hariTelat"`
	Nominal   int64 `json:"nominal"`
	// Tercapaikan menandai dendanya sudah menyentuh batas maksimum, sehingga
	// angkanya berhenti tumbuh. Ditampilkan supaya orang tidak menyimpulkan
	// sistemnya berhenti menghitung.
	Tercapaikan bool `json:"batasTercapai"`
}

// Hitung mengembalikan denda untuk satu invoice pada saat `now`.
//
// Yang TIDAK menumbuhkan denda, dan masing-masing punya alasannya sendiri:
//
//	draft       belum ditagihkan sama sekali; tidak ada yang terlambat
//	paid        uangnya sudah masuk
//	cancelled   tagihannya ditarik kembali
//	terminated  hubungannya berakhir; menagih denda atasnya tidak masuk akal
//
// Hanya `sent` dan `overdue` yang menumbuhkannya — keduanya berarti tagihan
// masih berdiri dan belum dibayar.
func (r LateFeeRule) Hitung(inv Invoice, now time.Time) LateFee {
	if !r.Aktif || r.PerHari <= 0 {
		return LateFee{}
	}
	switch inv.Status {
	case StatusSent, StatusOverdue:
	default:
		return LateFee{}
	}

	// Dibandingkan per HARI KALENDER, bukan per selisih jam. Invoice yang jatuh
	// tempo pukul 00:00 dan dilihat pukul 23:00 di hari yang sama belum telat
	// satu hari pun; membandingkan durasi mentah akan membulatkannya menjadi
	// nol juga, tapi pada zona waktu yang berbeda hasilnya bisa meleset satu
	// hari — dan denda yang meleset satu hari adalah denda yang salah.
	jatuh := time.Date(inv.DueDate.Year(), inv.DueDate.Month(), inv.DueDate.Day(), 0, 0, 0, 0, time.UTC)
	kini := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	hari := int(kini.Sub(jatuh).Hours() / 24)
	if hari <= 0 {
		return LateFee{}
	}

	batas := false
	if r.MaksHari > 0 && hari > r.MaksHari {
		hari = r.MaksHari
		batas = true
	}
	return LateFee{HariTelat: hari, Nominal: int64(hari) * r.PerHari, Tercapaikan: batas}
}
