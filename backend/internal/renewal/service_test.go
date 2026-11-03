package renewal

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/mailer"
)

type storeUji struct {
	Store
	belumDijawab map[string]int
	akun         []PenerimaMC
}

func (s *storeUji) PenerimaMC(_ context.Context, ids []string) ([]PenerimaMC, error) {
	return slices.DeleteFunc(slices.Clone(s.akun), func(p PenerimaMC) bool {
		return !slices.Contains(ids, p.ChapterID)
	}), nil
}

type suratUji struct {
	belumSiap bool
	gagalKe   string
	terkirim  []mailer.Pesan
}

func (s *suratUji) Siap() bool { return !s.belumSiap }

func (s *suratUji) Kirim(_ context.Context, p mailer.Pesan) error {
	if p.Ke == s.gagalKe {
		return errors.New("ditolak")
	}
	s.terkirim = append(s.terkirim, p)
	return nil
}

var (
	mcA  = PenerimaMC{ID: "a1", Email: "a1@uji.test", Nama: "MC A1", ChapterID: "ch-a", NamaChapter: "BNI A"}
	mcA2 = PenerimaMC{ID: "a2", Email: "a2@uji.test", Nama: "MC A2", ChapterID: "ch-a", NamaChapter: "BNI A"}
	mcB  = PenerimaMC{ID: "b1", Email: "b1@uji.test", Nama: "MC <B>", ChapterID: "ch-b", NamaChapter: "BNI B"}
)

func minta(st *storeUji, surat *suratUji, assigned *string) HasilEmail {
	return NewService(st, surat, "https://contoh.test/").
		beritahuMC(context.Background(), st.belumDijawab, "2027-S2", assigned)
}

func keSiapa(s *suratUji) []string {
	var out []string
	for _, p := range s.terkirim {
		out = append(out, p.Ke)
	}
	slices.Sort(out)
	return out
}

// TIAP CHAPTER DIBERI TAHU KE MC-NYA SENDIRI, DENGAN JUMLAH CHAPTER ITU.
func TestMintaMengirimEmailKeMCTiapChapter(t *testing.T) {
	surat := &suratUji{}
	h := minta(&storeUji{belumDijawab: map[string]int{"ch-a": 3, "ch-b": 2}, akun: []PenerimaMC{mcA, mcA2, mcB}}, surat, nil)

	if h.Terkirim != 3 || h.Gagal != 0 {
		t.Errorf("laporan email = %+v", h)
	}
	if got := keSiapa(surat); !slices.Equal(got, []string{"a1@uji.test", "a2@uji.test", "b1@uji.test"}) {
		t.Errorf("penerima = %v", got)
	}
	for _, p := range surat.terkirim {
		switch p.Ke {
		case mcB.Email:
			if !strings.Contains(p.Teks, "2 member BNI B") || !strings.Contains(p.Teks, "Jul–Des 2027") {
				t.Errorf("isi email chapter B: %q", p.Teks)
			}
			if !strings.Contains(p.Teks, "https://contoh.test/renewal?periode=2027-S2") {
				t.Errorf("tautan harus membuka periode yang ditanyakan: %q", p.Teks)
			}
			if strings.Contains(p.HTML, "<B>") {
				t.Errorf("nama harus diloloskan di HTML: %q", p.HTML)
			}
		default:
			if !strings.Contains(p.Teks, "3 member BNI A") {
				t.Errorf("isi email chapter A: %q", p.Teks)
			}
		}
	}
}

// MC yang ditugaskan menggantikan MC lain DI CHAPTERNYA saja.
func TestMintaMCDitugaskanHanyaMempersempitChapternya(t *testing.T) {
	surat := &suratUji{}
	ditugaskan := mcA2.ID
	minta(&storeUji{belumDijawab: map[string]int{"ch-a": 3, "ch-b": 2}, akun: []PenerimaMC{mcA, mcA2, mcB}}, surat, &ditugaskan)

	if got := keSiapa(surat); !slices.Equal(got, []string{"a2@uji.test", "b1@uji.test"}) {
		t.Errorf("penerima = %v", got)
	}
}

// Yang tidak terkirim dilaporkan, supaya ST tahu harus menghubungi MC sendiri.
func TestMintaMelaporkanEmailYangTidakSampai(t *testing.T) {
	t.Run("chapter tanpa akun MC", func(t *testing.T) {
		h := minta(&storeUji{belumDijawab: map[string]int{"ch-a": 1, "ch-c": 1}, akun: []PenerimaMC{mcA}}, &suratUji{}, nil)
		if h.Terkirim != 1 || !slices.Equal(h.ChapterTanpaMC, []string{"ch-c"}) {
			t.Errorf("laporan email = %+v", h)
		}
	})
	t.Run("email belum dikonfigurasi", func(t *testing.T) {
		surat := &suratUji{belumSiap: true}
		h := minta(&storeUji{belumDijawab: map[string]int{"ch-a": 1}, akun: []PenerimaMC{mcA}}, surat, nil)
		if !h.BelumDikonfigurasi || len(surat.terkirim) != 0 {
			t.Errorf("laporan email = %+v, terkirim %d", h, len(surat.terkirim))
		}
	})
	t.Run("satu penerima gagal", func(t *testing.T) {
		h := minta(&storeUji{belumDijawab: map[string]int{"ch-a": 1}, akun: []PenerimaMC{mcA, mcA2}},
			&suratUji{gagalKe: mcA.Email}, nil)
		if h.Terkirim != 1 || h.Gagal != 1 {
			t.Errorf("laporan email = %+v", h)
		}
	})
	t.Run("tidak ada yang menunggu jawaban", func(t *testing.T) {
		surat := &suratUji{}
		h := minta(&storeUji{belumDijawab: map[string]int{}, akun: []PenerimaMC{mcA}}, surat, nil)
		if len(surat.terkirim) != 0 || h.ChapterTanpaMC == nil {
			t.Errorf("laporan email = %+v, terkirim %d", h, len(surat.terkirim))
		}
	})
}
