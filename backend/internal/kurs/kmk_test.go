package kurs

import (
	"strings"
	"testing"
	"time"
)

// Potongan halaman Kurs Pajak, disusun dari teks yang tampil pada 10 Oktober
// 2026 (KMK 48/MK/EF.2/2026). Markupnya tebakan; yang diuji adalah teks yang
// dibaca orang, karena itu satu-satunya yang diandalkan pengurainya.
const halamanContoh = `<html><body>
<div class="kurs-header">
  <p><b>KMK Nomor 48/MK/EF.2/2026</b></p>
  <p><i>Tanggal berlaku: 07 Oktober 2026 - 13 Oktober 2026</i></p>
</div>
<table class="table">
  <thead><tr><th>No</th><th>Mata Uang</th><th>Nilai</th><th>Perubahan</th></tr></thead>
  <tbody>
    <tr><td>1</td><td><img alt="transparent.gif"> Dolar Amerika Serikat (USD) <span>USD</span></td>
        <td>17.935,00</td><td><img alt="up"> 72,00</td></tr>
    <tr><td>2</td><td>Dolar Australia (AUD) <span>AUD</span></td><td>11.812,45</td><td>-30,12</td></tr>
    <tr><td>9</td><td>Yen Jepang (JPY) <span>JPY</span></td><td>11.903,66</td><td>12,00</td></tr>
  </tbody>
</table>
<p>Catatan: untuk JPY, nilai rupiah ditampilkan per 100 yen.</p>
</body></html>`

func TestBacaHalamanKursPajak(t *testing.T) {
	k, err := Baca([]byte(halamanContoh))
	if err != nil {
		t.Fatalf("baca: %v", err)
	}
	if k.USD != 17_935 {
		t.Errorf("USD = %d, mau 17935", k.USD)
	}
	if k.Nomor != "48/MK/EF.2/2026" {
		t.Errorf("nomor = %q", k.Nomor)
	}
	if !k.BerlakuDari.Equal(time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)) ||
		!k.BerlakuSampai.Equal(time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("berlaku %s–%s", k.BerlakuDari, k.BerlakuSampai)
	}
}

// Pecahan sen dibulatkan ke Rupiah terdekat; harga di invoice tidak bersen.
func TestBacaMembulatkanSen(t *testing.T) {
	for masuk, mau := range map[string]int64{"17.935,49": 17_935, "17.935,50": 17_936} {
		k, err := Baca([]byte(strings.Replace(halamanContoh, "17.935,00", masuk, 1)))
		if err != nil || k.USD != mau {
			t.Errorf("%s: dapat %v %v, mau %d", masuk, k, err, mau)
		}
	}
}

// Halaman yang berganti bentuk harus GAGAL, bukan menghasilkan kurs ngawur.
// Kurs salah yang diterapkan diam-diam mengubah harga setiap invoice baru.
func TestBacaMenolakHalamanYangTidakDikenali(t *testing.T) {
	kasus := map[string]string{
		"tanpa baris USD":      strings.Replace(halamanContoh, "(USD)", "(XXX)", 1),
		"tanpa nomor KMK":      strings.Replace(halamanContoh, "KMK Nomor", "Keputusan", 1),
		"angka tak masuk akal": strings.Replace(halamanContoh, "17.935,00", "179.350,00", 1),
		"halaman kosong":       "<html></html>",
	}
	for nama, h := range kasus {
		if k, err := Baca([]byte(h)); err == nil {
			t.Errorf("%s: harus gagal, dapat %+v", nama, k)
		}
	}
}
