package renewal

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Daftar permintaan membawa NAMA MC yang ditugaskan, bukan hanya id akunnya.
//
// Halaman Konfirmasi Renewal dulu mencari nama itu dari daftar akun, yang
// hanya bisa dibaca admin. Bila daftar itu tidak tersedia, kolom Ditugaskan
// menampilkan id akun apa adanya.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_test
func TestDaftarMembawaNamaMC(t *testing.T) {
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
	const (
		ch    = "ch-uji-nama-mc"
		email = "mc.uji-nama@contoh.invalid"
	)
	bersih := func() {
		_, _ = pool.Exec(ctx, "DELETE FROM renewal_requests WHERE chapter_id = $1", ch)
		_, _ = pool.Exec(ctx, "DELETE FROM members WHERE chapter_id = $1", ch)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
		_, _ = pool.Exec(ctx, "DELETE FROM chapters WHERE id = $1", ch)
	}
	bersih()
	t.Cleanup(bersih)

	if _, err := pool.Exec(ctx,
		`INSERT INTO chapters (id, name, display_name) VALUES ($1, 'ujinamamc', 'Uji Nama MC')`, ch); err != nil {
		t.Fatalf("siapkan chapter: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO members (id, chapter_id, name, status, joined_date) VALUES
		('mem-uji-mc-ditugaskan', $1, 'Anggota Ditugaskan', 'active', CURRENT_DATE),
		('mem-uji-mc-bebas',      $1, 'Anggota Bebas',      'active', CURRENT_DATE)`, ch); err != nil {
		t.Fatalf("siapkan member: %v", err)
	}
	var mc string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name, role, chapter_id)
		VALUES ($1, 'tidak-dipakai', 'Mira Ujinama', 'mc', $2) RETURNING id`, email, ch).Scan(&mc); err != nil {
		t.Fatalf("siapkan akun MC: %v", err)
	}

	repo := NewRepository(pool)
	if _, err := repo.Create(ctx, []string{"mem-uji-mc-ditugaskan"}, "2031-S1", "uji", &mc); err != nil {
		t.Fatalf("minta konfirmasi dengan MC: %v", err)
	}
	if _, err := repo.Create(ctx, []string{"mem-uji-mc-bebas"}, "2031-S1", "uji", nil); err != nil {
		t.Fatalf("minta konfirmasi tanpa MC: %v", err)
	}

	items, _, err := repo.List(ctx, domain.RenewalFilter{ChapterID: ch, Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("dapat %d permintaan, ingin 2", len(items))
	}
	for _, r := range items {
		switch r.MemberID {
		case "mem-uji-mc-ditugaskan":
			if r.AssignedMCName == nil || *r.AssignedMCName != "Mira Ujinama" {
				t.Errorf("nama MC = %v, ingin Mira Ujinama", r.AssignedMCName)
			}
		case "mem-uji-mc-bebas":
			// Tanpa penugasan, permintaannya milik semua MC chapter itu.
			if r.AssignedMC != nil || r.AssignedMCName != nil {
				t.Errorf("permintaan tanpa MC membawa %v / %v", r.AssignedMC, r.AssignedMCName)
			}
		}
	}
}
