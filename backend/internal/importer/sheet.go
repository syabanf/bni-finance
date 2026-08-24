package importer

import (
	"fmt"
	"strings"
)

// Pemetaan kolom, dan alasan ia tidak boleh mengandalkan urutan.
//
// Berkas yang benar-benar dikirim orang tidak pernah punya urutan kolom yang
// sama: sebagian menaruh email sebelum telepon, sebagian menambahkan kolom
// catatan di tengah, sebagian memakai judul berbahasa Indonesia. Membaca kolom
// berdasarkan POSISI akan tetap "berhasil" pada semuanya — dengan nomor telepon
// tersimpan sebagai nama perusahaan.
//
// Karena itu kolom dicari lewat JUDULNYA, dan judul yang tidak dikenal
// dilaporkan alih-alih diabaikan.

// Format menentukan cara berkas dibaca.
type Format string

const (
	FormatCSV  Format = "csv"
	FormatXLSX Format = "xlsx"
)

// Baca membaca berkas menjadi baris-baris teks.
//
// Formatnya ditentukan dari ISI berkas, bukan dari nama atau content-type yang
// dikirim klien: berkas .xlsx yang di-rename menjadi .csv adalah kejadian
// sehari-hari, dan menebak dari namanya berarti menolak berkas yang sebenarnya
// sah dengan pesan yang tidak menjelaskan apa pun.
func Baca(data []byte) ([]sheetRow, Format, error) {
	// XLSX selalu diawali tanda tangan zip "PK\x03\x04".
	if len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 3 && data[3] == 4 {
		rows, err := bacaXLSX(data)
		return rows, FormatXLSX, err
	}
	rows, err := bacaCSV(data)
	return rows, FormatCSV, err
}

// Tabel adalah lembar yang judul kolomnya sudah dipetakan.
type Tabel struct {
	Judul []string
	Baris []sheetRow
	kolom map[string]int
}

// normalJudul menyeragamkan judul kolom sebelum dicocokkan.
//
// "Chapter ID", "chapter_id", dan "chapterId" adalah kolom yang sama bagi orang
// yang menyusun berkasnya, jadi harus sama juga di sini. Yang TIDAK dilakukan:
// menebak kolom yang mirip — "chapter" tidak otomatis berarti "chapter_id",
// karena tebakan yang meleset menulis data ke kolom yang salah tanpa ada yang
// tahu.
func normalJudul(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// BuatTabel memetakan judul kolom pada baris pertama.
func BuatTabel(rows []sheetRow) (*Tabel, error) {
	// Baris kosong di atas judul adalah hal biasa pada berkas yang disusun
	// manual — logo, judul laporan, baris pemisah. Dilewati sampai menemukan
	// baris pertama yang benar-benar berisi.
	mulai := -1
	for i, r := range rows {
		for _, sel := range r {
			if strings.TrimSpace(sel) != "" {
				mulai = i
				break
			}
		}
		if mulai >= 0 {
			break
		}
	}
	if mulai < 0 {
		return nil, fmt.Errorf("berkas kosong — tidak ada satu pun baris berisi")
	}

	judul := make([]string, len(rows[mulai]))
	kolom := map[string]int{}
	for i, sel := range rows[mulai] {
		judul[i] = strings.TrimSpace(sel)
		if n := normalJudul(sel); n != "" {
			// Judul ganda: yang PERTAMA menang, dan itu dilaporkan sebagai
			// masalah oleh pemanggil. Diam-diam memakai yang terakhir membuat
			// dua kolom berjudul sama menghasilkan data yang berbeda-beda
			// tergantung urutan.
			if _, ada := kolom[n]; !ada {
				kolom[n] = i
			}
		}
	}
	return &Tabel{Judul: judul, Baris: rows[mulai+1:], kolom: kolom}, nil
}

// Punya melaporkan kolom dengan salah satu nama itu ada.
func (t *Tabel) Punya(nama ...string) bool {
	for _, n := range nama {
		if _, ok := t.kolom[normalJudul(n)]; ok {
			return true
		}
	}
	return false
}

// Sel mengambil nilai kolom pada satu baris, dicari lewat judulnya.
//
// Beberapa nama diterima untuk kolom yang sama, sehingga berkas berjudul
// Indonesia maupun Inggris sama-sama terbaca.
func (t *Tabel) Sel(baris sheetRow, nama ...string) string {
	for _, n := range nama {
		i, ok := t.kolom[normalJudul(n)]
		if !ok || i >= len(baris) {
			continue
		}
		if v := strings.TrimSpace(baris[i]); v != "" {
			return v
		}
	}
	return ""
}

// JudulTakDikenal mengembalikan judul kolom yang tidak dipakai sama sekali.
//
// Dilaporkan, bukan diabaikan diam-diam: kolom yang salah ketik — "emial",
// "chapter id " dengan spasi ekstra yang tidak tersaring — akan tampak sebagai
// kolom tak dikenal di sini, dan itu satu-satunya petunjuk bahwa datanya tidak
// ikut terbaca.
func (t *Tabel) JudulTakDikenal(dikenal []string) []string {
	set := map[string]bool{}
	for _, n := range dikenal {
		set[normalJudul(n)] = true
	}
	var out []string
	for i, j := range t.Judul {
		n := normalJudul(j)
		if n == "" || set[n] {
			continue
		}
		// Hanya laporkan kolom yang benar-benar terpetakan ke indeks ini,
		// supaya judul ganda tidak dilaporkan dua kali.
		if t.kolom[n] == i {
			out = append(out, j)
		}
	}
	return out
}
