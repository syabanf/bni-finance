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

// Satuan waktu penghitungan denda.
type SatuanDenda string

const (
	SatuanHari   SatuanDenda = "hari"
	SatuanMinggu SatuanDenda = "minggu"
	SatuanBulan  SatuanDenda = "bulan"
)

// hariPerSatuan mengubah satuan menjadi jumlah hari.
//
// Bulan dihitung 30 hari, bukan panjang bulan kalender yang sebenarnya. Denda
// yang besarnya bergantung pada apakah keterlambatannya jatuh di Februari atau
// Juli tidak bisa dijelaskan kepada member — dan yang perlu dijelaskan adalah
// angkanya, bukan almanaknya.
func (s SatuanDenda) hari() int {
	switch s {
	case SatuanMinggu:
		return 7
	case SatuanBulan:
		return 30
	default:
		return 1
	}
}

// JenisDenda menentukan dendanya nominal tetap atau persentase tagihan.
type JenisDenda string

const (
	DendaRupiah JenisDenda = "rupiah"
	DendaPersen JenisDenda = "persen"
)

// LateFeeRule adalah pengaturan denda dari app_settings.
type LateFeeRule struct {
	Aktif bool       `json:"aktif"`
	Jenis JenisDenda `json:"jenis"`
	// Satuan waktu: dendanya bertambah tiap satu satuan keterlambatan.
	Satuan SatuanDenda `json:"satuan"`
	// PerSatuan adalah nominal rupiah per satuan, dipakai saat Jenis rupiah.
	PerSatuan int64 `json:"perSatuan"`
	// PersenPerSatuan adalah persentase dari nominal invoice per satuan,
	// dipakai saat Jenis persen. Ditulis sebagai 2 untuk 2%, bukan 0,02.
	PersenPerSatuan float64 `json:"persenPerSatuan"`
	MaksHari        int     `json:"maksHari"`
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
	if !r.Aktif {
		return LateFee{}
	}
	if r.Jenis == DendaPersen {
		if r.PersenPerSatuan <= 0 {
			return LateFee{}
		}
	} else if r.PerSatuan <= 0 {
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

	// SATUAN YANG SUDAH GENAP, bukan pecahannya.
	//
	// Denda "per minggu" pada hari keempat adalah nol, bukan empat-per-tujuh
	// minggu. Membebankan denda satu minggu penuh di hari pertama adalah hal
	// yang tidak seorang pun harapkan dari kata "per minggu", dan pecahan
	// minggu adalah angka yang tidak bisa dijelaskan di kuitansi.
	satuan := int64(hari / r.Satuan.hari())
	if satuan <= 0 {
		return LateFee{HariTelat: hari, Tercapaikan: batas}
	}

	var nominal int64
	if r.Jenis == DendaPersen {
		// Dibulatkan ke bawah ke rupiah penuh. Denda berkoma tidak bisa
		// ditransfer, dan membulatkan ke atas berarti menagih lebih dari
		// aturannya sendiri.
		nominal = int64(float64(inv.Amount) * r.PersenPerSatuan / 100 * float64(satuan))
	} else {
		nominal = satuan * r.PerSatuan
	}
	return LateFee{HariTelat: hari, Nominal: nominal, Tercapaikan: batas}
}
