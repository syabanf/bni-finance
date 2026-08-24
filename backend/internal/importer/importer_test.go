package importer

import (
	"context"
	"strings"
	"testing"
)

// stub menyimpan data di memori, sehingga tes ini menguji ATURAN importnya —
// bukan SQL-nya.
type stub struct {
	chapters map[string]ChapterRow
	members  map[string]MemberRow
	tulisCh  []ChapterRow
	tulisMem []MemberRow
}

func stubIsi() *stub {
	return &stub{
		chapters: map[string]ChapterRow{
			"ch-garuda": {ID: "ch-garuda", Name: "Garuda", DisplayName: "BNI Garuda",
				AreaName: "Jakarta Pusat", CityName: "Jakarta"},
		},
		members: map[string]MemberRow{
			"mem-001": {ID: "mem-001", ChapterID: "ch-garuda", Name: "Budi Santoso",
				Email: "budi@lama.id", Phone: "08111", Status: "active"},
		},
	}
}

func (s *stub) ChapterIDs(context.Context) (map[string]bool, error) {
	out := map[string]bool{}
	for id := range s.chapters {
		out[id] = true
	}
	return out, nil
}
func (s *stub) ChapterRows(context.Context) (map[string]ChapterRow, error) { return s.chapters, nil }
func (s *stub) MemberRows(context.Context) (map[string]MemberRow, error)   { return s.members, nil }
func (s *stub) UpsertChapters(_ context.Context, rows []ChapterRow) error {
	s.tulisCh = rows
	return nil
}
func (s *stub) UpsertMembers(_ context.Context, rows []MemberRow) error {
	s.tulisMem = rows
	return nil
}

func jalankan(t *testing.T, st *stub, jenis Jenis, csv string, terapkan bool) *Hasil {
	t.Helper()
	h, err := NewService(st).Jalankan(context.Background(), jenis, []byte(csv), terapkan)
	if err != nil {
		t.Fatalf("jalankan: %v", err)
	}
	return h
}

// PRATINJAU TIDAK BOLEH MENULIS APA PUN.
//
// Sifat terpenting fitur ini. Pratinjau yang diam-diam menulis mengubah "lihat
// dulu apa yang akan terjadi" menjadi "sudah terjadi", dan pada data
// keanggotaan itu tidak bisa dibatalkan.
func TestPratinjauTidakMenulis(t *testing.T) {
	st := stubIsi()
	h := jalankan(t, st, JenisMember,
		"id,chapter_id,name,email\nmem-900,ch-garuda,Baru Sekali,baru@contoh.id\n", false)

	if h.Diterapkan {
		t.Error("pratinjau melaporkan dirinya diterapkan")
	}
	if st.tulisMem != nil {
		t.Errorf("pratinjau MENULIS %d baris", len(st.tulisMem))
	}
	if h.Baru != 1 {
		t.Errorf("baru = %d, mau 1", h.Baru)
	}
}

func TestTerapkanMenulis(t *testing.T) {
	st := stubIsi()
	h := jalankan(t, st, JenisMember,
		"id,chapter_id,name,email\nmem-900,ch-garuda,Baru Sekali,baru@contoh.id\n", true)
	if !h.Diterapkan || len(st.tulisMem) != 1 {
		t.Errorf("terapkan tidak menulis: diterapkan=%v, baris=%d", h.Diterapkan, len(st.tulisMem))
	}
}

// Chapter yang tidak ada adalah kesalahan paling merusak yang bisa lolos:
// member berpindah ke chapter yang salah, dan pendapatan chapter ikut salah
// hitung tanpa tanda apa pun.
func TestChapterTidakAdaDitolakDenganAlasan(t *testing.T) {
	st := stubIsi()
	h := jalankan(t, st, JenisMember,
		"id,chapter_id,name\nmem-901,ch-salah-ketik,Andi\n", false)

	if h.Ditolak != 1 {
		t.Fatalf("ditolak = %d, mau 1", h.Ditolak)
	}
	if !strings.Contains(h.Baris[0].Alasan, "ch-salah-ketik") {
		t.Errorf("alasan = %q — harus menyebut chapter yang salah supaya bisa diperbaiki",
			h.Baris[0].Alasan)
	}
	if h.Baris[0].Nomor != 2 {
		t.Errorf("nomor baris = %d, mau 2 (nomor seperti di Excel)", h.Baris[0].Nomor)
	}
}

// Id ganda DI DALAM berkas: tanpa pemeriksaan, baris terakhir menang diam-diam
// dan salah satu barisnya hilang tanpa jejak.
func TestIDGandaDalamBerkasDitolak(t *testing.T) {
	st := stubIsi()
	h := jalankan(t, st, JenisMember,
		"id,chapter_id,name\nmem-902,ch-garuda,Pertama\nmem-902,ch-garuda,Kedua\n", false)

	if h.Ditolak != 1 {
		t.Fatalf("ditolak = %d, mau 1", h.Ditolak)
	}
	if !strings.Contains(h.Baris[1].Alasan, "baris 2") {
		t.Errorf("alasan = %q — harus menunjuk baris pertamanya", h.Baris[1].Alasan)
	}
}

