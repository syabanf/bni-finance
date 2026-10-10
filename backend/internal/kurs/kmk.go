// Package kurs mengambil kurs pajak USD yang ditetapkan lewat Keputusan
// Menteri Keuangan (KMK) dan menerapkannya ke harga keanggotaan.
//
// BNI menetapkan harga dalam Dollar; Rupiah yang tercetak di invoice adalah
// terjemahannya. Kurs KMK dipakai karena itulah kurs resmi untuk keperluan
// pajak, sehingga angka di invoice sama dengan angka yang dipakai saat
// menyusun faktur pajaknya.
package kurs

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SumberURL adalah halaman Kurs Pajak Badan Kebijakan Fiskal.
//
// Kemenkeu juga punya "Layanan API Nilai Kurs", tapi aksesnya lewat
// pendaftaran. Halaman publik ini memuat angka yang sama, berikut nomor KMK
// dan masa berlakunya, tanpa kredensial apa pun.
const SumberURL = "https://fiskal.kemenkeu.go.id/informasi-publik/kurs-pajak"

// KMK adalah kurs pajak USD untuk satu periode berlaku (Rabu sampai Selasa).
type KMK struct {
	// USD adalah Rupiah per satu dolar, dibulatkan ke Rupiah terdekat.
	USD           int64     `json:"usd"`
	Nomor         string    `json:"nomor"`
	BerlakuDari   time.Time `json:"berlakuDari"`
	BerlakuSampai time.Time `json:"berlakuSampai"`
}

// Batas kewajaran kurs USD. Angka di luar rentang ini hampir pasti hasil
// membaca sel yang salah (nomor urut, kolom perubahan, JPY per 100 yen), dan
// menerapkannya akan mengubah harga keanggotaan dengan diam-diam.
const (
	usdMin  = 5_000
	usdMaks = 50_000
)

var (
	tag       = regexp.MustCompile(`(?s)<[^>]*>`)
	spasi     = regexp.MustCompile(`\s+`)
	polaUSD   = regexp.MustCompile(`\(USD\)\D{0,60}?(\d{1,3}(?:\.\d{3})+|\d{4,6}),(\d{2})`)
	polaNo    = regexp.MustCompile(`KMK\s+Nomor\s+([0-9A-Za-z./-]+)`)
	polaTgl   = regexp.MustCompile(`(?i)Tanggal\s+berlaku\s*:?\s*(\d{1,2})\s+([A-Za-z]+)\s+(\d{4})\s*[-–]\s*(\d{1,2})\s+([A-Za-z]+)\s+(\d{4})`)
	namaBulan = map[string]time.Month{
		"januari": time.January, "februari": time.February, "maret": time.March,
		"april": time.April, "mei": time.May, "juni": time.June, "juli": time.July,
		"agustus": time.August, "september": time.September, "oktober": time.October,
		"november": time.November, "desember": time.December,
	}
)

// Baca mengurai halaman Kurs Pajak menjadi KMK.
//
// Halaman diratakan jadi teks lebih dulu, lalu dicari polanya. Markup
// halamannya bukan kontrak dan bisa berganti kapan saja; teks yang dibaca
// orang ("Dolar Amerika Serikat (USD) USD 17.935,00", "KMK Nomor ...",
// "Tanggal berlaku: ...") jauh lebih jarang berubah.
func Baca(halaman []byte) (*KMK, error) {
	teks := spasi.ReplaceAllString(html.UnescapeString(tag.ReplaceAllString(string(halaman), " ")), " ")

	m := polaUSD.FindStringSubmatch(teks)
	if m == nil {
		return nil, errors.New("baris USD tidak ditemukan di halaman kurs pajak")
	}
	rupiah, err := strconv.ParseInt(strings.ReplaceAll(m[1], ".", ""), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("angka kurs USD %q tidak terbaca: %w", m[1], err)
	}
	if sen, _ := strconv.Atoi(m[2]); sen >= 50 {
		rupiah++
	}
	if rupiah < usdMin || rupiah > usdMaks {
		return nil, fmt.Errorf("kurs USD %d di luar rentang wajar %d–%d", rupiah, usdMin, usdMaks)
	}

	no := polaNo.FindStringSubmatch(teks)
	if no == nil {
		return nil, errors.New("nomor KMK tidak ditemukan di halaman kurs pajak")
	}

	k := &KMK{USD: rupiah, Nomor: no[1]}
	if t := polaTgl.FindStringSubmatch(teks); t != nil {
		k.BerlakuDari, _ = tanggal(t[1], t[2], t[3])
		k.BerlakuSampai, _ = tanggal(t[4], t[5], t[6])
	}
	return k, nil
}

func tanggal(hari, bulan, tahun string) (time.Time, error) {
	b, ok := namaBulan[strings.ToLower(bulan)]
	if !ok {
		return time.Time{}, fmt.Errorf("bulan %q tidak dikenal", bulan)
	}
	h, err1 := strconv.Atoi(hari)
	y, err2 := strconv.Atoi(tahun)
	if err := errors.Join(err1, err2); err != nil {
		return time.Time{}, err
	}
	return time.Date(y, b, h, 0, 0, 0, 0, time.UTC), nil
}

// Ambil mengunduh halaman di url lalu menguraikannya.
func Ambil(ctx context.Context, client *http.Client, url string) (*KMK, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BNI-Finance-Hub/1.0 (kurs pajak harian)")
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unduh halaman kurs pajak: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("halaman kurs pajak menjawab %d", res.StatusCode)
	}
	halaman, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("baca halaman kurs pajak: %w", err)
	}
	return Baca(halaman)
}
