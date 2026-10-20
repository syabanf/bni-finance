package importer

import (
	"context"
	"strings"
	"testing"
)

func jalankanOpsiErr(t *testing.T, st *stub, jenis Jenis, doc string, terapkan bool, opsi Opsi) (*Hasil, error) {
	t.Helper()
	return NewService(st).Jalankan(context.Background(), jenis, []byte(doc), terapkan, opsi)
}

// LAPORAN CHAPTER LAIN DITOLAK, SEBELUM SATU NAMA PUN TERTEMPEL KE TUJUAN.
//
// Laporan BNI Connect tidak membawa chapter per baris; satu-satunya penanda
// ada di baris parameter "Chapter: Supreme". Tanpa penjaga, berkas chapter
// Supreme yang diunggah ke BNI Rise membuat 54 member "baru" di Rise, dan
// pratinjaunya terlihat wajar.
func TestBNIConnectMenolakLaporanChapterLain(t *testing.T) {
	laporan := func(chapter string) string {
		return sml(
			baris(sel(0, "Chapter ► Chapter Roster Report")),
			baris(sel(0, "Chapter:"), sel(12, chapter)),
			baris(sel(0, "Member Name"), sel(4, "Classification"), sel(8, "Company Name"), sel(12, "Phone")),
			baris(sel(0, "Budi Santoso"), sel(4, "Bisnis > Konstruksi"), sel(8, "PT Uji"), sel(12, "0812")),
		)
	}

	t.Run("chapter lain ditolak", func(t *testing.T) {
		st := stubBNI()
		_, err := jalankanOpsiErr(t, st, JenisMember, laporan("Supreme"), true, Opsi{Chapter: "ch-rise"})
		if err == nil || !strings.Contains(err.Error(), `milik chapter "Supreme"`) {
			t.Fatalf("harus ditolak dengan menyebut chapter laporannya, dapat %v", err)
		}
		if len(st.tulisMem) != 0 {
			t.Errorf("tidak boleh ada yang tertulis: %+v", st.tulisMem)
		}
	})

	for _, nama := range []string{"Rise", "BNI Rise", "rise", "  RISE "} {
		t.Run("cocok: "+nama, func(t *testing.T) {
			if _, err := jalankanOpsiErr(t, stubBNI(), JenisMember, laporan(nama), false, Opsi{Chapter: "ch-rise"}); err != nil {
				t.Errorf("%q harus cocok dengan BNI Rise: %v", nama, err)
			}
		})
	}
}

func TestSamaChapter(t *testing.T) {
	c := ChapterRow{ID: "ch-supreme", Name: "Supreme", DisplayName: "BNI Supreme"}
	for _, n := range []string{"Supreme", "BNI Supreme", "supreme", "ch-supreme"} {
		if !samaChapter(n, c) {
			t.Errorf("%q harus cocok", n)
		}
	}
	// Nama yang diawali "ch" tidak boleh terpotong menjadi nama lain.
	if !samaChapter("Champion", ChapterRow{ID: "ch-champion", Name: "Champion", DisplayName: "BNI Champion"}) {
		t.Error("Champion harus cocok dengan BNI Champion")
	}
	for _, n := range []string{"Supreme Jakarta", "Rise", ""} {
		if samaChapter(n, c) {
			t.Errorf("%q tidak boleh cocok", n)
		}
	}
}
