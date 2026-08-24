package paperid

import (
	"context"
	"strings"
	"testing"
)

// Pengiriman massal harus TETAP JALAN saat sebagian gagal.
//
// Ini sifat yang paling menentukan, dan alasannya bukan kenyamanan: tiap
// pengiriman membakar nomor invoice Paper.id secara PERMANEN. Berhenti di tengah
// meninggalkan sebagian terkirim dan sebagian tidak — dan tanpa daftar yang
// menyebut mana yang mana, mengulang seluruhnya bukan pilihan karena nomor yang
// sudah terpakai tidak bisa dipakai ulang.
func TestBulkTidakBerhentiSaatSatuGagal(t *testing.T) {
	// Gateway TANPA hasil: tiap pengiriman gagal. Yang diuji di sini adalah
	// bahwa kegagalan tidak menghentikan sisanya — bukan aturan Paper.id-nya.
	store := &stubStore{sendable: draftSendable()}
	svc := newService(store, &stubGateway{}, "rahasia")

	hasil, err := svc.SendBulk(context.Background(), BulkInput{
		InvoiceIDs: []string{"inv-1", "inv-2", "inv-3"},
	})
	if err != nil {
		t.Fatalf("SendBulk mengembalikan galat menyeluruh: %v", err)
	}
	if hasil.Total != 3 {
		t.Errorf("total = %d, mau 3", hasil.Total)
	}
	if len(hasil.Baris) != 3 {
		t.Fatalf("baris = %d, mau 3 — pengiriman berhenti di tengah", len(hasil.Baris))
	}
	// Tiap baris harus menyebut nasibnya sendiri.
	for _, b := range hasil.Baris {
		if !b.Berhasil && b.Alasan == "" {
			t.Errorf("baris %s gagal tanpa alasan — orang harus membuka blackbox satu per satu", b.InvoiceID)
		}
	}
	if hasil.Berhasil+hasil.Gagal != hasil.Total {
		t.Errorf("berhasil %d + gagal %d != total %d", hasil.Berhasil, hasil.Gagal, hasil.Total)
	}
}

// Id ganda dibuang: pengiriman kedua atas invoice yang sama pasti gagal karena
// statusnya sudah sent, dan laporannya akan memuat kegagalan membingungkan atas
// sesuatu yang sebenarnya berhasil.
func TestBulkMembuangIDGanda(t *testing.T) {
	store := &stubStore{sendable: draftSendable()}
	svc := newService(store, &stubGateway{}, "rahasia")

	hasil, err := svc.SendBulk(context.Background(), BulkInput{
		InvoiceIDs: []string{"inv-1", "inv-1", " inv-1 ", "", "inv-2"},
	})
	if err != nil {
		t.Fatalf("SendBulk: %v", err)
	}
	if hasil.Total != 2 {
		t.Errorf("total = %d, mau 2 (inv-1 dan inv-2)", hasil.Total)
	}
}

// Batas atas menolak SEBELUM satu pun panggilan dilakukan.
func TestBulkMenolakDaftarTerlaluPanjang(t *testing.T) {
	svc := newService(&stubStore{sendable: draftSendable()}, &stubGateway{}, "rahasia")
	ids := make([]string, MaksBulk+1)
	for i := range ids {
		ids[i] = "inv-" + strings.Repeat("x", i%3+1)
	}
	if _, err := svc.SendBulk(context.Background(), BulkInput{InvoiceIDs: ids}); err == nil {
		t.Fatal("daftar melebihi batas diterima")
	} else if statusOf(err) != 400 {
		t.Errorf("status = %d, mau 400", statusOf(err))
	}
}

func TestBulkMenolakDaftarKosong(t *testing.T) {
	svc := newService(&stubStore{sendable: draftSendable()}, &stubGateway{}, "rahasia")
	if _, err := svc.SendBulk(context.Background(), BulkInput{}); err == nil {
		t.Fatal("daftar kosong diterima")
	}
}

// Context yang dibatalkan menghentikan sisanya, TAPI yang sudah terkirim tetap
// dilaporkan — klien yang menutup koneksi tidak boleh kehilangan catatan tentang
// nomor yang telanjur terbakar.
func TestBulkContextDibatalkanTetapMelaporkanYangSudah(t *testing.T) {
	svc := newService(&stubStore{sendable: draftSendable()}, &stubGateway{}, "rahasia")
	ctx, batal := context.WithCancel(context.Background())
	batal()

	hasil, err := svc.SendBulk(ctx, BulkInput{InvoiceIDs: []string{"inv-1", "inv-2"}})
	if err != nil {
		t.Fatalf("SendBulk: %v", err)
	}
	if hasil.Total != 2 {
		t.Errorf("total = %d, mau tetap 2", hasil.Total)
	}
	if len(hasil.Baris) > hasil.Total {
		t.Errorf("baris %d melebihi total %d", len(hasil.Baris), hasil.Total)
	}
}
