package importer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// bacaCSV membaca CSV menjadi bentuk yang sama dengan pembaca XLSX.
//
// Tiga hal yang membuat CSV "dari Excel" sering gagal terbaca, dan ketiganya
// ditangani di sini karena kegagalannya tidak kentara:
//
//	BOM UTF-8      Excel di Windows menaruh EF BB BF di awal berkas, sehingga
//	               judul kolom pertama terbaca "<BOM>id" dan tidak pernah cocok
//	               dengan "id" — importnya menolak seluruh berkas dengan pesan
//	               "kolom id tidak ada", padahal kolomnya jelas terlihat
//	pemisah ;      Excel pada locale Indonesia menulis titik koma, bukan koma;
//	               dibaca sebagai koma, seluruh baris menjadi satu kolom
//	jumlah kolom   baris yang lebih pendek dari judulnya adalah hal biasa pada
//	  tidak sama    berkas sungguhan, dan menolaknya membuat berkas yang sah
//	               ditolak seluruhnya karena satu baris berekor kosong
func bacaCSV(data []byte) ([]sheetRow, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	if !utf8.Valid(data) {
		return nil, fmt.Errorf("berkas bukan UTF-8 — simpan ulang sebagai CSV UTF-8")
	}

	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = tebakPemisah(data)
	// Baris boleh punya jumlah kolom berbeda; pemetaannya dilakukan berdasarkan
	// judul kolom, bukan posisi tetap.
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	var out []sheetRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("baris %d tidak bisa dibaca: %w", len(out)+1, err)
		}
		out = append(out, sheetRow(rec))
	}
	return out, nil
}

// tebakPemisah memilih antara koma dan titik koma.
//
// Dihitung dari BARIS PERTAMA saja — itu judul kolomnya, dan pemisah yang benar
// adalah yang menghasilkan lebih banyak potongan di sana. Menghitung seluruh
// berkas akan tertipu oleh titik koma yang muncul di dalam alamat atau catatan.
func tebakPemisah(data []byte) rune {
	baris := data
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		baris = data[:i]
	}
	if strings.Count(string(baris), ";") > strings.Count(string(baris), ",") {
		return ';'
	}
	return ','
}
