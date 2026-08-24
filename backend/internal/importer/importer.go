package importer

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Import chapter dan member dari berkas, DENGAN PRATINJAU LEBIH DULU.
//
// Pratinjau bukan kenyamanan tambahan. Berkas keanggotaan disusun manual,
// sering hasil salin-tempel dari beberapa sumber, dan kesalahan di dalamnya
// tidak kelihatan sampai tagihannya salah kirim: chapter yang salah ketik
// membuat member pindah chapter, kolom yang tergeser membuat nomor telepon
// tersimpan sebagai nama perusahaan, dan id yang tanpa sengaja sama menimpa
// orang lain.
//
// Import yang langsung menulis menyembunyikan semua itu di balik satu kalimat
// "berhasil". Karena itu jalurnya selalu dua langkah: lihat dulu apa yang akan
// terjadi, baru terapkan.

// Jenis menentukan data apa yang diimpor.
type Jenis string

const (
	JenisChapter Jenis = "chapters"
	JenisMember  Jenis = "members"
)

// Tindakan adalah apa yang akan terjadi pada satu baris.
type Tindakan string

const (
	TindakanBaru       Tindakan = "baru"
	TindakanDiperbarui Tindakan = "diperbarui"
	TindakanSama       Tindakan = "sama"
	TindakanDitolak    Tindakan = "ditolak"
)

// Baris adalah hasil pemeriksaan satu baris berkas.
type Baris struct {
	// Nomor mengikuti nomor baris DI BERKAS, termasuk baris judul, supaya
	// orang bisa membukanya di Excel dan langsung menemukan barisnya. Nomor
	// yang dihitung dari nol setelah judul memaksa orang menghitung sendiri.
	Nomor    int      `json:"nomor"`
	ID       string   `json:"id"`
	Nama     string   `json:"nama"`
	Tindakan Tindakan `json:"tindakan"`
	// Alasan hanya terisi bila Tindakan = ditolak.
	Alasan string `json:"alasan,omitempty"`
	// Perubahan menyebut kolom apa saja yang berbeda dari yang tersimpan,
	// hanya untuk baris yang diperbarui. Tanpa ini, "12 diperbarui" tidak
	// memberi tahu apakah yang berubah nomor telepon atau seluruh chapternya.
	Perubahan []string `json:"perubahan,omitempty"`
}

// Hasil adalah ringkasan seluruh berkas.
type Hasil struct {
	Format     Format  `json:"format"`
	Jenis      Jenis   `json:"jenis"`
	Diterapkan bool    `json:"diterapkan"`
	Total      int     `json:"total"`
	Baru       int     `json:"baru"`
	Diperbarui int     `json:"diperbarui"`
	Sama       int     `json:"sama"`
	Ditolak    int     `json:"ditolak"`
	Baris      []Baris `json:"baris"`
	// Peringatan memuat hal yang tidak menggagalkan berkas tapi patut dilihat —
	// terutama judul kolom yang tidak dikenal, satu-satunya petunjuk bahwa ada
	// kolom yang salah ketik dan datanya tidak ikut terbaca.
	Peringatan []string `json:"peringatan,omitempty"`
}

// Store adalah kontrak persistensi yang dibutuhkan importer.
type Store interface {
	ChapterIDs(ctx context.Context) (map[string]bool, error)
	ChapterRows(ctx context.Context) (map[string]ChapterRow, error)
	MemberRows(ctx context.Context) (map[string]MemberRow, error)
	UpsertChapters(ctx context.Context, rows []ChapterRow) error
	UpsertMembers(ctx context.Context, rows []MemberRow) error
}

// ChapterRow dan MemberRow adalah bentuk baris yang diimpor.
type ChapterRow struct {
	ID          string
	Name        string
	DisplayName string
	AreaName    string
	CityName    string
}

type MemberRow struct {
	ID            string
	ChapterID     string
	Name          string
	Email         string
	Phone         string
	Company       string
	BusinessField string
	Status        string
}

