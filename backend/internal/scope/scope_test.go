package scope_test

import (
	"context"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/scope"
)

// Context tanpa lingkup harus berarti TIDAK BOLEH MELIHAT APA PUN.
//
// Ini satu-satunya sifat yang benar-benar penting di paket ini. Kalau ketiadaan
// lingkup diperlakukan sebagai "boleh semua", maka setiap jalur yang lupa
// memasang lingkup — middleware terlewat, handler di rantai yang salah,
// goroutine dengan context.Background() — membocorkan SELURUH chapter dengan
// status 200. Tidak ada yang merah, dan kebocorannya hanya ketahuan kalau ada
// yang kebetulan memperhatikan data orang lain di layarnya.
func TestContextPolosTidakBolehMelihatApaPun(t *testing.T) {
	lim := scope.Chapter(context.Background())
	if !lim.Buntu {
		t.Fatal("context tanpa lingkup tidak buntu — ini membocorkan seluruh chapter")
	}
	klausa, _, pakaiArg := lim.SQL("chapter_id", 1)
	if klausa != "1=0" || pakaiArg {
		t.Errorf("klausa buntu = %q (pakaiArg=%v), mau \"1=0\" tanpa argumen", klausa, pakaiArg)
	}
}

// Klausa buntu harus berupa SQL yang sah dan tidak memakai nilai sentinel.
//
// Versi pertama memakai id sentinel berisi byte nol, dengan harapan tidak cocok
// dengan apa pun. Postgres menolaknya mentah-mentah — "invalid byte sequence for
// encoding UTF8: 0x00" — jadi jalur tanpa lingkup berakhir 500, bukan nol baris.
// Tes ini mengunci pelajaran itu: keadaan buntu tidak boleh lagi diwakili nilai.
func TestKlausaBuntuTidakMemakaiNilaiSentinel(t *testing.T) {
	lim := scope.Chapter(context.Background())
	if lim.ChapterID != "" {
		t.Errorf("keadaan buntu masih membawa nilai %q — nilai apa pun bisa ditolak basis data", lim.ChapterID)
	}
}

func TestWithChapterMembatasi(t *testing.T) {
	lim := scope.Chapter(scope.WithChapter(context.Background(), "ch-garuda"))
	if !lim.Terbatas || lim.Buntu || lim.ChapterID != "ch-garuda" {
		t.Errorf("dapat %+v, mau terbatas ke ch-garuda", lim)
	}
	klausa, arg, pakai := lim.SQL("chapter_id", 3)
	if klausa != "chapter_id = $3" || arg != "ch-garuda" || !pakai {
		t.Errorf("SQL = (%q, %v, %v)", klausa, arg, pakai)
	}
}

// WithoutLimit harus DINYATAKAN, dan itu bedanya dengan context polos: yang satu
// pernyataan sadar bahwa pemanggilnya nasional, yang lain hanya kelalaian.
func TestWithoutLimitMembukaBatas(t *testing.T) {
	lim := scope.Chapter(scope.WithoutLimit(context.Background()))
	if lim.Terbatas || lim.Buntu {
		t.Errorf("WithoutLimit masih membatasi: %+v", lim)
	}
	if klausa, _, _ := lim.SQL("chapter_id", 1); klausa != "" {
		t.Errorf("lingkup nasional menghasilkan klausa %q, mau kosong", klausa)
	}
}

// Lingkup yang lebih dalam menang, supaya satu permintaan tidak bisa menaikkan
// haknya sendiri di tengah jalan.
func TestLingkupTerdalamYangBerlaku(t *testing.T) {
	ctx := scope.WithoutLimit(context.Background())
	ctx = scope.WithChapter(ctx, "ch-mahakam")
	lim := scope.Chapter(ctx)
	if !lim.Terbatas || lim.ChapterID != "ch-mahakam" {
		t.Errorf("dapat %+v, mau terbatas ke ch-mahakam", lim)
	}
}
