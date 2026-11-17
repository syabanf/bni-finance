package domain

import "testing"

func TestKunciChapter(t *testing.T) {
	for in, want := range map[string]string{
		"Supreme":               "supreme",
		"BNI Supreme":           "supreme",
		"BNI Supreme Chapter":   "supreme",
		"bni  supreme-chapter ": "supreme",
		"ch-supreme":            "supreme",
		"ch-grow-chapter":       "grow",
		"Champion":              "champion",
		"Chapter":               "chapter",
		"":                      "",
	} {
		if got := KunciChapter(in); got != want {
			t.Errorf("KunciChapter(%q) = %q, harusnya %q", in, got, want)
		}
	}
}

// Nama organisasi ditulis lengkap di nama tampil chapter. Nama yang sengaja
// ditulis lain oleh admin tidak boleh ikut ditimpa.
func TestPanjangkanBNI(t *testing.T) {
	kasus := map[string]string{
		"BNI Supreme":                            "Business Network International Supreme",
		"bni  supreme":                           "Business Network International supreme",
		" BNI Rise ":                             "Business Network International Rise",
		"Supreme":                                "Supreme",
		"Garuda Jakarta":                         "Garuda Jakarta",
		"BNI":                                    "BNI",
		"":                                       "",
		"Business Network International Supreme": "Business Network International Supreme",
	}
	for masuk, ingin := range kasus {
		if got := PanjangkanBNI(masuk); got != ingin {
			t.Errorf("PanjangkanBNI(%q) = %q, ingin %q", masuk, got, ingin)
		}
	}
}

// Laporan BNI Connect menulis "Supreme" atau "BNI Supreme", sedangkan nama
// tampil tersimpan kini memakai nama lengkap. Ketiganya harus berkunci sama,
// kalau tidak impor membuat chapter kembar.
func TestKunciChapterMengenaliNamaLengkap(t *testing.T) {
	ingin := KunciChapter("Supreme")
	for _, n := range []string{"BNI Supreme", "Business Network International Supreme", "ch-supreme", "business network international supreme chapter"} {
		if got := KunciChapter(n); got != ingin {
			t.Errorf("KunciChapter(%q) = %q, ingin %q", n, got, ingin)
		}
	}
}
