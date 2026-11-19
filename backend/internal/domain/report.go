package domain

import "time"

// ReportFilter membatasi laporan ke satu periode dan, bila diisi, satu tipe
// invoice. From dan To berformat YYYY-MM-DD dan keduanya inklusif; kosong
// berarti tanpa batas di sisi itu.
//
// Dua dasar waktu dipakai sekaligus dan sengaja tidak dicampur:
//
//	ditagih   invoice yang TERBIT pada periode (created_at)
//	diterima  pembayaran yang MASUK pada periode (paid_at)
type ReportFilter struct {
	From string
	To   string
	Type string
}

// ReportChapterRow adalah satu chapter di laporan, termasuk yang belum punya
// invoice sama sekali: chapter dengan nol tagihan adalah chapter yang belum
// disentuh, dan menghilangkannya dari tabel membuatnya terbaca tidak ada.
type ReportChapterRow struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	// Ditagih dipecah per tipe: Renewal + Pendaftaran == Ditagih.
	Ditagih     int64 `json:"ditagih"`
	Renewal     int64 `json:"renewal"`
	Pendaftaran int64 `json:"pendaftaran"`
	Diterima    int64 `json:"diterima"`
	Outstanding int64 `json:"outstanding"`
	Rate        int   `json:"rate"`
}

type ReportMonthly struct {
	Month  string `json:"month"` // YYYY-MM
	Issued int64  `json:"issued"`
	Paid   int64  `json:"paid"`
}

type ReportMethod struct {
	Method string `json:"method"`
	Count  int    `json:"count"`
	Amount int64  `json:"amount"`
}

// ReportSummary adalah seluruh angka halaman Laporan selain tabel detail.
type ReportSummary struct {
	Count       int   `json:"count"`
	Ditagih     int64 `json:"ditagih"`
	Diterima    int64 `json:"diterima"`
	Outstanding int64 `json:"outstanding"`
	// Rate adalah collection rate dalam persen bulat: diterima terhadap ditagih.
	Rate     int                `json:"rate"`
	Chapters []ReportChapterRow `json:"chapters"`
	// Types menghitung invoice yang terbit per tipe.
	Types map[InvoiceType]int `json:"types"`
	// Monthly memuat paling banyak 12 bulan terakhir yang punya data.
	Monthly []ReportMonthly `json:"monthly"`
	Methods []ReportMethod  `json:"methods"`
	// DetailDiterima menjumlahkan kolom diterima di SELURUH baris detail:
	// pembayaran yang masuk pada periode untuk invoice yang terbit pada periode.
	// Lebih kecil dari Diterima bila ada pembayaran untuk invoice terbitan
	// periode lain. Tabel detail berhalaman, jadi jumlah ini tidak bisa lagi
	// dihitung dari baris yang tampil.
	DetailDiterima int64 `json:"detailDiterima"`
}

// ReportDetailRow adalah satu invoice yang terbit pada periode.
type ReportDetailRow struct {
	ID         string        `json:"id"`
	Number     string        `json:"number"`
	Member     *string       `json:"member"`
	Chapter    *string       `json:"chapter"`
	Type       InvoiceType   `json:"type"`
	Terbit     time.Time     `json:"terbit"`
	JatuhTempo Date          `json:"jatuhTempo"`
	Status     InvoiceStatus `json:"status"`
	Ditagih    int64         `json:"ditagih"`
	// Diterima adalah pembayaran yang MASUK pada periode untuk invoice ini.
	Diterima int64 `json:"diterima"`
}

// CollectionRate membulatkan diterima/ditagih ke persen terdekat. Tanpa
// tagihan hasilnya nol, bukan pembagian dengan nol.
func CollectionRate(diterima, ditagih int64) int {
	if ditagih <= 0 {
		return 0
	}
	// Pembulatan setengah ke atas dengan bilangan bulat, supaya hasilnya tidak
	// bergantung pada ketelitian float.
	return int((diterima*200 + ditagih) / (ditagih * 2))
}
