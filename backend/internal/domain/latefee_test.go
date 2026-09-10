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

var aturan = domain.LateFeeRule{Aktif: true, Jenis: domain.DendaRupiah, Satuan: domain.SatuanHari, PerSatuan: 25_000, MaksHari: 90}

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
	mati := domain.LateFeeRule{Aktif: false, PerSatuan: 25_000, MaksHari: 90}
	if got := mati.Hitung(invoicePada(domain.StatusOverdue, "2026-08-24"), now); got.Nominal != 0 {
		t.Errorf("aturan mati tetap menghitung denda %d", got.Nominal)
	}
	// Nominal nol juga berarti tidak ada denda, meski sakelarnya menyala —
	// kalau tidak, UI menampilkan "telat 120 hari, denda Rp 0" yang membingungkan.
	nol := domain.LateFeeRule{Aktif: true, Jenis: domain.DendaRupiah, Satuan: domain.SatuanHari, PerSatuan: 0, MaksHari: 90}
	if got := nol.Hitung(invoicePada(domain.StatusOverdue, "2026-08-24"), now); got.HariTelat != 0 {
		t.Errorf("nominal nol tetap melaporkan %d hari telat", got.HariTelat)
	}
}

// MaksHari nol berarti TANPA batas, bukan denda nol.
func TestMaksHariNolBerartiTanpaBatas(t *testing.T) {
	now, _ := time.Parse("2006-01-02", "2027-08-24") // 365 hari
	tanpaBatas := domain.LateFeeRule{Aktif: true, Jenis: domain.DendaRupiah, Satuan: domain.SatuanHari, PerSatuan: 1_000, MaksHari: 0}
	got := tanpaBatas.Hitung(invoicePada(domain.StatusSent, "2026-08-24"), now)
	if got.HariTelat != 365 || got.Nominal != 365_000 || got.Tercapaikan {
		t.Errorf("dapat %+v, mau 365 hari / 365.000 / tanpa batas", got)
	}
}

// --- persentase & satuan waktu -----------------------------------------------

func rule(j domain.JenisDenda, sat domain.SatuanDenda, rupiah int64, persen float64) domain.LateFeeRule {
	return domain.LateFeeRule{
		Aktif: true, Jenis: j, Satuan: sat,
		PerSatuan: rupiah, PersenPerSatuan: persen,
	}
}

// acuan adalah "hari ini" yang tetap, supaya hasilnya tidak berubah tiap hari.
var acuan = func() time.Time {
	t, _ := time.Parse("2006-01-02", "2026-12-01")
	return t
}()

func invTelat(nominal int64, hariTelat int) domain.Invoice {
	return domain.Invoice{
		Status:  domain.StatusOverdue,
		Amount:  nominal,
		DueDate: tanggal(acuan.AddDate(0, 0, -hariTelat).Format("2006-01-02")),
	}
}

// SATUAN YANG SUDAH GENAP, bukan pecahannya.
//
// Denda "per minggu" pada hari keempat adalah NOL. Membebankan denda satu
// minggu penuh di hari pertama bukan hal yang diharapkan siapa pun dari kata
// "per minggu", dan pecahan minggu adalah angka yang tidak bisa dijelaskan di
// kuitansi.
func TestSatuanDihitungYangSudahGenap(t *testing.T) {
	r := rule(domain.DendaRupiah, domain.SatuanMinggu, 50_000, 0)

	for _, k := range []struct {
		hari int
		mau  int64
	}{
		{1, 0}, {6, 0}, // belum genap seminggu
		{7, 50_000}, {13, 50_000}, // satu minggu
		{14, 100_000}, // dua minggu
	} {
		got := r.Hitung(invTelat(1_000_000, k.hari), acuan)
		if got.Nominal != k.mau {
			t.Errorf("telat %d hari → %d, seharusnya %d", k.hari, got.Nominal, k.mau)
		}
		// Hari telatnya tetap dilaporkan apa adanya walau dendanya nol —
		// menyembunyikannya membuat orang mengira sistem tidak menghitung.
		if got.HariTelat != k.hari {
			t.Errorf("telat %d hari dilaporkan %d", k.hari, got.HariTelat)
		}
	}
}

