package importer

import (
	"os"
	"path/filepath"
	"testing"
)

// Diuji terhadap berkas XLSX SUNGGUHAN yang dibuat Excel/openpyxl, bukan
// terhadap XML yang saya karang sendiri.
//
// Bedanya menentukan: XML karangan cenderung mencerminkan cara saya MENGIRA
// Excel menulis berkas, dan tes seperti itu hanya membuktikan pembacanya cocok
// dengan tebakan saya. Berkas nyata memuat hal-hal yang tidak akan saya tulis
// sendiri — tabel teks bersama, sel kosong yang benar-benar dihilangkan, teks
// berformat campuran yang terpecah menjadi beberapa potongan.
//
// Berkasnya dibuat ulang dengan scripts di testdata; lihat commit yang
// menambahkannya.

func muat(t *testing.T, nama string) []sheetRow {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", nama))
	if err != nil {
		t.Fatalf("baca %s: %v", nama, err)
	}
	rows, err := bacaXLSX(b)
	if err != nil {
		t.Fatalf("parse %s: %v", nama, err)
	}
	return rows
}

func TestBacaXLSXNormal(t *testing.T) {
	rows := muat(t, "member-normal.xlsx")
	if len(rows) != 3 {
		t.Fatalf("jumlah baris = %d, mau 3", len(rows))
	}
	mau := []string{"id", "chapter_id", "name", "email", "phone", "company"}
	for i, m := range mau {
		if rows[0][i] != m {
			t.Errorf("judul kolom %d = %q, mau %q", i, rows[0][i], m)
		}
	}
	// Teks harus benar-benar terbaca sebagai teks. Kalau tabel teks bersama
	// tidak dibaca, seluruh kolom ini akan berisi "0", "1", "2" — angka indeks —
	// dan importnya tampak berhasil dengan data yang sepenuhnya salah.
	if rows[1][2] != "Andi Pratama" {
		t.Errorf("nama = %q, mau \"Andi Pratama\" — tabel teks bersama tidak terbaca?", rows[1][2])
	}
	if rows[2][1] != "ch-nusantara" {
		t.Errorf("chapter = %q, mau \"ch-nusantara\"", rows[2][1])
	}
}

// Sel kosong TIDAK DITULIS Excel sama sekali.
//
// Baris dengan A dan C terisi datang tanpa B. Pembaca yang menyusun sel
// berurutan akan menggeser seluruh kolom setelahnya — nama masuk ke kolom
// chapter, email masuk ke kolom nama — dan tidak ada yang gagal. Data yang
// tergeser satu kolom adalah kegagalan paling sunyi yang bisa dihasilkan
// sebuah importer.
func TestSelKosongTidakMenggeserKolom(t *testing.T) {
	rows := muat(t, "member-sel-kosong.xlsx")
	if len(rows) != 3 {
		t.Fatalf("jumlah baris = %d, mau 3", len(rows))
	}

	// Baris 2: B kosong (chapter), C dan D terisi.
	if rows[1][0] != "mem-201" {
		t.Errorf("id = %q, mau mem-201", rows[1][0])
	}
	if rows[1][1] != "" {
		t.Errorf("chapter = %q, mau kosong — kolom tergeser", rows[1][1])
	}
	if rows[1][2] != "Tanpa Chapter" {
		t.Errorf("nama = %q, mau \"Tanpa Chapter\" — kolom tergeser", rows[1][2])
	}
	if rows[1][3] != "a@contoh.id" {
		t.Errorf("email = %q, mau a@contoh.id — kolom tergeser", rows[1][3])
	}

	// Baris 3: D kosong (email) di ujung — barisnya jadi lebih pendek, dan itu
	// sah. Yang penting kolom sebelumnya tidak ikut bergeser.
	if rows[2][1] != "ch-garuda" || rows[2][2] != "Tanpa Email" {
		t.Errorf("baris 3 = %q", rows[2])
	}
}

// Teks berformat campuran dipecah Excel menjadi beberapa potongan <r><t>.
// Mengambil potongan pertama saja akan memotong nama di tengah.
func TestTeksBerformatCampuranUtuh(t *testing.T) {
	rows := muat(t, "member-richtext.xlsx")
	const mau = "Nama Panjang Sekali Yang Sebagian Ditebalkan"
	if len(rows) < 2 || rows[1][1] != mau {
		t.Errorf("nama = %q, mau %q", rows[1][1], mau)
	}
}

func TestBerkasBukanXLSXDitolakJelas(t *testing.T) {
	_, err := bacaXLSX([]byte("id,name\nmem-1,Andi\n"))
	if err == nil {
		t.Fatal("CSV diterima sebagai XLSX")
	}
	if got := err.Error(); got == "" {
		t.Error("pesan galatnya kosong")
	}
}

func TestKolomDariRef(t *testing.T) {
	kasus := map[string]int{
		"A1": 0, "B1": 1, "Z1": 25,
		"AA1": 26, "AB12": 27, "BA3": 52,
		"a1": 0,
		"1":  -1, "": -1, "?1": -1,
	}
	for ref, mau := range kasus {
		if got := kolomDariRef(ref); got != mau {
			t.Errorf("kolomDariRef(%q) = %d, mau %d", ref, got, mau)
		}
	}
}

// Bentuk sharedStrings — yang dipakai Excel dan Google Sheets sungguhan.
//
// Fixture di atas dibuat openpyxl, dan openpyxl menulis inlineStr. Akibatnya
// jalur tabel teks bersama TIDAK PERNAH dijalankan oleh tes mana pun:
// menyabotasenya sampai mengembalikan indeks mentah tetap menghasilkan seluruh
// tes hijau.
//
// Itu berbahaya justru karena arah kesalahannya: hampir setiap berkas yang
// benar-benar diunggah orang datang dari Excel, jadi jalur yang tidak teruji
// itulah yang paling sering dilewati. Kalau rusak, seluruh kolom teks terbaca
// sebagai "0", "1", "2" — dan importnya tampak berhasil.
func TestBacaXLSXBentukSharedStrings(t *testing.T) {
	rows := muat(t, "member-sharedstrings.xlsx")
	if len(rows) != 4 {
		t.Fatalf("jumlah baris = %d, mau 4", len(rows))
	}
	if rows[0][2] != "name" {
		t.Errorf("judul kolom = %q, mau \"name\" — tabel teks bersama tidak terbaca", rows[0][2])
	}
	if rows[1][2] != "Citra Melati" {
		t.Errorf("nama = %q, mau \"Citra Melati\"", rows[1][2])
	}
	// Kolom kosong di tengah, pada bentuk sharedStrings.
	if rows[2][1] != "" || rows[2][2] != "Dian Kusuma" {
		t.Errorf("baris 3 tergeser: %q", rows[2])
	}
	// Teks yang dipakai ulang harus terbaca sama di kedua barisnya — inilah
	// gunanya tabel bersama, dan indeks yang salah baca akan terlihat di sini.
	if rows[1][1] != "ch-garuda" || rows[3][1] != "ch-garuda" {
		t.Errorf("chapter berulang terbaca berbeda: %q vs %q", rows[1][1], rows[3][1])
	}
}
