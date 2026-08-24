package invoice

import (
	"context"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Batas chapter tidak punya lapisan lain yang menegakkannya.
//
// Postgres tidak bisa: backend menyambung sebagai SATU peran tepercaya, jadi
// basis data melihat satu identitas saja dan tidak punya apa pun untuk dipakai
// sebagai kunci kebijakan per-baris. Tes di berkas ini karena itu bukan
// pelengkap — ia satu-satunya bukti bahwa batasnya benar-benar ada.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev

// duaChapter MEMBUAT dua chapter beserta membernya, khusus untuk tes ini.
//
// Versi pertama membacanya dari data contoh, dan itu keliru dengan cara yang
// berbahaya: bila datanya kurang, tes memanggil t.Skip. Integration test lain
// men-TRUNCATE tabel, jadi urutan run yang biasa saja sudah cukup membuat
// seluruh tes batas akses ini melewati dirinya sendiri — dan `go test`
// melaporkannya "ok".
//
// Terbukti: sebuah sabotase yang membuang filter chapter dari List tetap
// menghasilkan "ok", karena tesnya tidak pernah berjalan. Tes keamanan yang
// bisa hijau tanpa menguji apa pun lebih buruk daripada tidak ada tes, sebab ia
// membuat orang berhenti memeriksa.
//
// Dengan menyediakan dunianya sendiri, tes ini tidak bisa lagi melewati diri
// sendiri, dan tidak bergantung pada urutan run maupun isi basis data.
func duaChapter(t *testing.T, repo *Repository) (a, b string, memberA, memberB string) {
	t.Helper()
	ctx := scope.WithoutLimit(context.Background())

	a, b = "ch-uji-lingkup-a", "ch-uji-lingkup-b"
	memberA, memberB = "mem-uji-lingkup-a", "mem-uji-lingkup-b"

	for i, ch := range []string{a, b} {
		if _, err := repo.db.Exec(ctx, `
			INSERT INTO chapters (id, name, display_name, area_name, city_name)
			VALUES ($1,$2,$3,'Uji','Uji') ON CONFLICT (id) DO NOTHING`,
			ch, "Uji"+string(rune('A'+i)), "Uji "+string(rune('A'+i))); err != nil {
			t.Fatalf("siapkan chapter %s: %v", ch, err)
		}
	}
	for i, m := range []string{memberA, memberB} {
		ch := []string{a, b}[i]
		if _, err := repo.db.Exec(ctx, `
			INSERT INTO members (id, chapter_id, name, email, phone, status)
			VALUES ($1,$2,$3,'uji@contoh.invalid','080000000000','active')
			ON CONFLICT (id) DO NOTHING`, m, ch, "Member Uji "+string(rune('A'+i))); err != nil {
			t.Fatalf("siapkan member %s: %v", m, err)
		}
	}

	t.Cleanup(func() {
		bersih := scope.WithoutLimit(context.Background())
		_, _ = repo.db.Exec(bersih,
			`DELETE FROM invoice_audit_log WHERE invoice_id IN
			 (SELECT id FROM invoices WHERE member_id = ANY($1))`, []string{memberA, memberB})
		_, _ = repo.db.Exec(bersih, `DELETE FROM invoices WHERE member_id = ANY($1)`,
			[]string{memberA, memberB})
		_, _ = repo.db.Exec(bersih, `DELETE FROM members WHERE id = ANY($1)`,
			[]string{memberA, memberB})
		_, _ = repo.db.Exec(bersih, `DELETE FROM chapters WHERE id = ANY($1)`, []string{a, b})
	})
	return a, b, memberA, memberB
}

// ST chapter A tidak boleh melihat invoice chapter B — tidak lewat daftar,
// tidak lewat id langsung.
func TestSTHanyaMelihatChapternyaSendiri(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB, memA, memB := duaChapter(t, repo)
	nasional := scope.WithoutLimit(context.Background())

	invA, err := repo.Create(nasional, contohInput(memA, chA, 1_500_000), "", "IDR")
	if err != nil {
		t.Fatalf("buat invoice chapter A: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(nasional, invA.ID) })

	invB, err := repo.Create(nasional, contohInput(memB, chB, 1_500_000), "", "IDR")
	if err != nil {
		t.Fatalf("buat invoice chapter B: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(nasional, invB.ID) })

	stA := scope.WithChapter(context.Background(), chA)

	// --- daftar --------------------------------------------------------------
	items, _, err := repo.List(stA, domain.InvoiceFilter{Limit: 200})
	if err != nil {
		t.Fatalf("list sebagai ST: %v", err)
	}
	for _, inv := range items {
		if inv.ChapterID != chA {
			t.Errorf("ST chapter %s melihat invoice %s milik chapter %s", chA, inv.Number, inv.ChapterID)
		}
	}

	// --- ambil langsung lewat id --------------------------------------------
	// Menyembunyikannya dari daftar saja tidak cukup: id invoice muncul di URL,
	// dan menebaknya harus tetap buntu.
	if _, err := repo.GetByID(stA, invB.ID); err == nil {
		t.Error("ST chapter A bisa membuka invoice chapter B lewat id langsung")
	} else if httpx.StatusOf(err) != 404 {
		// 404 dan bukan 403: jawaban "tidak boleh" mengonfirmasi invoicenya ADA.
		t.Errorf("status = %d, mau 404 supaya keberadaan invoicenya tidak bocor", httpx.StatusOf(err))
	}

	// Invoice chapternya sendiri tetap terbaca — pembatasnya harus memotong
	// tepat, bukan memblokir semuanya.
	if _, err := repo.GetByID(stA, invA.ID); err != nil {
		t.Errorf("ST tidak bisa membuka invoice chapternya sendiri: %v", err)
	}
}

// Filter chapter lain yang dikirim ST harus mengembalikan KOSONG, bukan
// diam-diam dialihkan ke chapternya sendiri.
func TestFilterChapterLainTidakDialihkan(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB, memA, _ := duaChapter(t, repo)
	nasional := scope.WithoutLimit(context.Background())

	inv, err := repo.Create(nasional, contohInput(memA, chA, 1_500_000), "", "IDR")
	if err != nil {
		t.Fatalf("buat invoice: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(nasional, inv.ID) })

	stA := scope.WithChapter(context.Background(), chA)
	items, total, err := repo.List(stA, domain.InvoiceFilter{ChapterID: chB, Limit: 200})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 || total != 0 {
		t.Errorf("ST menyaring chapter %s dan mendapat %d baris (total %d) — "+
			"filternya dialihkan, jadi jawabannya salah tanpa ada yang tahu",
			chB, len(items), total)
	}
}

// ST tidak boleh menerbitkan invoice untuk chapter lain.
//
// Ditolak SEBELUM basis data disentuh: nomor invoice yang sudah terbakar tidak
// bisa dikembalikan, dan setiap penerbitan ke Paper.id memakainya permanen.
func TestSTTidakBisaMenerbitkanUntukChapterLain(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	chA, chB, _, memB := duaChapter(t, repo)
	stA := scope.WithChapter(context.Background(), chA)

	inv, err := repo.Create(stA, contohInput(memB, chB, 1_500_000), "", "IDR")
	if err == nil {
		_ = repo.Delete(scope.WithoutLimit(context.Background()), inv.ID)
		t.Fatalf("ST chapter %s berhasil menerbitkan invoice untuk chapter %s", chA, chB)
	}
	if got := httpx.StatusOf(err); got != 403 {
		t.Errorf("status = %d, mau 403: %v", got, err)
	}
}

// Context tanpa lingkup sama sekali harus buntu, bukan terbuka.
//
// Ini jaring terakhirnya: jalur yang lupa memasang lingkup — worker latar,
// perintah CLI, handler di rantai yang salah — tidak boleh berakhir sebagai
// akses penuh ke seluruh chapter.
func TestContextTanpaLingkupTidakMengembalikanApaPun(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)

	items, total, err := repo.List(context.Background(), domain.InvoiceFilter{Limit: 200})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 || total != 0 {
		t.Errorf("context polos mengembalikan %d invoice (total %d) — seharusnya nol", len(items), total)
	}
}