// Bulan dihitung 30 hari, bukan panjang bulan kalender.
//
// Denda yang besarnya bergantung pada apakah keterlambatannya jatuh di Februari
// atau Juli tidak bisa dijelaskan kepada member.
func TestBulanTigaPuluhHari(t *testing.T) {
	r := rule(domain.DendaRupiah, domain.SatuanBulan, 100_000, 0)
	if got := r.Hitung(invTelat(1_000_000, 29), acuan).Nominal; got != 0 {
		t.Errorf("telat 29 hari → %d, seharusnya 0", got)
	}
	if got := r.Hitung(invTelat(1_000_000, 30), acuan).Nominal; got != 100_000 {
		t.Errorf("telat 30 hari → %d, seharusnya 100000", got)
	}
	if got := r.Hitung(invTelat(1_000_000, 61), acuan).Nominal; got != 200_000 {
		t.Errorf("telat 61 hari → %d, seharusnya 200000", got)
	}
}

// Persentase dihitung dari NOMINAL INVOICE, jadi tagihan besar kena lebih besar.
func TestDendaPersenDariNominalInvoice(t *testing.T) {
	r := rule(domain.DendaPersen, domain.SatuanHari, 0, 1) // 1% per hari

	if got := r.Hitung(invTelat(1_000_000, 3), acuan).Nominal; got != 30_000 {
		t.Errorf("1%%/hari × 3 hari atas Rp1jt → %d, seharusnya 30000", got)
	}
	// Nominal sepuluh kali lipat menghasilkan denda sepuluh kali lipat.
	if got := r.Hitung(invTelat(10_000_000, 3), acuan).Nominal; got != 300_000 {
		t.Errorf("1%%/hari × 3 hari atas Rp10jt → %d, seharusnya 300000", got)
	}
}

// Pecahan rupiah DIBULATKAN KE BAWAH.
//
// Denda berkoma tidak bisa ditransfer, dan membulatkan ke atas berarti menagih
// lebih dari aturannya sendiri.
func TestPersenDibulatkanKeBawah(t *testing.T) {
	r := rule(domain.DendaPersen, domain.SatuanHari, 0, 0.5) // 0,5% per hari
	// 0,5% dari 12.345 = 61,725 → 61
	if got := r.Hitung(invTelat(12_345, 1), acuan).Nominal; got != 61 {
		t.Errorf("0,5%% dari 12345 → %d, seharusnya 61 (dibulatkan ke bawah)", got)
	}
}

// Jenis persen dengan persentase nol berarti ATURANNYA MATI, bukan sekadar
// dendanya kebetulan nol — dan bedanya terlihat di layar.
//
// "Aturan mati" mengembalikan hasil kosong: tidak ada denda, tidak ada hari
// telat, tidak ada yang ditampilkan. "Denda nol pada aturan yang hidup" tetap
// melaporkan hari telatnya, seperti pada minggu yang belum genap — di sana
// angka hari itu justru penting, karena ia memberi tahu dendanya akan mulai
// tumbuh sebentar lagi.
//
// Membedakan keduanya berarti orang tidak melihat "telat 5 hari, denda Rp0"
// pada pengaturan yang sebenarnya belum diisi sama sekali.
//
// CATATAN: asersi pertama saya hanya memeriksa Nominal, dan itu TIDAK menjaga
// apa pun — perhitungannya juga bercabang pada Jenis, jadi persen nol selalu
// menghasilkan nominal nol dengan atau tanpa penjaganya. Sabotase yang
// menghapus penjaga itu tetap hijau. HariTelat yang membedakannya.
func TestPersenNolBerartiAturanMati(t *testing.T) {
	r := domain.LateFeeRule{
		Aktif: true, Jenis: domain.DendaPersen, Satuan: domain.SatuanHari,
		PerSatuan: 50_000, PersenPerSatuan: 0,
	}
	got := r.Hitung(invTelat(1_000_000, 5), acuan)
	if got != (domain.LateFee{}) {
		t.Errorf("persen nol → %+v, seharusnya kosong sepenuhnya "+
			"(aturan yang belum diisi bukan aturan yang hidup dengan denda nol)", got)
	}
}
