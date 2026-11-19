package settings

import (
	"context"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
)

// storePalsu menyimpan biaya di memori dan menerapkan pembaruan seperti
// Repository: hanya field yang dikirim yang berubah.
type storePalsu struct {
	Store
	fees domain.FeeSettings
}

func (s *storePalsu) GetFees(context.Context) (*domain.FeeSettings, error) {
	f := s.fees
	return &f, nil
}

func (s *storePalsu) UpdateFees(_ context.Context, in domain.UpdateFeeSettingsInput) (*domain.FeeSettings, error) {
	if in.RegistrationFee != nil {
		s.fees.RegistrationFee = *in.RegistrationFee
	}
	if in.RenewalFee != nil {
		s.fees.RenewalFee = *in.RenewalFee
	}
	if in.RegistrationFeeUSD != nil {
		s.fees.RegistrationFeeUSD = *in.RegistrationFeeUSD
	}
	if in.RenewalFeeUSD != nil {
		s.fees.RenewalFeeUSD = *in.RenewalFeeUSD
	}
	if in.UsdRate != nil {
		s.fees.UsdRate = *in.UsdRate
	}
	if in.Rincian != nil {
		s.fees.Rincian = *in.Rincian
	}
	f := s.fees
	return &f, nil
}

// Keadaan setelah deploy: harga lama, rincian terisi bawaan tetapi belum aktif.
func biayaLama() domain.FeeSettings {
	return domain.FeeSettings{
		RegistrationFee: 18_000_000, RenewalFee: 12_700_000,
		RegistrationFeeUSD: 1125.00, RenewalFeeUSD: 793.75, UsdRate: 16_000,
		Rincian: domain.RincianBiaya{PendaftaranUSD: 180.00, KeanggotaanUSD: 648.83, PPNPersen: 11, Training: 500_000},
	}
}

// Selama rincian belum diaktifkan, harga tetap dihitung dari satu harga Dollar
// per tipe. Worker kurs yang berjalan setiap hari tidak boleh ikut mengaktifkan
// harga baru.
func TestRincianTidakAktifTidakMengubahHarga(t *testing.T) {
	store := &storePalsu{fees: biayaLama()}
	kurs := int64(16_500)

	fees, err := NewService(store).UpdateFees(context.Background(), domain.UpdateFeeSettingsInput{UsdRate: &kurs})
	if err != nil {
		t.Fatalf("UpdateFees: %v", err)
	}
	if fees.RegistrationFee != 18_562_500 || fees.RenewalFee != 13_096_875 {
		t.Errorf("harga = %d dan %d, ingin 1.125 × 16.500 dan 793,75 × 16.500", fees.RegistrationFee, fees.RenewalFee)
	}
	if fees.RegistrationFeeUSD != 1125.00 {
		t.Errorf("harga Dollar berubah menjadi %v", fees.RegistrationFeeUSD)
	}
}

// Begitu admin menyimpan rincian aktif, kedua harga diturunkan dari komponen.
func TestMengaktifkanRincianMenurunkanHarga(t *testing.T) {
	store := &storePalsu{fees: biayaLama()}
	rincian := biayaLama().Rincian
	rincian.Aktif = true

	fees, err := NewService(store).UpdateFees(context.Background(), domain.UpdateFeeSettingsInput{Rincian: &rincian})
	if err != nil {
		t.Fatalf("UpdateFees: %v", err)
	}
	if fees.RegistrationFee != 15_220_000 {
		t.Errorf("pendaftaran = %d, ingin 15.220.000 (920 × 16.000 + 500.000)", fees.RegistrationFee)
	}
	if fees.RenewalFee != 11_523_200 {
		t.Errorf("renewal = %d, ingin 11.523.200 (720,20 × 16.000)", fees.RenewalFee)
	}
	if fees.RegistrationFeeUSD != 920.00 || fees.RenewalFeeUSD != 720.20 {
		t.Errorf("harga Dollar = %v dan %v, ingin 920.00 dan 720.20", fees.RegistrationFeeUSD, fees.RenewalFeeUSD)
	}
}

// Dengan rincian aktif, perubahan kurs menghitung ulang dari komponen, dan
// harga Dollar yang dikirim langsung ditimpa: komponenlah sumbernya.
func TestRincianAktifMengikutiKursDanMenimpaDollar(t *testing.T) {
	awal := biayaLama()
	awal.Rincian.Aktif = true
	store := &storePalsu{fees: awal}
	kurs, dollarLain := int64(16_233), 5000.0

	fees, err := NewService(store).UpdateFees(context.Background(),
		domain.UpdateFeeSettingsInput{UsdRate: &kurs, RegistrationFeeUSD: &dollarLain})
	if err != nil {
		t.Fatalf("UpdateFees: %v", err)
	}
	if fees.RegistrationFee != 15_434_360 {
		t.Errorf("pendaftaran = %d, ingin 15.434.360", fees.RegistrationFee)
	}
	if fees.RegistrationFeeUSD != 920.00 {
		t.Errorf("harga Dollar = %v, ingin tetap 920.00 dari komponen", fees.RegistrationFeeUSD)
	}
	// Harga per tipe selalu sama dengan jumlah barisnya.
	if total := domain.JumlahBaris(fees.BarisFor(domain.TypeRegistration)); total != fees.RegistrationFee {
		t.Errorf("jumlah baris %d tidak sama dengan harga %d", total, fees.RegistrationFee)
	}
}

// Mematikan rincian mengembalikan perhitungan ke harga Dollar per tipe.
func TestMematikanRincian(t *testing.T) {
	awal := biayaLama()
	awal.Rincian.Aktif = true
	awal.RegistrationFeeUSD, awal.RenewalFeeUSD = 920.00, 720.20
	store := &storePalsu{fees: awal}
	mati := awal.Rincian
	mati.Aktif = false

	fees, err := NewService(store).UpdateFees(context.Background(), domain.UpdateFeeSettingsInput{Rincian: &mati})
	if err != nil {
		t.Fatalf("UpdateFees: %v", err)
	}
	// Tanpa training: 920 × 16.000.
	if fees.RegistrationFee != 14_720_000 {
		t.Errorf("pendaftaran = %d, ingin 14.720.000", fees.RegistrationFee)
	}
	if fees.BarisFor(domain.TypeRegistration) != nil {
		t.Error("rincian mati seharusnya tidak menghasilkan baris")
	}
}

func TestRincianSalahDitolak(t *testing.T) {
	store := &storePalsu{fees: biayaLama()}
	_, err := NewService(store).UpdateFees(context.Background(),
		domain.UpdateFeeSettingsInput{Rincian: &domain.RincianBiaya{PPNPersen: 150}})
	if err == nil {
		t.Fatal("PPN 150% seharusnya ditolak")
	}
}