type Service struct {
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

// Jalankan memeriksa berkas, dan menerapkannya hanya bila terapkan = true.
//
// Satu fungsi untuk keduanya, dan itu disengaja: pratinjau yang dihitung oleh
// kode yang BERBEDA dari kode yang menulis adalah pratinjau yang bisa berbohong.
// Selisih sekecil apa pun di antara keduanya berarti orang menyetujui sesuatu
// yang tidak sama dengan yang akhirnya terjadi.
func (s *Service) Jalankan(ctx context.Context, jenis Jenis, data []byte, terapkan bool) (*Hasil, error) {
	rows, format, err := Baca(data)
	if err != nil {
		return nil, err
	}
	tabel, err := BuatTabel(rows)
	if err != nil {
		return nil, err
	}

	switch jenis {
	case JenisChapter:
		return s.chapters(ctx, tabel, format, terapkan)
	case JenisMember:
		return s.members(ctx, tabel, format, terapkan)
	}
	return nil, fmt.Errorf("jenis import tidak dikenal: %q", jenis)
}

var judulChapter = []string{
	"id", "chapter_id", "chapterid", "kode",
	"name", "nama",
	"display_name", "displayname", "nama_tampilan",
	"area_name", "areaname", "area", "wilayah",
	"city_name", "cityname", "city", "kota",
}

func (s *Service) chapters(ctx context.Context, t *Tabel, format Format, terapkan bool) (*Hasil, error) {
	if !t.Punya("id", "chapter_id", "chapterid", "kode") {
		return nil, fmt.Errorf("kolom id tidak ditemukan — judul yang terbaca: %s",
			strings.Join(t.Judul, ", "))
	}
	if !t.Punya("name", "nama") {
		return nil, fmt.Errorf("kolom name tidak ditemukan — judul yang terbaca: %s",
			strings.Join(t.Judul, ", "))
	}

	tersimpan, err := s.repo.ChapterRows(ctx)
	if err != nil {
		return nil, err
	}

	hasil := &Hasil{Format: format, Jenis: JenisChapter, Peringatan: peringatanJudul(t, judulChapter)}
	var tulis []ChapterRow
	terlihat := map[string]int{}

	for i, baris := range t.Baris {
		nomor := i + 2 // +1 baris judul, +1 karena Excel mulai dari 1
		id := t.Sel(baris, "id", "chapter_id", "chapterid", "kode")
		nama := t.Sel(baris, "name", "nama")

		if id == "" && nama == "" {
			continue // baris kosong di tengah/akhir berkas
		}
		b := Baris{Nomor: nomor, ID: id, Nama: nama}
		switch {
		case id == "":
			b.Tindakan, b.Alasan = TindakanDitolak, "id kosong"
		case nama == "":
			b.Tindakan, b.Alasan = TindakanDitolak, "name kosong"
		case terlihat[id] > 0:
			// Id ganda DI DALAM BERKAS. Dibiarkan lewat, baris terakhir menang
			// diam-diam dan salah satu barisnya hilang tanpa jejak.
			b.Tindakan = TindakanDitolak
			b.Alasan = fmt.Sprintf("id %q sudah dipakai di baris %d", id, terlihat[id])
		}
		if b.Tindakan == TindakanDitolak {
			hasil.Ditolak++
			hasil.Baris = append(hasil.Baris, b)
			continue
		}
		terlihat[id] = nomor

		row := ChapterRow{
			ID:          id,
			Name:        nama,
			DisplayName: t.Sel(baris, "display_name", "displayname", "nama_tampilan"),
			AreaName:    t.Sel(baris, "area_name", "areaname", "area", "wilayah"),
			CityName:    t.Sel(baris, "city_name", "cityname", "city", "kota"),
		}
		if row.DisplayName == "" {
			row.DisplayName = nama
		}

		lama, ada := tersimpan[id]
		switch {
		case !ada:
			b.Tindakan = TindakanBaru
			hasil.Baru++
		default:
			b.Perubahan = bedaChapter(lama, row)
			if len(b.Perubahan) == 0 {
				b.Tindakan = TindakanSama
				hasil.Sama++
			} else {
				b.Tindakan = TindakanDiperbarui
				hasil.Diperbarui++
			}
		}
		tulis = append(tulis, row)
		hasil.Baris = append(hasil.Baris, b)
	}

	hasil.Total = len(hasil.Baris)
	if terapkan && len(tulis) > 0 {
		if err := s.repo.UpsertChapters(ctx, tulis); err != nil {
			return nil, err
		}
		hasil.Diterapkan = true
	}
	return hasil, nil
}

var judulMember = []string{
	"id", "member_id", "memberid", "kode",
	"chapter_id", "chapterid", "chapter",
	"name", "nama",
	"email", "surel",
	"phone", "telepon", "hp", "no_hp", "nohp",
	"company", "perusahaan",
	"business_field", "businessfield", "bidang", "bidang_usaha",
	"status",
}

func (s *Service) members(ctx context.Context, t *Tabel, format Format, terapkan bool) (*Hasil, error) {
	for _, wajib := range [][]string{
		{"id", "member_id", "memberid", "kode"},
		{"chapter_id", "chapterid", "chapter"},
		{"name", "nama"},
	} {
		if !t.Punya(wajib...) {
			return nil, fmt.Errorf("kolom %s tidak ditemukan — judul yang terbaca: %s",
				wajib[0], strings.Join(t.Judul, ", "))
		}
	}

	chapterAda, err := s.repo.ChapterIDs(ctx)
	if err != nil {
		return nil, err
	}
	tersimpan, err := s.repo.MemberRows(ctx)
	if err != nil {
		return nil, err
	}

	hasil := &Hasil{Format: format, Jenis: JenisMember, Peringatan: peringatanJudul(t, judulMember)}
	var tulis []MemberRow
	terlihat := map[string]int{}

	for i, baris := range t.Baris {
		nomor := i + 2
		id := t.Sel(baris, "id", "member_id", "memberid", "kode")
		nama := t.Sel(baris, "name", "nama")
		chapter := t.Sel(baris, "chapter_id", "chapterid", "chapter")

		if id == "" && nama == "" && chapter == "" {
			continue
		}
		b := Baris{Nomor: nomor, ID: id, Nama: nama}
		status := strings.ToLower(t.Sel(baris, "status"))
		if status == "" {
			status = "active"
		}

		switch {
		case id == "":
			b.Tindakan, b.Alasan = TindakanDitolak, "id kosong"
		case nama == "":
			b.Tindakan, b.Alasan = TindakanDitolak, "name kosong"
		case chapter == "":
			b.Tindakan, b.Alasan = TindakanDitolak, "chapter_id kosong"
		case !chapterAda[chapter]:
			// Chapter yang tidak ada adalah kesalahan paling sering pada berkas
			// yang disusun manual, dan yang paling merusak bila lolos: member
			// berpindah ke chapter yang salah, dan pendapatan chapter ikut salah
			// hitung tanpa tanda apa pun.
			b.Tindakan = TindakanDitolak
			b.Alasan = fmt.Sprintf("chapter %q tidak ada", chapter)
		case status != "active" && status != "inactive" && status != "pending":
			b.Tindakan = TindakanDitolak
			b.Alasan = fmt.Sprintf("status %q tidak dikenal (active/inactive/pending)", status)
		case terlihat[id] > 0:
			b.Tindakan = TindakanDitolak
			b.Alasan = fmt.Sprintf("id %q sudah dipakai di baris %d", id, terlihat[id])
		}
		if b.Tindakan == TindakanDitolak {
			hasil.Ditolak++
			hasil.Baris = append(hasil.Baris, b)
			continue
		}
		terlihat[id] = nomor

		row := MemberRow{
			ID:            id,
			ChapterID:     chapter,
			Name:          nama,
			Email:         t.Sel(baris, "email", "surel"),
			Phone:         t.Sel(baris, "phone", "telepon", "hp", "no_hp", "nohp"),
			Company:       t.Sel(baris, "company", "perusahaan"),
			BusinessField: t.Sel(baris, "business_field", "businessfield", "bidang", "bidang_usaha"),
			Status:        status,
		}

		lama, ada := tersimpan[id]
		switch {
		case !ada:
			b.Tindakan = TindakanBaru
			hasil.Baru++
		default:
			b.Perubahan = bedaMember(lama, row)
			if len(b.Perubahan) == 0 {
				b.Tindakan = TindakanSama
				hasil.Sama++
			} else {
				b.Tindakan = TindakanDiperbarui
				hasil.Diperbarui++
			}
		}
		tulis = append(tulis, row)
		hasil.Baris = append(hasil.Baris, b)
	}

	hasil.Total = len(hasil.Baris)
	if terapkan && len(tulis) > 0 {
		if err := s.repo.UpsertMembers(ctx, tulis); err != nil {
			return nil, err
		}
		hasil.Diterapkan = true
	}
	return hasil, nil
}

func peringatanJudul(t *Tabel, dikenal []string) []string {
	tak := t.JudulTakDikenal(dikenal)
	if len(tak) == 0 {
		return nil
	}
	sort.Strings(tak)
	return []string{fmt.Sprintf(
		"kolom tidak dikenal dan diabaikan: %s — periksa ejaannya bila kolom itu seharusnya ikut terbaca",
		strings.Join(tak, ", "))}
}

// beda* menyebutkan kolom apa saja yang berubah.
//
// Kolom yang KOSONG di berkas tidak dihitung sebagai perubahan. Berkas yang
// hanya memuat sebagian kolom adalah hal biasa — orang mengirim daftar nomor
// telepon terbaru saja — dan menganggap kolom yang tidak ada sebagai "kosongkan"
// akan menghapus email seluruh member dalam satu impor yang tampak wajar.
func bedaChapter(lama, baru ChapterRow) []string {
	var out []string
	cek := func(nama, l, b string) {
		if b != "" && b != l {
			out = append(out, nama)
		}
	}
	cek("name", lama.Name, baru.Name)
	cek("display_name", lama.DisplayName, baru.DisplayName)
	cek("area_name", lama.AreaName, baru.AreaName)
	cek("city_name", lama.CityName, baru.CityName)
	return out
}

func bedaMember(lama, baru MemberRow) []string {
	var out []string
	cek := func(nama, l, b string) {
		if b != "" && b != l {
			out = append(out, nama)
		}
	}
	cek("chapter_id", lama.ChapterID, baru.ChapterID)
	cek("name", lama.Name, baru.Name)
	cek("email", lama.Email, baru.Email)
	cek("phone", lama.Phone, baru.Phone)
	cek("company", lama.Company, baru.Company)
	cek("business_field", lama.BusinessField, baru.BusinessField)
	cek("status", lama.Status, baru.Status)
	return out
}