// KOLOM YANG TIDAK ADA DI BERKAS TIDAK BOLEH MENGOSONGKAN DATA TERSIMPAN.
//
// Orang mengirim daftar nomor telepon terbaru saja, tanpa kolom email. Kalau
// kolom yang hilang diperlakukan sebagai "kosongkan", satu impor yang tampak
// wajar akan menghapus email SELURUH member — dan tanpa email, tidak ada
// invoice yang bisa terkirim.
func TestKolomYangTidakDikirimTidakMenghapusData(t *testing.T) {
	st := stubIsi()
	h := jalankan(t, st, JenisMember,
		"id,chapter_id,name,phone\nmem-001,ch-garuda,Budi Santoso,08999\n", true)

	if len(st.tulisMem) != 1 {
		t.Fatalf("baris tertulis = %d", len(st.tulisMem))
	}
	if st.tulisMem[0].Email != "" {
		t.Errorf("email terisi %q — importer mengarang nilai", st.tulisMem[0].Email)
	}
	// Yang dilaporkan berubah hanya phone, bukan email.
	if got := strings.Join(h.Baris[0].Perubahan, ","); got != "phone" {
		t.Errorf("perubahan = %q, mau \"phone\" saja", got)
	}
}

// Baris yang isinya sama persis dilaporkan "sama", bukan "diperbarui".
//
// Impor ulang berkas yang sama adalah hal yang sangat sering terjadi, dan
// melaporkan seluruhnya sebagai "diperbarui" membuat angka pratinjaunya tidak
// berarti apa-apa.
func TestBarisSamaTidakDihitungDiperbarui(t *testing.T) {
	st := stubIsi()
	h := jalankan(t, st, JenisMember,
		"id,chapter_id,name,email,phone,status\nmem-001,ch-garuda,Budi Santoso,budi@lama.id,08111,active\n", false)

	if h.Sama != 1 || h.Diperbarui != 0 {
		t.Errorf("sama=%d diperbarui=%d, mau 1 dan 0", h.Sama, h.Diperbarui)
	}
}

// Judul kolom yang salah ketik harus DILAPORKAN, bukan diabaikan diam-diam —
// itu satu-satunya petunjuk bahwa datanya tidak ikut terbaca.
func TestJudulKolomTakDikenalDilaporkan(t *testing.T) {
	st := stubIsi()
	h := jalankan(t, st, JenisMember,
		"id,chapter_id,name,emial\nmem-903,ch-garuda,Andi,a@contoh.id\n", false)

	if len(h.Peringatan) == 0 {
		t.Fatal("kolom salah ketik tidak dilaporkan sama sekali")
	}
	if !strings.Contains(h.Peringatan[0], "emial") {
		t.Errorf("peringatan = %q, harus menyebut \"emial\"", h.Peringatan[0])
	}
}

// Judul berbahasa Indonesia harus terbaca sama.
func TestJudulBahasaIndonesiaTerbaca(t *testing.T) {
	st := stubIsi()
	h := jalankan(t, st, JenisMember,
		"kode;chapter;nama;telepon;perusahaan\nmem-904;ch-garuda;Siti;08222;PT Contoh\n", true)

	if h.Ditolak != 0 {
		t.Fatalf("ditolak %d: %+v", h.Ditolak, h.Baris)
	}
	if len(st.tulisMem) != 1 || st.tulisMem[0].Phone != "08222" ||
		st.tulisMem[0].Company != "PT Contoh" {
		t.Errorf("baris tertulis = %+v", st.tulisMem)
	}
}

// Berkas tanpa kolom wajib ditolak SEBELUM apa pun ditulis, dengan pesan yang
// menyebut judul yang benar-benar terbaca — supaya orang bisa membandingkannya
// dengan berkasnya sendiri.
func TestKolomWajibHilangDitolakDenganJudulYangTerbaca(t *testing.T) {
	_, err := NewService(stubIsi()).Jalankan(context.Background(), JenisMember,
		[]byte("nomor,nama\n1,Andi\n"), true)
	if err == nil {
		t.Fatal("berkas tanpa kolom wajib diterima")
	}
	if !strings.Contains(err.Error(), "nomor") {
		t.Errorf("pesan = %q — harus menyebut judul yang terbaca", err)
	}
}

// Baris kosong di tengah berkas dilewati, bukan ditolak.
func TestBarisKosongDilewati(t *testing.T) {
	st := stubIsi()
	h := jalankan(t, st, JenisChapter,
		"id,name\nch-a,Alpha\n,\nch-b,Beta\n", false)
	if h.Total != 2 || h.Ditolak != 0 {
		t.Errorf("total=%d ditolak=%d, mau 2 dan 0", h.Total, h.Ditolak)
	}
}
