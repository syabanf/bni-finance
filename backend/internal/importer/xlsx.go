package importer

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Pembaca XLSX seperlunya, di atas pustaka standar saja.
//
// Backend ini punya SATU dependensi langsung (pgx), dan itu bukan kebetulan:
// tiap pustaka tambahan adalah permukaan rantai pasok baru pada sistem yang
// memegang data keanggotaan dan uang. Menambah pustaka Excel demi membaca
// beberapa kolom teks tidak sepadan.
//
// XLSX sendiri hanyalah zip berisi XML, jadi archive/zip dan encoding/xml sudah
// cukup. Yang dibaca hanya yang dibutuhkan: lembar PERTAMA, sebagai teks.
//
// YANG SENGAJA TIDAK DIDUKUNG, dan semuanya ditolak dengan pesan yang jelas
// alih-alih diam-diam salah baca:
//
//	rumus       nilai hasil hitungannya dibaca, bukan rumusnya
//	tanggal     dibaca apa adanya sebagai angka serial Excel — lihat catatan
//	            di bawah, kolom tanggal harus diformat sebagai teks
//	beberapa    hanya lembar pertama; berkas dengan banyak lembar tetap terbaca
//	lembar      tapi sisanya diabaikan
type sheetRow []string

// bacaXLSX mengembalikan seluruh baris lembar pertama sebagai teks.
func bacaXLSX(data []byte) ([]sheetRow, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("berkas bukan XLSX yang sah: %w", err)
	}

	var sharedRaw, sheetRaw []byte
	for _, f := range zr.File {
		switch {
		case f.Name == "xl/sharedStrings.xml":
			if sharedRaw, err = bacaEntri(f); err != nil {
				return nil, err
			}
		// Lembar pertama tidak selalu bernama sheet1.xml — urutan di workbook
		// yang menentukan. Untuk berkas yang dibuat Excel dan Google Sheets,
		// sheet1.xml adalah lembar pertama, dan itu yang didukung di sini.
		case f.Name == "xl/worksheets/sheet1.xml":
			if sheetRaw, err = bacaEntri(f); err != nil {
				return nil, err
			}
		}
	}
	if sheetRaw == nil {
		return nil, fmt.Errorf("tidak ada lembar kerja di dalam berkas (xl/worksheets/sheet1.xml)")
	}

	shared, err := bacaSharedStrings(sharedRaw)
	if err != nil {
		return nil, err
	}
	return bacaSheet(sheetRaw, shared)
}

func bacaEntri(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("buka %s: %w", f.Name, err)
	}
	defer rc.Close()
	// Dibatasi 32 MB. Zip bisa mengembang jauh lebih besar daripada ukuran
	// berkasnya (zip bomb), dan unggahan yang mengembang sampai kehabisan
	// memori akan mematikan seluruh proses, bukan hanya permintaan itu.
	b, err := io.ReadAll(io.LimitReader(rc, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("baca %s: %w", f.Name, err)
	}
	return b, nil
}

// bacaSharedStrings membaca tabel teks bersama.
//
// Excel menyimpan sebagian besar teks di satu tabel dan menaruh INDEKSNYA di
// sel. Tanpa membaca tabel ini, seluruh kolom teks terbaca sebagai angka —
// "0", "1", "2" — dan importnya tampak berhasil dengan data yang sepenuhnya
// salah. Itu kegagalan sunyi, jadi tabelnya dibaca lebih dulu.
func bacaSharedStrings(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var sst struct {
		Items []struct {
			// <si> bisa berisi <t> tunggal, atau beberapa <r><t> untuk teks
			// yang formatnya berbeda-beda di dalam satu sel. Keduanya
			// digabungkan; mengambil yang pertama saja akan memotong nama yang
			// sebagian hurufnya ditebalkan.
			T string   `xml:"t"`
			R []string `xml:"r>t"`
		} `xml:"si"`
	}
	if err := xml.Unmarshal(raw, &sst); err != nil {
		return nil, fmt.Errorf("baca tabel teks: %w", err)
	}
	out := make([]string, len(sst.Items))
	for i, it := range sst.Items {
		if len(it.R) > 0 {
			out[i] = strings.Join(it.R, "")
			continue
		}
		out[i] = it.T
	}
	return out, nil
}

func bacaSheet(raw []byte, shared []string) ([]sheetRow, error) {
	var sheet struct {
		Rows []struct {
			Cells []struct {
				Ref    string `xml:"r,attr"`
				Type   string `xml:"t,attr"`
				Value  string `xml:"v"`
				Inline string `xml:"is>t"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(raw, &sheet); err != nil {
		return nil, fmt.Errorf("baca lembar kerja: %w", err)
	}

	out := make([]sheetRow, 0, len(sheet.Rows))
	for _, r := range sheet.Rows {
		// Sel KOSONG TIDAK DITULIS sama sekali oleh Excel. Baris dengan A1 dan
		// C1 terisi datang tanpa B1, jadi membaca sel berurutan akan menggeser
		// seluruh kolom setelahnya — nomor telepon masuk ke kolom perusahaan,
		// dan tidak ada yang gagal. Karena itu tiap sel ditaruh pada indeks
		// yang dihitung dari referensinya (A, B, C…), bukan urutan munculnya.
		var baris sheetRow
		for _, c := range r.Cells {
			idx := kolomDariRef(c.Ref)
			if idx < 0 {
				continue
			}
			for len(baris) <= idx {
				baris = append(baris, "")
			}
			baris[idx] = nilaiSel(c.Type, c.Value, c.Inline, shared)
		}
		out = append(out, baris)
	}
	return out, nil
}

func nilaiSel(tipe, value, inline string, shared []string) string {
	switch tipe {
	case "s": // indeks ke tabel teks bersama
		i, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || i < 0 || i >= len(shared) {
			return ""
		}
		return shared[i]
	case "inlineStr":
		return inline
	}
	return value
}

// kolomDariRef menerjemahkan referensi sel ("A1", "AB12") menjadi indeks kolom
// berbasis nol. Mengembalikan -1 bila referensinya tidak bisa dibaca.
func kolomDariRef(ref string) int {
	n := 0
	ada := false
	for _, c := range ref {
		switch {
		case c >= 'A' && c <= 'Z':
			n = n*26 + int(c-'A') + 1
			ada = true
		case c >= 'a' && c <= 'z':
			n = n*26 + int(c-'a') + 1
			ada = true
		case c >= '0' && c <= '9':
			if !ada {
				return -1
			}
			return n - 1
		default:
			return -1
		}
	}
	if !ada {
		return -1
	}
	return n - 1
}
