package member

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

const chapterUji = "ch-uji-saring-member"

// Lima member dengan status dan kelengkapan kontak yang berbeda, di satu
// chapter khusus tes supaya hitungannya tidak tercampur data lain.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_test
func siapkan(t *testing.T) (*Repository, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL tidak diset — integration test dilewati")
	}
	if nama := url[strings.LastIndex(url, "/")+1:]; !strings.Contains(nama, "test") && !strings.Contains(nama, "dev") {
		t.Fatalf("menolak berjalan atas basis data %q — namanya tidak mengandung test/dev", nama)
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("sambung basis data: %v", err)
	}
	t.Cleanup(pool.Close)
	testdb.Serialize(t, pool)

	ctx := scope.WithoutLimit(context.Background())
	bersih := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM members WHERE chapter_id = $1`, chapterUji)
		_, _ = pool.Exec(ctx, `DELETE FROM chapters WHERE id = $1`, chapterUji)
	}
	bersih()
	t.Cleanup(bersih)

	if _, err := pool.Exec(ctx, `
		INSERT INTO chapters (id, name, display_name) VALUES ($1, 'Uji Saring', 'Uji Saring')`, chapterUji); err != nil {
		t.Fatalf("siapkan chapter: %v", err)
	}
	baris := []struct {
		id, status   string
		email, phone *string
		jatuhTempo   *string
	}{
		{"mem-saring-1", "active", teks("a@contoh.invalid"), teks("0811"), teks("2027-01-10")},
		{"mem-saring-2", "active", nil, teks("0812"), nil},
		{"mem-saring-3", "inactive", teks("c@contoh.invalid"), teks("   "), teks("2026-03-01")},
		{"mem-saring-4", "pending", nil, nil, nil},
		{"mem-saring-5", "visitor", teks("e@contoh.invalid"), nil, nil},
	}
	for _, b := range baris {
		if _, err := pool.Exec(ctx, `
			INSERT INTO members (id, chapter_id, name, email, phone, status, renewal_date)
			VALUES ($1, $2, $3, $4, $5, $6::member_status, $7::date)`,
			b.id, chapterUji, "Uji "+b.id, b.email, b.phone, b.status, b.jatuhTempo); err != nil {
			t.Fatalf("siapkan %s: %v", b.id, err)
		}
	}
	return NewRepository(pool), ctx
}

func teks(s string) *string { return &s }

func idTerurut(t *testing.T, repo *Repository, ctx context.Context, f domain.MemberFilter) []string {
	t.Helper()
	f.ChapterID, f.Limit = chapterUji, 50
	items, total, err := repo.List(ctx, f)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != len(items) {
		t.Errorf("total %d tidak sama dengan jumlah baris %d", total, len(items))
	}
	var ids []string
	for _, m := range items {
		ids = append(ids, strings.TrimPrefix(m.ID, "mem-saring-"))
	}
	sort.Strings(ids)
	return ids
}

func TestSaringanMember(t *testing.T) {
	repo, ctx := siapkan(t)

	kasus := []struct {
		nama  string
		f     domain.MemberFilter
		ingin string
	}{
		{"kelompok member mencakup yang inactive", domain.MemberFilter{Kelompok: "member"}, "1,2,3"},
		{"kelompok visitor mencakup pending", domain.MemberFilter{Kelompok: "visitor"}, "4,5"},
		// Nomor yang hanya berisi spasi dihitung kosong.
		{"tanpa nomor HP", domain.MemberFilter{Kontak: "no-phone"}, "3,4,5"},
		{"tanpa email", domain.MemberFilter{Kontak: "no-email"}, "2,4"},
		{"tanpa keduanya", domain.MemberFilter{Kontak: "no-contact"}, "4"},
		{"punya jatuh tempo", domain.MemberFilter{PunyaJatuhTempo: true}, "1,3"},
		{"per id", domain.MemberFilter{IDs: []string{"mem-saring-2", "mem-saring-5", "tidak-ada"}}, "2,5"},
		{"cari lewat id", domain.MemberFilter{Search: "saring-3"}, "3"},
		{"gabungan", domain.MemberFilter{Kelompok: "member", Kontak: "no-phone"}, "3"},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			if got := strings.Join(idTerurut(t, repo, ctx, k.f), ","); got != k.ingin {
				t.Errorf("dapat %q, ingin %q", got, k.ingin)
			}
		})
	}
}

// Ringkasan mengabaikan saringan status, daftarnya tidak. Kalau keduanya
// sama-sama menyaring, kartu status selain yang dipilih selalu nol dan tidak
// bisa lagi dipakai berpindah status.
func TestRingkasanMemberMengabaikanSaringanStatus(t *testing.T) {
	repo, ctx := siapkan(t)
	f := domain.MemberFilter{ChapterID: chapterUji, Status: "active", Limit: 50}

	items, _, err := repo.List(ctx, f)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("daftar berstatus active = %d baris, ingin 2", len(items))
	}

	ringkas, err := repo.Summary(ctx, f)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	ingin := map[string]int{"active": 2, "inactive": 1, "pending": 1, "visitor": 1}
	for status, n := range ingin {
		if ringkas[status] != n {
			t.Errorf("ringkasan[%s] = %d, ingin %d", status, ringkas[status], n)
		}
	}

	// Saringan lain tetap berlaku pada ringkasan.
	f.Kontak = "no-email"
	ringkas, err = repo.Summary(ctx, f)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if ringkas["active"] != 1 || ringkas["pending"] != 1 || ringkas["visitor"] != 0 {
		t.Errorf("ringkasan tanpa email = %v, ingin active 1, pending 1, visitor 0", ringkas)
	}
}
