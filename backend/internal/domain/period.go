package domain

import (
	"fmt"
	"strings"
	"time"
)

// RenewalPeriod mencerminkan satu baris renewal_periods: rentang tanggal yang
// keanggotaannya ditanyakan ke MC dalam satu putaran konfirmasi.
type RenewalPeriod struct {
	// ID adalah teks yang tersimpan di renewal_requests.period.
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartDate Date   `json:"startDate"`
	EndDate   Date   `json:"endDate"`
	// RequestCount menghitung permintaan konfirmasi di periode ini, dalam
	// lingkup chapter pemanggil. Periode yang sudah dipakai tidak bisa dihapus.
	RequestCount int `json:"requestCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SaveRenewalPeriodInput dipakai untuk membuat dan mengubah periode. Saat
// membuat, ketiganya wajib. Saat mengubah, yang kosong dibiarkan.
type SaveRenewalPeriodInput struct {
	Name      *string `json:"name"`
	StartDate *Date   `json:"startDate"`
	EndDate   *Date   `json:"endDate"`
}

// MaxPeriodName membatasi nama periode. Nama ini tampil di dropdown dan di
// subjek email ke MC.
const MaxPeriodName = 60

// Terapkan menimpa p dengan isi in, lalu memeriksa hasilnya.
func (in SaveRenewalPeriodInput) Terapkan(p *RenewalPeriod) error {
	if in.Name != nil {
		p.Name = strings.TrimSpace(*in.Name)
	}
	if in.StartDate != nil {
		p.StartDate = *in.StartDate
	}
	if in.EndDate != nil {
		p.EndDate = *in.EndDate
	}
	switch {
	case p.Name == "":
		return fmt.Errorf("name wajib diisi")
	case len([]rune(p.Name)) > MaxPeriodName:
		return fmt.Errorf("name maksimal %d karakter", MaxPeriodName)
	case p.StartDate.IsZero() || p.EndDate.IsZero():
		return fmt.Errorf("startDate dan endDate wajib diisi")
	case p.EndDate.Before(p.StartDate.Time):
		return fmt.Errorf("endDate tidak boleh sebelum startDate")
	}
	return nil
}

// KodePeriode mengusulkan id untuk periode baru, dari yang paling enak dibaca.
//
// Semester dan tahun penuh memakai bentuk yang sudah dipakai baris lama
// ("2029-S2", "2029"). Rentang lain memakai kedua tanggalnya. Bentuk tanggal
// selalu ikut diusulkan sebagai cadangan, untuk keadaan langka ketika kode
// semesternya sudah dipakai periode yang tanggalnya pernah diubah.
func KodePeriode(mulai, selesai Date) []string {
	tanggal := mulai.Format("20060102") + "-" + selesai.Format("20060102")
	if mulai.Year() != selesai.Year() {
		return []string{tanggal}
	}
	tahun := mulai.Format("2006")
	awal, akhir := mulai.Format("0102"), selesai.Format("0102")
	switch {
	case awal == "0101" && akhir == "0630":
		return []string{tahun + "-S1", tanggal}
	case awal == "0701" && akhir == "1231":
		return []string{tahun + "-S2", tanggal}
	case awal == "0101" && akhir == "1231":
		return []string{tahun, tanggal}
	}
	return []string{tanggal}
}
