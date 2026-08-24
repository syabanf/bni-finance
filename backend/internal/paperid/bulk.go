package paperid

import (
	"context"
	"fmt"
	"strings"

	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// Pengiriman massal — LOOPING panggilan API, satu per invoice.
//
// Bukan satu permintaan besar, dan itu bukan sekadar mengikuti bentuk API
// Paper.id: satu permintaan gabungan yang gagal di tengah tidak memberi tahu
// invoice mana yang sudah terkirim. Karena tiap pengiriman membakar nomor
// invoice secara PERMANEN, tidak tahu apa yang sudah terkirim berarti tidak
// bisa mengulang dengan aman sama sekali.
//
// Dengan satu panggilan per invoice, tiap hasil berdiri sendiri: yang berhasil
// tercatat berhasil, yang gagal menyebutkan alasannya, dan mengulang hanya
// menyentuh yang belum terkirim.

// MaksBulk membatasi satu permintaan.
//
// Pagar, bukan aturan bisnis: pengiriman yang menahan koneksi Paper.id selama
// berjam-jam akan membuat setiap pengiriman manual di belakangnya menunggu.
// Chapter terbesar pun jauh di bawah angka ini.
const MaksBulk = 200

// BulkInput adalah permintaan pengiriman massal.
type BulkInput struct {
	InvoiceIDs []string `json:"invoiceIds"`
	SendOptions
}

func (in BulkInput) Validate() error {
	switch {
	case len(in.InvoiceIDs) == 0:
		return fmt.Errorf("invoiceIds wajib diisi")
	case len(in.InvoiceIDs) > MaksBulk:
		return fmt.Errorf("terlalu banyak invoice dalam satu pengiriman (maksimal %d)", MaksBulk)
	}
	return nil
}

// BulkBaris adalah hasil satu invoice.
type BulkBaris struct {
	InvoiceID string `json:"invoiceId"`
	Number    string `json:"number,omitempty"`
	Berhasil  bool   `json:"berhasil"`
	// Alasan diisi apa adanya dari galat yang muncul, termasuk pesan dari
	// Paper.id. Meringkasnya menjadi "gagal" membuat orang harus membuka
	// blackbox satu per satu untuk tahu apa yang sebenarnya terjadi.
	Alasan string `json:"alasan,omitempty"`
	Status int    `json:"status,omitempty"`
}

// BulkHasil merangkum seluruh pengiriman.
type BulkHasil struct {
	Total    int         `json:"total"`
	Berhasil int         `json:"berhasil"`
	Gagal    int         `json:"gagal"`
	Baris    []BulkBaris `json:"baris"`
}

// SendBulk mengirim banyak invoice, satu per satu.
//
// TIDAK BERHENTI saat satu gagal. Berhenti di tengah meninggalkan sebagian
// terkirim dan sebagian tidak, tanpa daftar yang menyebut mana yang mana — dan
// nomor invoice yang sudah terbakar tidak bisa dipakai ulang, jadi mengulang
// seluruh daftar bukan pilihan.
//
// Galat yang dikembalikan hanya untuk hal yang menggagalkan SELURUH permintaan
// (masukan tidak sah). Kegagalan per invoice masuk ke Baris, dan permintaannya
// tetap 200 — karena sebagiannya memang berhasil, dan menjawab 500 akan
// membuat klien mengira tidak ada satu pun yang terkirim.
func (s *Service) SendBulk(ctx context.Context, in BulkInput) (*BulkHasil, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}

	ids := dedupIDs(in.InvoiceIDs)
	hasil := &BulkHasil{Total: len(ids), Baris: make([]BulkBaris, 0, len(ids))}

	for _, id := range ids {
		// Context yang dibatalkan menghentikan sisanya, tapi yang SUDAH
		// terkirim tetap dilaporkan. Klien yang menutup koneksi di tengah tidak
		// boleh kehilangan catatan tentang nomor yang telanjur terbakar.
		if ctx.Err() != nil {
			break
		}

		baris := BulkBaris{InvoiceID: id}
		inv, err := s.Send(ctx, id, in.SendOptions)
		switch {
		case err != nil:
			baris.Alasan = err.Error()
			baris.Status = httpx.StatusOf(err)
			hasil.Gagal++
		default:
			baris.Berhasil = true
			if inv != nil {
				baris.Number = inv.Number
			}
			hasil.Berhasil++
		}
		hasil.Baris = append(hasil.Baris, baris)
	}
	return hasil, nil
}

// dedupIDs membuang id ganda.
//
// Id yang sama dua kali dalam satu permintaan akan menghasilkan pengiriman
// kedua yang pasti gagal — invoicenya sudah berstatus sent — dan laporannya
// memuat kegagalan yang membingungkan atas sesuatu yang sebenarnya berhasil.
func dedupIDs(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
