package domain

import (
	"reflect"
	"testing"
)

// Angka dari lembar klien 19 November 2026.
var rincianKlien = RincianBiaya{
	Aktif: true, PendaftaranUSD: 180.00, KeanggotaanUSD: 648.83, PPNPersen: 11, Training: 500_000,
}

// VAT dan total Dollar harus sama dengan yang tertulis di lembar itu:
// $91,17 dan $71,37.
func TestRincianDollarSesuaiLembarKlien(t *testing.T) {
	if got := rincianKlien.PPNUSD(TypeRegistration); got != 91.17 {
		t.Errorf("VAT pendaftaran = %v, ingin 91.17", got)
	}
	if got := rincianKlien.PPNUSD(TypeRenewal); got != 71.37 {
		t.Errorf("VAT renewal = %v, ingin 71.37", got)
	}
	if got := rincianKlien.TotalUSD(TypeRegistration); got != 920.00 {
		t.Errorf("total pendaftaran = %v, ingin 920.00", got)
	}
	if got := rincianKlien.TotalUSD(TypeRenewal); got != 720.20 {
		t.Errorf("total renewal = %v, ingin 720.20", got)
	}
}

func TestBarisPendaftaran(t *testing.T) {
	got := rincianKlien.Baris(TypeRegistration, 16_000)
	ingin := []BarisBiaya{
		{BiayaPendaftaran, "Registration Fee", 2_880_000},
		{BiayaKeanggotaan, "Membership Fee", 10_381_280},
		{BiayaPPN, "VAT (11%)", 1_458_720},
		{BiayaTraining, "Online Training Investment Bundling", 500_000},
	}
	if !reflect.DeepEqual(got, ingin) {
		t.Errorf("baris pendaftaran:\n dapat %+v\n ingin %+v", got, ingin)
	}
	// 920 × 16.000 + 500.000.
	if total := JumlahBaris(got); total != 15_220_000 {
		t.Errorf("total pendaftaran = %d, ingin 15.220.000", total)
	}
}

// Renewal tidak memuat Registration Fee maupun training.
func TestBarisRenewal(t *testing.T) {
	got := rincianKlien.Baris(TypeRenewal, 16_000)
	ingin := []BarisBiaya{
		{BiayaKeanggotaan, "Membership Fee", 10_381_280},
		{BiayaPPN, "VAT (11%)", 1_141_920},
	}
	if !reflect.DeepEqual(got, ingin) {
		t.Errorf("baris renewal:\n dapat %+v\n ingin %+v", got, ingin)
	}
	if total := JumlahBaris(got); total != 11_523_200 {
		t.Errorf("total renewal = %d, ingin 11.523.200", total)
	}
}

// Dengan kurs yang tidak bulat, tiap baris dibulatkan sendiri dan totalnya
// adalah jumlah baris itu. Total Dollar × kurs bisa berbeda beberapa rupiah,
// dan yang tercetak di invoice harus selalu berjumlah sama dengan barisnya.
func TestTotalAdalahJumlahBaris(t *testing.T) {
	const kurs = 16_233
	baris := rincianKlien.Baris(TypeRegistration, kurs)
	// 180 × 16.233 = 2.921.940; 648,83 × 16.233 = 10.532.457,39;
	// 91,17 × 16.233 = 1.479.962,61.
	ingin := []int64{2_921_940, 10_532_457, 1_479_963, 500_000}
	for i, b := range baris {
		if b.Jumlah != ingin[i] {
			t.Errorf("baris %s = %d, ingin %d", b.Kode, b.Jumlah, ingin[i])
		}
	}
	if total := JumlahBaris(baris); total != 15_434_360 {
		t.Errorf("total = %d, ingin 15.434.360", total)
	}
}

// Komponen nol tidak mencetak baris kosong di invoice.
func TestKomponenNolTidakMenjadiBaris(t *testing.T) {
	r := RincianBiaya{Aktif: true, KeanggotaanUSD: 100}
	got := r.Baris(TypeRegistration, 16_000)
	if len(got) != 1 || got[0].Kode != BiayaKeanggotaan {
		t.Errorf("dapat %+v, ingin hanya Membership Fee", got)
	}
}

func TestLabelPPNPecahan(t *testing.T) {
	r := RincianBiaya{KeanggotaanUSD: 100, PPNPersen: 11.5}
	baris := r.Baris(TypeRenewal, 10_000)
	if baris[1].Nama != "VAT (11.5%)" {
		t.Errorf("label = %q", baris[1].Nama)
	}
}

func TestValidasiRincian(t *testing.T) {
	salah := []RincianBiaya{
		{PendaftaranUSD: -1}, {KeanggotaanUSD: -1}, {PPNPersen: -1}, {PPNPersen: 101}, {Training: -1},
	}
	for _, r := range salah {
		if r.Validate() == nil {
			t.Errorf("rincian %+v seharusnya ditolak", r)
		}
	}
	if err := rincianKlien.Validate(); err != nil {
		t.Errorf("rincian klien ditolak: %v", err)
	}
}
