package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Pratinjau atas ekspor BNI Connect SUNGGUHAN, bila berkasnya ada di mesin ini.
//
// Berkasnya tidak ikut repo: isinya nama dan nomor telepon anggota sebuah
// chapter, dan repositori ini publik. Tes ini dilewati tanpa variabel
// lingkungannya, dan dengan variabel itu ia membuktikan ketiga tata letak
// nyata terbaca sampai ke baris terakhir, bukan hanya tiruannya.
//
//	BNI_CONNECT_SAMPLE_DIR=~/Downloads go test ./internal/importer -run Sample -v
func TestBNIConnectSampleAsli(t *testing.T) {
	dir := os.Getenv("BNI_CONNECT_SAMPLE_DIR")
	if dir == "" {
		t.Skip("BNI_CONNECT_SAMPLE_DIR tidak diset")
	}
	berkas, _ := filepath.Glob(filepath.Join(dir, "Chapter_*_Report_*.xls"))
	if len(berkas) == 0 {
		t.Skipf("tidak ada Chapter_*_Report_*.xls di %s", dir)
	}
	for _, f := range berkas {
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			st := &stub{chapters: map[string]ChapterRow{"ch-rise": {ID: "ch-rise"}}, members: map[string]MemberRow{}}
			h, err := NewService(st).Jalankan(context.Background(), JenisMember, data, false, Opsi{Chapter: "ch-rise"})
			if err != nil {
				t.Fatalf("jalankan: %v", err)
			}
			if h.Format != FormatSpreadsheetML || h.Total == 0 || h.Ditolak != 0 {
				t.Errorf("format=%s total=%d baru=%d ditolak=%d", h.Format, h.Total, h.Baru, h.Ditolak)
			}
			// Yang dicetak hanya ringkasan dan kolom mana yang terisi, bukan
			// datanya.
			var jatuhTempo, bergabung, telepon, bidang int
			for _, b := range bacaBNIConnect(mustRows(t, data), deteksi(t, data)) {
				if b.JatuhTempo != "" {
					jatuhTempo++
				}
				if b.Bergabung != "" {
					bergabung++
				}
				if b.Telepon != "" {
					telepon++
				}
				if b.Industri != "" {
					bidang++
				}
			}
			t.Logf("%s: %d baris, jatuh tempo %d, bergabung %d, telepon %d, bidang %d, peringatan %d",
				h.Format, h.Total, jatuhTempo, bergabung, telepon, bidang, len(h.Peringatan))
		})
	}
}

func mustRows(t *testing.T, data []byte) []sheetRow {
	t.Helper()
	rows, _, err := Baca(data)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func deteksi(t *testing.T, data []byte) laporanBNI {
	t.Helper()
	lap, ok := deteksiBNIConnect(mustRows(t, data))
	if !ok {
		t.Fatal("bukan laporan BNI Connect")
	}
	return lap
}
