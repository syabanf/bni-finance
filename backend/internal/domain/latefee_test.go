package domain_test

import (
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
)

func tanggal(s string) domain.Date {
	d, err := domain.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func invoicePada(status domain.InvoiceStatus, jatuhTempo string) domain.Invoice {
	return domain.Invoice{Status: status, DueDate: tanggal(jatuhTempo)}
}

var aturan = domain.LateFeeRule{Aktif: true, PerHari: 25_000, MaksHari: 90}

func TestDendaTumbuhPerHari(t *testing.T) {
	kasus := []struct {
		nama     string
		kini     string
		mauHari  int
		mauDenda int64
		mauBatas bool
	}{
		{"belum jatuh tempo", "2026-08-20", 0, 0, false},
		{"tepat hari jatuh tempo", "2026-08-24", 0, 0, false},
		{"telat sehari", "2026-08-25", 1, 25_000, false},
		{"telat sepuluh hari", "2026-09-03", 10, 250_000, false},
		{"tepat di batas", "2026-11-22", 90, 2_250_000, false},
		{"lewat batas — berhenti tumbuh", "2027-08-24", 90, 2_250_000, true},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			now, _ := time.Parse("2006-01-02", k.kini)
			got := aturan.Hitung(invoicePada(domain.StatusSent, "2026-08-24"), now)
			if got.HariTelat != k.mauHari || got.Nominal != k.mauDenda || got.Tercapaikan != k.mauBatas {
				t.Errorf("dapat %+v, mau hari=%d denda=%d batas=%v",
					got, k.mauHari, k.mauDenda, k.mauBatas)
			}
		})
	}
}

// Hanya tagihan yang MASIH BERDIRI dan belum dibayar yang menumbuhkan denda.
//
// Ini yang paling mudah salah, dan salahnya tidak terlihat sampai ada yang
// membaca laporan: menagih denda atas invoice yang sudah lunas berarti menuntut
// uang yang sudah diterima, dan atas invoice yang dibatalkan berarti menuntut
// uang atas tagihan yang sudah ditarik kembali.
func TestHanyaTagihanBerdiriYangKenaDenda(t *testing.T) {
	now, _ := time.Parse("2006-01-02", "2026-09-24") // 31 hari lewat
	kena := map[domain.InvoiceStatus]bool{
		domain.StatusSent:       true,
		domain.StatusOverdue:    true,
		domain.StatusDraft:      false,
		domain.StatusPaid:       false,
		domain.StatusCancelled:  false,
		domain.StatusTerminated: false,
	}
	for status, mauKena := range kena {
		got := aturan.Hitung(invoicePada(status, "2026-08-24"), now)
		if (got.Nominal > 0) != mauKena {
			t.Errorf("status %s: denda=%d, mau kena=%v", status, got.Nominal, mauKena)
		}
	}
}

// Sakelar mati harus benar-benar mematikan, termasuk saat nominalnya terisi.
func TestSakelarMatiMenolMutlak(t *testing.T) {
	now, _ := time.Parse("2006-01-02", "2026-12-24")
	mati := domain.LateFeeRule{Aktif: false, PerHari: 25_000, MaksHari: 90}
	if got := mati.Hitung(invoicePada(domain.StatusOverdue, "2026-08-24"), now); got.Nominal != 0 {
		t.Errorf("aturan mati tetap menghitung denda %d", got.Nominal)
	}
	// Nominal nol juga berarti tidak ada denda, meski sakelarnya menyala —
	// kalau tidak, UI menampilkan "telat 120 hari, denda Rp 0" yang membingungkan.
	nol := domain.LateFeeRule{Aktif: true, PerHari: 0, MaksHari: 90}
	if got := nol.Hitung(invoicePada(domain.StatusOverdue, "2026-08-24"), now); got.HariTelat != 0 {
		t.Errorf("nominal nol tetap melaporkan %d hari telat", got.HariTelat)
	}
}

// MaksHari nol berarti TANPA batas, bukan denda nol.
func TestMaksHariNolBerartiTanpaBatas(t *testing.T) {
	now, _ := time.Parse("2006-01-02", "2027-08-24") // 365 hari
	tanpaBatas := domain.LateFeeRule{Aktif: true, PerHari: 1_000, MaksHari: 0}
	got := tanpaBatas.Hitung(invoicePada(domain.StatusSent, "2026-08-24"), now)
	if got.HariTelat != 365 || got.Nominal != 365_000 || got.Tercapaikan {
		t.Errorf("dapat %+v, mau 365 hari / 365.000 / tanpa batas", got)
	}
}
