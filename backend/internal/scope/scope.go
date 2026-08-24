// Package scope membawa batas chapter dari middleware ke query.
//
// Dipisahkan menjadi paketnya sendiri supaya repository tidak perlu mengimpor
// paket auth. Persistensi yang bergantung pada autentikasi sulit diuji dan
// gampang melingkar; yang benar-benar dibutuhkan repository hanyalah satu
// pertanyaan — "chapter mana yang boleh dilihat?" — dan itulah seluruh isi
// paket ini.
//
// TIDAK ADA row-level security yang menegakkan batas ini. Backend menyambung ke
// Postgres sebagai SATU peran tepercaya, jadi basis data melihat satu identitas
// saja dan tidak punya apa pun untuk dipakai sebagai kunci kebijakan per-baris.
// Batas chapter hanya ada di sini dan di query yang memakainya.
package scope

import (
	"context"
	"fmt"
)

type ctxKey struct{}

type chapterScope struct {
	id       string
	terbatas bool
}

// Limit adalah batas yang berlaku untuk satu permintaan.
//
// TIGA keadaan, bukan dua, dan keadaan ketiga itulah yang paling penting:
//
//	Buntu           context tidak membawa lingkup sama sekali -> nol baris
//	Terbatas        satu chapter                              -> filter chapter
//	keduanya false  nasional                                  -> tanpa filter
//
// Versi pertama paket ini memampatkan "buntu" menjadi sebuah id sentinel berisi
// byte nol, dengan harapan `where chapter_id = <sentinel>` tidak cocok dengan
// apa pun. Postgres menolaknya mentah-mentah:
//
//	invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)
//
// Jadi jalur tanpa lingkup memang tidak membocorkan data — tapi berakhir sebagai
// 500 "terjadi kesalahan pada server", pesan yang menyuruh orang mencari
// kerusakan server yang tidak pernah ada. Keadaan buntu yang eksplisit
// menghasilkan `1=0`: nol baris, tanpa galat, tanpa menebak-nebak nilai apa yang
// mustahil.
type Limit struct {
	ChapterID string
	Terbatas  bool
	Buntu     bool
}

// SQL mengembalikan klausa yang menegakkan batas ini, beserta argumennya.
//
// Satu tempat yang menyusunnya, supaya setiap repository menegakkan hal yang
// persis sama. Nomor placeholder diserahkan pemanggil karena tiap query punya
// urutan argumennya sendiri.
func (l Limit) SQL(kolom string, nomor int) (klausa string, arg any, pakaiArg bool) {
	switch {
	case l.Buntu:
		return "1=0", nil, false
	case l.Terbatas:
		return fmt.Sprintf("%s = $%d", kolom, nomor), l.ChapterID, true
	}
	return "", nil, false
}

// WithChapter membatasi seluruh query di bawah context ini ke satu chapter.
func WithChapter(ctx context.Context, chapterID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, chapterScope{id: chapterID, terbatas: true})
}

// WithoutLimit menandai context ini berjangkauan nasional — admin dan user.
//
// HARUS dipanggil eksplisit. Ketiadaannya bukan berarti "tanpa batas"; lihat
// Chapter di bawah.
func WithoutLimit(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, chapterScope{terbatas: false})
}

// Chapter mengembalikan batas chapter yang berlaku, untuk dipasang pada query.
//
// GAGAL TERTUTUP, dan inilah inti keamanan paket ini. Bila context sama sekali
// tidak membawa lingkup — middleware terlewat, handler terpasang di rantai yang
// salah, goroutine dijalankan dengan context.Background() — fungsi ini
// mengembalikan lingkup MUSTAHIL, bukan "tanpa batas". Query yang memakainya
// mengembalikan nol baris, dan layarnya kosong: keras, langsung terlihat.
//
// Arah kebalikannya — menganggap ketiadaan lingkup sebagai "boleh semua" —
// membuat kesalahan pemasangan yang sama membocorkan SELURUH chapter dengan
// status 200. Tidak ada yang merah, tidak ada yang mengeluh, dan kebocorannya
// hanya ketahuan kalau ada yang kebetulan memperhatikan data milik orang lain
// di layarnya sendiri.
func Chapter(ctx context.Context) Limit {
	s, ok := ctx.Value(ctxKey{}).(chapterScope)
	if !ok {
		return Limit{Buntu: true}
	}
	if !s.terbatas {
		return Limit{}
	}
	return Limit{ChapterID: s.id, Terbatas: true}
}
