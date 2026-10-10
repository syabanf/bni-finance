package domain

import "testing"

// SETIAP STATUS HARUS PUNYA KEPUTUSAN TERTULIS SOAL PENAGIHAN.
//
// Bahaya yang dijaga di sini muncul dari kelalaian, bukan dari kesalahan
// berpikir: seseorang menambah status baru — "alumni", "suspended", apa pun —
// dan tidak terpikir bahwa ada tempat lain yang memutuskan siapa boleh
// ditagih. Kalau aturannya ditulis sebagai `!= visitor`, status baru itu
// otomatis ikut tertagih, dan tidak ada apa pun yang bertanya.
//
// Tabel di bawah memaksa pertanyaannya diajukan: status yang ditambahkan ke
// SemuaStatusMember tanpa baris di sini membuat tes ini memerah.
//
// Arah sebaliknya — konstanta baru yang LUPA dimasukkan ke SemuaStatusMember —
// sengaja tidak dijaga di sini, karena ia sudah gagal ke sisi yang aman dan
// gagal dengan keras: Valid() menolaknya, jadi nilai itu tidak bisa masuk lewat
// API sama sekali, dan BolehDitagih() menjawab false untuk tipe apa pun. Yang terjadi bukan
// tagihan yang salah kirim, melainkan status yang terlihat tidak berfungsi
// pada percobaan pertama.
func TestSetiapStatusPunyaKeputusanPenagihan(t *testing.T) {
	type keputusan struct{ pendaftaran, renewal bool }
	mau := map[MemberStatus]keputusan{
		MemberActive:   {pendaftaran: false, renewal: true},
		MemberInactive: {pendaftaran: false, renewal: true}, // keanggotaannya lewat, perpanjangannya tetap ditagih
		MemberPending:  {pendaftaran: true, renewal: false}, // sedang menunggu tagihan pendaftarannya
		MemberVisitor:  {pendaftaran: true, renewal: false},
		// Belum bayar biaya bergabung bila dibuat lewat Tambah Member; renewal
		// saat tahun pertamanya habis.
		MemberNewMember: {pendaftaran: true, renewal: true},
	}

	for _, s := range SemuaStatusMember {
		harap, ada := mau[s]
		if !ada {
			t.Errorf("status %q belum punya keputusan penagihan.\n"+
				"Tambahkan barisnya di sini DAN di BolehDitagih(): status baru\n"+
				"tidak boleh mewarisi perlakuan yang kebetulan berlaku.", s)
			continue
		}
		if got := s.BolehDitagih(TypeRegistration); got != harap.pendaftaran {
			t.Errorf("%q.BolehDitagih(pendaftaran) = %v, seharusnya %v", s, got, harap.pendaftaran)
		}
		if got := s.BolehDitagih(TypeRenewal); got != harap.renewal {
			t.Errorf("%q.BolehDitagih(renewal) = %v, seharusnya %v", s, got, harap.renewal)
		}
	}

	if len(mau) != len(SemuaStatusMember) {
		t.Errorf("tabel memuat %d status, SemuaStatusMember %d: ada yang dihapus dari daftar tapi tertinggal di sini",
			len(mau), len(SemuaStatusMember))
	}
}

// Status yang tidak dikenal TIDAK boleh ditagih.
//
// Nilai bisa datang dari basis data yang enum-nya lebih maju daripada binary
// ini — persis yang terjadi saat migrasi sudah jalan tapi deploy belum. Arah
// gagalnya harus ke sisi yang aman.
func TestStatusAsingTidakBolehDitagih(t *testing.T) {
	for _, s := range []MemberStatus{"", "alumni", "VISITOR", "active "} {
		for _, tipe := range []InvoiceType{TypeRegistration, TypeRenewal, InvoiceType("asing")} {
			if MemberStatus(s).BolehDitagih(tipe) {
				t.Errorf("status asing %q dianggap boleh ditagih %s", s, tipe)
			}
		}
		if MemberStatus(s).Valid() {
			t.Errorf("status asing %q dianggap sah", s)
		}
	}
}
