package importer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/syabanf/bni-finance/backend/internal/domain"
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
// Opsi mempersempit impor ke satu chapter, dan menentukan status bawaannya.
//
// Keduanya datang dari KONTEKS tempat tombol impornya ditekan — kartu chapter
// tertentu, tab Visitor tertentu — bukan dari isi berkasnya. Berkas yang
// disusun manual sering tidak memuat kolomnya sama sekali, dan menuntut orang
// menambahkan kolom chapter_id yang isinya sama di setiap baris adalah cara
// paling mudah menghasilkan satu baris yang salah ketik.
type Opsi struct {
	// Chapter, bila diisi, adalah SATU-SATUNYA chapter yang boleh disentuh.
	//
	// Baris tanpa kolom chapter mengikutinya. Baris yang menyebut chapter LAIN
	// DITOLAK — tidak diam-diam dipindahkan. Memindahkan member antar chapter
	// mengubah ke mana tagihannya pergi dan pendapatan siapa yang bertambah;
	// itu keputusan yang harus diambil orang, bukan efek samping impor.
	Chapter string
	// StatusBawaan dipakai untuk baris yang tidak punya kolom status.
	//
	// Kosong berarti "active", seperti sebelumnya. Diisi "visitor" oleh tombol
	// impor tamu: daftar hadir pertemuan tidak pernah memuat kolom status, dan
	// tanpa ini setiap tamu masuk sebagai anggota penuh.
	StatusBawaan string
}

func (s *Service) Jalankan(ctx context.Context, jenis Jenis, data []byte, terapkan bool, opsi Opsi) (*Hasil, error) {
	rows, format, err := Baca(data)
	if err != nil {
		return nil, err
	}
	tabel, err := BuatTabel(rows)
	if err != nil {
		return nil, err
	}

	if opsi.StatusBawaan == "" {
		opsi.StatusBawaan = string(domain.MemberActive)
	}
	if !domain.MemberStatus(opsi.StatusBawaan).Valid() {
		return nil, fmt.Errorf("status bawaan %q tidak dikenal (%s)", opsi.StatusBawaan, daftarStatus())
	}

	switch jenis {
	case JenisChapter:
		if opsi.Chapter != "" {
			// Impor chapter yang "ditujukan ke satu chapter" tidak punya arti
			// yang bisa dipertahankan: berkasnya justru mendefinisikan chapter.
			// Menerimanya diam-diam berarti mengabaikan niat pemanggilnya.
			return nil, fmt.Errorf("impor chapter tidak bisa dibatasi ke satu chapter")
		}
		return s.chapters(ctx, tabel, format, terapkan)
	case JenisMember:
		return s.members(ctx, tabel, format, terapkan, opsi)
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

func (s *Service) members(ctx context.Context, t *Tabel, format Format, terapkan bool, opsi Opsi) (*Hasil, error) {
	wajib := [][]string{
		{"id", "member_id", "memberid", "kode"},
		{"name", "nama"},
	}
	// Kolom chapter hanya wajib bila impornya TIDAK ditujukan ke satu chapter.
	//
	// Kalau tujuannya sudah ditentukan tombolnya, menuntut kolom yang isinya
	// sama di setiap baris tidak menambah kejelasan apa pun — ia hanya menambah
	// satu tempat untuk salah ketik, dan salah ketik di kolom itu memindahkan
	// member ke chapter lain beserta tagihannya.
	if opsi.Chapter == "" {
		wajib = append(wajib, []string{"chapter_id", "chapterid", "chapter"})
	}
	for _, w := range wajib {
		if !t.Punya(w...) {
			return nil, fmt.Errorf("kolom %s tidak ditemukan — judul yang terbaca: %s",
				w[0], strings.Join(t.Judul, ", "))
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
			status = opsi.StatusBawaan
		}
		// Baris tanpa chapter mengikuti chapter tujuan; yang menyebut chapter
		// lain ditolak di bawah, bukan ditimpa.
		if chapter == "" && opsi.Chapter != "" {
			chapter = opsi.Chapter
		}

		switch {
		case id == "":
			b.Tindakan, b.Alasan = TindakanDitolak, "id kosong"
		case nama == "":
			b.Tindakan, b.Alasan = TindakanDitolak, "name kosong"
		case chapter == "":
			b.Tindakan, b.Alasan = TindakanDitolak, "chapter_id kosong"
		case opsi.Chapter != "" && chapter != opsi.Chapter:
			// DITOLAK, bukan dipindahkan diam-diam.
			//
			// Impor ini ditujukan ke satu chapter, dan baris ini menyebut
			// chapter lain. Menurutinya berarti memindahkan member keluar dari
			// tempat yang sedang dibuka orangnya — dan bersamanya, ke mana
			// tagihannya pergi. Menimpanya dengan chapter tujuan sama buruknya:
			// berkasnya menyatakan sesuatu, dan kita mengabaikannya tanpa
			// memberi tahu.
			b.Tindakan = TindakanDitolak
			b.Alasan = fmt.Sprintf("baris ini menyebut chapter %q, sedangkan impor ini ditujukan ke %q",
				chapter, opsi.Chapter)
		case !chapterAda[chapter]:
			// Chapter yang tidak ada adalah kesalahan paling sering pada berkas
			// yang disusun manual, dan yang paling merusak bila lolos: member
			// berpindah ke chapter yang salah, dan pendapatan chapter ikut salah
			// hitung tanpa tanda apa pun.
			b.Tindakan = TindakanDitolak
			b.Alasan = fmt.Sprintf("chapter %q tidak ada", chapter)
		case !domain.MemberStatus(status).Valid():
			b.Tindakan = TindakanDitolak
			b.Alasan = fmt.Sprintf("status %q tidak dikenal (%s)", status, daftarStatus())
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

// daftarStatus merangkai status yang sah untuk pesan galat.
//
// Dibaca dari domain, bukan ditulis ulang di sini: daftar yang disalin akan
// tertinggal saat status baru ditambahkan, dan yang tertinggal justru pesan
// yang dibaca orang saat imporya gagal — menyuruh mereka memakai nilai yang
// tidak lagi lengkap.
func daftarStatus() string {
	nama := make([]string, 0, len(domain.SemuaStatusMember))
	for _, s := range domain.SemuaStatusMember {
		nama = append(nama, string(s))
	}
	return strings.Join(nama, "/")
}
