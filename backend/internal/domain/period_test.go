package domain

import (
	"slices"
	"strings"
	"testing"
)

func tgl(t *testing.T, s string) Date {
	t.Helper()
	d, err := ParseDate(s)
	if err != nil {
		t.Fatalf("tanggal uji %q: %v", s, err)
	}
	return d
}

func TestKodePeriode(t *testing.T) {
	for _, k := range []struct {
		mulai, selesai string
		mau            []string
	}{
		{"2029-01-01", "2029-06-30", []string{"2029-S1", "20290101-20290630"}},
		{"2029-07-01", "2029-12-31", []string{"2029-S2", "20290701-20291231"}},
		{"2029-01-01", "2029-12-31", []string{"2029", "20290101-20291231"}},
		{"2029-07-01", "2029-09-30", []string{"20290701-20290930"}},
		// Juli sampai Juni tahun berikutnya bukan semester mana pun.
		{"2029-07-01", "2030-06-30", []string{"20290701-20300630"}},
	} {
		if dapat := KodePeriode(tgl(t, k.mulai), tgl(t, k.selesai)); !slices.Equal(dapat, k.mau) {
			t.Errorf("%s sampai %s: dapat %v, mau %v", k.mulai, k.selesai, dapat, k.mau)
		}
	}
}

func TestTerapkanPeriode(t *testing.T) {
	str := func(s string) *string { return &s }
	awal, akhir := tgl(t, "2029-07-01"), tgl(t, "2029-12-31")

	t.Run("membuat menuntut ketiganya", func(t *testing.T) {
		for nama, in := range map[string]SaveRenewalPeriodInput{
			"tanpa nama":    {StartDate: &awal, EndDate: &akhir},
			"nama spasi":    {Name: str("  "), StartDate: &awal, EndDate: &akhir},
			"tanpa selesai": {Name: str("2029 · Jul–Des"), StartDate: &awal},
			"terbalik":      {Name: str("2029 · Jul–Des"), StartDate: &akhir, EndDate: &awal},
			"nama panjang":  {Name: str(strings.Repeat("a", MaxPeriodName+1)), StartDate: &awal, EndDate: &akhir},
		} {
			if err := in.Terapkan(&RenewalPeriod{}); err == nil {
				t.Errorf("%s: seharusnya ditolak", nama)
			}
		}
	})

	t.Run("mengubah membiarkan yang kosong", func(t *testing.T) {
		p := RenewalPeriod{Name: "lama", StartDate: awal, EndDate: akhir}
		if err := (SaveRenewalPeriodInput{Name: str(" 2029 · Jul–Des ")}).Terapkan(&p); err != nil {
			t.Fatalf("ubah nama saja: %v", err)
		}
		if p.Name != "2029 · Jul–Des" || !p.StartDate.Equal(awal.Time) || !p.EndDate.Equal(akhir.Time) {
			t.Errorf("hasil tidak sesuai: %+v", p)
		}
	})

	t.Run("periode satu hari sah", func(t *testing.T) {
		in := SaveRenewalPeriodInput{Name: str("sehari"), StartDate: &awal, EndDate: &awal}
		if err := in.Terapkan(&RenewalPeriod{}); err != nil {
			t.Errorf("seharusnya diterima: %v", err)
		}
	})
}
