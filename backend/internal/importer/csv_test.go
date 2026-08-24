package importer

import "testing"

// Ketiga kasus di bawah adalah bentuk yang benar-benar keluar dari Excel, dan
// ketiganya gagal dengan cara yang menyesatkan bila tidak ditangani.
func TestBacaCSVBentukDariExcel(t *testing.T) {
	t.Run("BOM UTF-8 di awal berkas", func(t *testing.T) {
		// Excel di Windows menaruh EF BB BF. Tanpa dibuang, judul kolom pertama
		// terbaca "<BOM>id" dan tidak pernah cocok dengan "id" — importnya
		// menolak berkas dengan pesan "kolom id tidak ada", padahal kolomnya
		// jelas terlihat di layar.
		rows, err := bacaCSV([]byte("\xEF\xBB\xBFid,name\nmem-1,Andi\n"))
		if err != nil {
			t.Fatalf("gagal: %v", err)
		}
		if rows[0][0] != "id" {
			t.Errorf("judul kolom = %q, mau \"id\" — BOM tidak dibuang", rows[0][0])
		}
	})

	t.Run("pemisah titik koma", func(t *testing.T) {
		// Excel pada locale Indonesia menulis titik koma. Dibaca sebagai koma,
		// seluruh baris menjadi SATU kolom dan tidak ada yang gagal.
		rows, err := bacaCSV([]byte("id;name;email\nmem-1;Andi;a@contoh.id\n"))
		if err != nil {
			t.Fatalf("gagal: %v", err)
		}
		if len(rows[0]) != 3 {
			t.Fatalf("jumlah kolom = %d, mau 3 — pemisah tidak terdeteksi", len(rows[0]))
		}
		if rows[1][1] != "Andi" {
			t.Errorf("nama = %q", rows[1][1])
		}
	})

	t.Run("baris lebih pendek dari judul", func(t *testing.T) {
		rows, err := bacaCSV([]byte("id,name,email\nmem-1,Andi\n"))
		if err != nil {
			t.Fatalf("baris pendek ditolak: %v", err)
		}
		if len(rows) != 2 {
			t.Errorf("jumlah baris = %d, mau 2", len(rows))
		}
	})
}

// Titik koma di dalam nilai tidak boleh membalik tebakan pemisah.
func TestPemisahDitebakDariJudulSaja(t *testing.T) {
	// Judulnya jelas berkoma; catatan di baris data memuat banyak titik koma.
	csv := "id,name,catatan\nmem-1,Andi,\"a; b; c; d; e\"\n"
	rows, err := bacaCSV([]byte(csv))
	if err != nil {
		t.Fatalf("gagal: %v", err)
	}
	if len(rows[0]) != 3 {
		t.Fatalf("jumlah kolom = %d, mau 3", len(rows[0]))
	}
	if rows[1][2] != "a; b; c; d; e" {
		t.Errorf("catatan = %q — pemisah tertipu isi baris data", rows[1][2])
	}
}

func TestBerkasBukanUTF8Ditolak(t *testing.T) {
	// Byte 0xFF tidak pernah sah di UTF-8.
	if _, err := bacaCSV([]byte("id,name\n\xFF\xFE,Andi\n")); err == nil {
		t.Error("berkas non-UTF-8 diterima; namanya akan tersimpan sebagai sampah")
	}
}
