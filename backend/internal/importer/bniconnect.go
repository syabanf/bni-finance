package importer

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/syabanf/bni-finance/backend/internal/domain"
)

// Laporan ekspor BNI Connect, dibaca apa adanya.
//
// BNI Connect tidak mengekspor daftar member dengan kolom id; yang ada hanya
// NAMA. Tiga laporannya punya tata letak yang berbeda-beda, dan laporan Dues
// menumpuk EMPAT tabel di satu lembar (anggota aktif, anggota baru, anggota
// terlambat, anggota keluar), masing-masing dengan baris judul sendiri yang
// kolomnya tersebar di posisi berbeda. Pemeta judul generik hanya melihat
// baris pertama yang berisi, yaitu judul laporannya, dan berhenti di situ.
//
// Pembaca ini berjalan di seluruh lembar: setiap baris yang memuat "Member
// Name" (atau "First Name" + "Last Name") menjadi judul baru yang berlaku
// untuk baris-baris di bawahnya, sampai judul berikutnya. Baris pemisah dan
// tanggal parameter jatuh ke kolom nama yang kosong dan dilewati.
//
// Pencocokan ke member tersimpan memakai nama yang dinormalkan, DI DALAM
// chapter tujuan saja. Nama bukan kunci yang andal lintas chapter, tapi di
// dalam satu chapter BNI ia unik dalam praktiknya, dan laporan BNI Connect
// memang selalu per chapter. Yang tidak ketemu dibuat sebagai member baru
// dengan id turunan dari nama, deterministik supaya impor ulang laporan yang
// sama memperbarui baris yang sama alih-alih menggandakannya.

type laporanBNI string

const (
	laporanDues   laporanBNI = "Membership Dues Report"
	laporanRoster laporanBNI = "Chapter Roster Report"
	laporanLength laporanBNI = "Membership Length Report"
)

// barisBNI adalah satu member seperti yang muncul di laporan.
type barisBNI struct {
	Nomor      int
	Nama       string
	Industri   string
	Perusahaan string
	Telepon    string
	Status     string // sudah dipetakan ke member_status, atau kosong
	JatuhTempo string // YYYY-MM-DD atau kosong
	Bergabung  string // YYYY-MM-DD atau kosong
}

var stempelWaktu = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// deteksiBNIConnect mengenali laporan dari kombinasi judul kolomnya, di baris
// mana pun. Urutan pemeriksaannya tidak penting: ketiganya saling asing.
func deteksiBNIConnect(rows []sheetRow) (laporanBNI, bool) {
	for _, r := range rows {
		ada := map[string]bool{}
		for _, sel := range r {
			if n := normalJudul(sel); n != "" {
				ada[n] = true
			}
		}
		switch {
		case ada["membername"] && ada["duedate"]:
			return laporanDues, true
		case ada["membername"] && ada["classification"] && ada["phone"]:
			return laporanRoster, true
		case ada["firstname"] && ada["lastname"] && ada["cumulativestartdate"]:
			return laporanLength, true
		}
	}
	return "", false
}

// adalahJudulBNI melaporkan baris ini judul tabel, dan mengembalikan petanya.
func adalahJudulBNI(r sheetRow) (map[string]int, bool) {
	kolom := map[string]int{}
	for i, sel := range r {
		if n := normalJudul(sel); n != "" {
			if _, dup := kolom[n]; !dup {
				kolom[n] = i
			}
		}
	}
	_, nama := kolom["membername"]
	_, depan := kolom["firstname"]
	_, belakang := kolom["lastname"]
	if nama || (depan && belakang) {
		return kolom, true
	}
	return nil, false
}

func selBNI(r sheetRow, kolom map[string]int, nama string) string {
	i, ok := kolom[nama]
	if !ok || i >= len(r) {
		return ""
	}
	return strings.TrimSpace(r[i])
}

// tanggalBNI memotong "2027-05-01T00:00:00.000" menjadi "2027-05-01".
func tanggalBNI(v string) string {
	if stempelWaktu.MatchString(v) {
		return v[:10]
	}
	return ""
}

// statusBNI memetakan "Membership Status" BNI Connect ke member_status.
//
// Hanya dua nilai yang pernah terlihat di laporan sungguhan: Active dan
// Dropped. Yang lain dikembalikan kosong, yang berarti status tersimpan tidak
// disentuh; menebak pemetaan untuk nilai yang belum pernah dilihat adalah cara
// menonaktifkan member yang masih aktif.
func statusBNI(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "active":
		return string(domain.MemberActive)
	case "dropped", "expired":
		return string(domain.MemberInactive)
	}
	return ""
}

// bidangBNI mengambil cabang terakhir dari klasifikasi bertingkat
// "Computer & Programming > IT Consultants > IT Consultants".
func bidangBNI(klasifikasi string) string {
	bagian := strings.Split(klasifikasi, ">")
	return strings.TrimSpace(bagian[len(bagian)-1])
}

