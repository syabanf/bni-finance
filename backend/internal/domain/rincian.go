package domain

import (
	"fmt"
	"math"
	"strconv"
)

// Kode komponen biaya. Kodenya stabil; nama yang tercetak di invoice boleh
// berubah tanpa mengubah cara baris lama dibaca.
const (
	BiayaPendaftaran = "registration"
	BiayaKeanggotaan = "membership"
	BiayaPPN         = "vat"
	BiayaTraining    = "training"
)

// BarisBiaya adalah satu komponen harga dalam Rupiah, seperti yang tercetak
// sebagai satu baris di invoice.
type BarisBiaya struct {
	Kode   string `json:"kode"`
	Nama   string `json:"nama"`
	Jumlah int64  `json:"jumlah"`
}

// RincianBiaya memecah harga menjadi komponennya.
//
//	Pendaftaran  Registration Fee + Membership Fee + VAT + Online Training
//	Renewal      Membership Fee + VAT
//
// VAT dihitung dari komponen Dollar sebelum pajak, dibulatkan ke sen, lalu
// dikalikan kurs. Biaya training ditetapkan dalam Rupiah dan tidak kena VAT.
//
// Aktif=false berarti sistem memakai satu harga Dollar per tipe invoice
// seperti sebelum rincian ini ada. Rincian baru berlaku saat admin
// menyimpannya dengan Aktif=true: sebuah deploy tidak boleh mengubah harga
// yang ditagihkan.
type RincianBiaya struct {
	Aktif          bool    `json:"aktif"`
	PendaftaranUSD float64 `json:"pendaftaranUsd"`
	KeanggotaanUSD float64 `json:"keanggotaanUsd"`
	PPNPersen      float64 `json:"ppnPersen"`
	// Training dalam Rupiah.
	Training int64 `json:"training"`
}

func (r RincianBiaya) Validate() error {
	switch {
	case r.PendaftaranUSD < 0:
		return fmt.Errorf("rincian.pendaftaranUsd tidak boleh negatif")
	case r.KeanggotaanUSD < 0:
		return fmt.Errorf("rincian.keanggotaanUsd tidak boleh negatif")
	case r.PPNPersen < 0 || r.PPNPersen > 100:
		return fmt.Errorf("rincian.ppnPersen harus di antara 0 dan 100")
	case r.Training < 0:
		return fmt.Errorf("rincian.training tidak boleh negatif")
	}
	return nil
}

func sen(usd float64) float64 { return math.Round(usd*100) / 100 }

// dasarUSD adalah komponen Dollar sebelum pajak untuk satu tipe invoice.
func (r RincianBiaya) dasarUSD(t InvoiceType) float64 {
	if t == TypeRegistration {
		return r.PendaftaranUSD + r.KeanggotaanUSD
	}
	return r.KeanggotaanUSD
}

// PPNUSD adalah pajak dalam Dollar, dibulatkan ke sen.
func (r RincianBiaya) PPNUSD(t InvoiceType) float64 {
	return sen(r.dasarUSD(t) * r.PPNPersen / 100)
}

// TotalUSD adalah bagian Dollar dari harga, termasuk pajak. Biaya training
// tidak ikut karena ditetapkan dalam Rupiah.
func (r RincianBiaya) TotalUSD(t InvoiceType) float64 {
	return sen(r.dasarUSD(t) + r.PPNUSD(t))
}

// Baris menghitung komponen dalam Rupiah pada kurs yang diberikan.
//
// Tiap baris dibulatkan ke rupiah lebih dulu, dan totalnya adalah jumlah
// baris-baris itu. Mengalikan total Dollar dengan kurs bisa berbeda beberapa
// rupiah dari jumlah barisnya, dan invoice yang barisnya tidak berjumlah
// sama dengan totalnya menimbulkan pertanyaan pada setiap tagihan.
//
// Komponen bernilai nol tidak menghasilkan baris.
func (r RincianBiaya) Baris(t InvoiceType, kurs int64) []BarisBiaya {
	var out []BarisBiaya
	tambah := func(kode, nama string, jumlah int64) {
		if jumlah > 0 {
			out = append(out, BarisBiaya{Kode: kode, Nama: nama, Jumlah: jumlah})
		}
	}
	if t == TypeRegistration {
		tambah(BiayaPendaftaran, "Registration Fee", TurunkanRupiah(r.PendaftaranUSD, kurs))
	}
	tambah(BiayaKeanggotaan, "Membership Fee", TurunkanRupiah(r.KeanggotaanUSD, kurs))
	tambah(BiayaPPN, "VAT ("+strconv.FormatFloat(r.PPNPersen, 'f', -1, 64)+"%)", TurunkanRupiah(r.PPNUSD(t), kurs))
	if t == TypeRegistration {
		tambah(BiayaTraining, "Online Training Investment Bundling", r.Training)
	}
	return out
}

// JumlahBaris menjumlahkan baris-baris biaya.
func JumlahBaris(baris []BarisBiaya) int64 {
	var total int64
	for _, b := range baris {
		total += b.Jumlah
	}
	return total
}
