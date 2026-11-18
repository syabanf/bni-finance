package invoice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type penyapuPalsu struct {
	hasil int
	galat error
}

func (p penyapuPalsu) TandaiOverdue(context.Context) (int, error) { return p.hasil, p.galat }

// Cache angka hanya digugurkan bila memang ada invoice yang berubah. Kalau
// setiap putaran menggugurkannya, cache dashboard tidak pernah bertahan lebih
// dari satu interval dan setiap replika mengulanginya.
func TestPutaranHanyaMengabariBilaAdaYangDitandai(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	kasus := []struct {
		nama   string
		repo   penyapuPalsu
		kabari int
	}{
		{"tiga invoice ditandai", penyapuPalsu{hasil: 3}, 1},
		{"tidak ada yang telat", penyapuPalsu{}, 0},
		{"basis data gagal", penyapuPalsu{galat: errors.New("putus")}, 0},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			dikabari := 0
			NewWorker(k.repo, log).SetelahMenandai(func(context.Context) { dikabari++ }).Putaran(context.Background())
			if dikabari != k.kabari {
				t.Errorf("dikabari %d kali, ingin %d", dikabari, k.kabari)
			}
		})
	}
}