// bacaBNIConnect membaca seluruh tabel dalam laporan menjadi baris member.
func bacaBNIConnect(rows []sheetRow, jenis laporanBNI) []barisBNI {
	var out []barisBNI
	var kolom map[string]int
	for i, r := range rows {
		if k, ok := adalahJudulBNI(r); ok {
			kolom = k
			continue
		}
		if kolom == nil {
			continue
		}
		nama := selBNI(r, kolom, "membername")
		if nama == "" {
			depan, belakang := selBNI(r, kolom, "firstname"), selBNI(r, kolom, "lastname")
			nama = strings.TrimSpace(depan + " " + belakang)
		}
		// Baris pemisah ("Late Members Since", tanggal parameter) jatuh ke
		// kolom nama yang kosong atau berisi stempel waktu.
		if nama == "" || stempelWaktu.MatchString(nama) {
			continue
		}
		b := barisBNI{Nomor: i + 1, Nama: nama}
		switch jenis {
		case laporanDues:
			b.Industri = selBNI(r, kolom, "industry")
			b.Status = statusBNI(selBNI(r, kolom, "membershipstatus"))
			b.JatuhTempo = tanggalBNI(selBNI(r, kolom, "duedate"))
			b.Bergabung = tanggalBNI(selBNI(r, kolom, "startdate"))
		case laporanRoster:
			b.Industri = bidangBNI(selBNI(r, kolom, "classification"))
			b.Perusahaan = selBNI(r, kolom, "companyname")
			b.Telepon = selBNI(r, kolom, "phone")
		case laporanLength:
			b.Bergabung = tanggalBNI(selBNI(r, kolom, "cumulativestartdate"))
		}
		out = append(out, b)
	}
	return out
}

var bukanAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// normalNama menyamakan "Anasthasia  Winna" dan "anasthasia winna".
func normalNama(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// idBNI menurunkan id member baru dari chapter dan namanya.
//
// Deterministik, bukan acak: laporan yang sama diimpor dua kali harus
// menyentuh baris yang sama. Awalan "bc-" menandai asalnya, supaya id dari
// sinkronisasi BNI VM dan id dari laporan BNI Connect tidak pernah bertabrakan.
func idBNI(chapter, nama string) string {
	slug := strings.Trim(bukanAlnum.ReplaceAllString(strings.ToLower(nama), "-"), "-")
	if len(slug) > 40 {
		slug = strings.TrimRight(slug[:40], "-")
	}
	return "bc-" + strings.TrimPrefix(chapter, "ch-") + "-" + slug
}

// membersBNIConnect adalah jalur impor member untuk laporan BNI Connect.
func (s *Service) membersBNIConnect(ctx context.Context, rows []sheetRow, jenis laporanBNI,
	format Format, terapkan bool, opsi Opsi) (*Hasil, error) {

	// Laporan BNI Connect tidak memuat chapter yang bisa dicocokkan ke id kita
	// (Dues menyebutnya hanya di baris parameter, Roster tidak sama sekali),
	// jadi chapter tujuan harus datang dari tombolnya.
	if opsi.Chapter == "" {
		return nil, fmt.Errorf("laporan BNI Connect tidak memuat kolom chapter — pilih chapter tujuan dulu, lalu unggah lagi")
	}
	chapterAda, err := s.repo.ChapterIDs(ctx)
	if err != nil {
		return nil, err
	}
	if !chapterAda[opsi.Chapter] {
		return nil, fmt.Errorf("chapter %q tidak ada", opsi.Chapter)
	}
	tersimpan, err := s.repo.MemberRows(ctx)
	if err != nil {
		return nil, err
	}
	// Indeks nama hanya untuk chapter tujuan. Nama yang sama di chapter lain
	// adalah orang lain, atau setidaknya bukan urusan impor ini.
	perNama := map[string]MemberRow{}
	for _, m := range tersimpan {
		if m.ChapterID == opsi.Chapter {
			perNama[normalNama(m.Name)] = m
		}
	}

	hasil := &Hasil{Format: format, Jenis: JenisMember}
	hasil.Peringatan = append(hasil.Peringatan, fmt.Sprintf(
		"Laporan BNI Connect (%s). Member dicocokkan lewat NAMA di chapter %s; yang belum ada dibuat baru dengan id bc-…",
		jenis, opsi.Chapter))
	if jenis == laporanLength {
		hasil.Peringatan = append(hasil.Peringatan,
			"Kolom Recent Length dan Recent Start date tidak disimpan — sistem ini hanya mencatat satu tanggal bergabung")
	}

	var tulis []MemberRow
	terlihat := map[string]int{}
	for _, br := range bacaBNIConnect(rows, jenis) {
		b := Baris{Nomor: br.Nomor, Nama: br.Nama}
		kunci := normalNama(br.Nama)
		if n := terlihat[kunci]; n > 0 {
			b.Tindakan = TindakanDitolak
			b.Alasan = fmt.Sprintf("nama %q sudah muncul di baris %d", br.Nama, n)
			hasil.Ditolak++
			hasil.Baris = append(hasil.Baris, b)
			continue
		}
		terlihat[kunci] = br.Nomor

		lama, ada := perNama[kunci]
		row := MemberRow{
			ChapterID:     opsi.Chapter,
			Name:          br.Nama,
			Phone:         br.Telepon,
			Company:       br.Perusahaan,
			BusinessField: br.Industri,
			RenewalDate:   br.JatuhTempo,
			JoinedDate:    br.Bergabung,
		}
		switch {
		case ada:
			row.ID = lama.ID
			// Status hanya berubah bila laporannya menyebutkannya. Roster dan
			// Length tidak punya kolom status, dan impor dari keduanya tidak
			// boleh menyalakan atau mematikan keanggotaan siapa pun.
			row.Status = lama.Status
			if br.Status != "" {
				row.Status = br.Status
			}
			b.Perubahan = bedaMember(lama, row)
			if len(b.Perubahan) == 0 {
				b.Tindakan = TindakanSama
				hasil.Sama++
			} else {
				b.Tindakan = TindakanDiperbarui
				hasil.Diperbarui++
			}
		default:
			row.ID = idBNI(opsi.Chapter, br.Nama)
			row.Status = opsi.StatusBawaan
			if br.Status != "" {
				row.Status = br.Status
			}
			b.Tindakan = TindakanBaru
			hasil.Baru++
		}
		b.ID = row.ID
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
