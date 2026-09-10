package renewal

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// VISITOR TIDAK BOLEH MENERIMA PERMINTAAN RENEWAL.
//
// Dua query yang MEMANEN member untuk renewal menyaring `status = 'active'`,
// jadi visitor terlewat dengan sendirinya di sana. Jalur ini berbeda: daftar
// id-nya dipilih operator lewat "pilih banyak", tanpa satu pun penyaring
// status. Menekan "pilih semua" pada satu chapter lalu mengirim sudah cukup
// untuk meminta perpanjangan keanggotaan dari orang yang belum menjadi
// anggota — dan pesan itu berangkat atas nama BNI.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
func TestVisitorDilewatiSaatMintaRenewal(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL tidak diset — integration test dilewati")
	}
	nama := url[strings.LastIndex(url, "/")+1:]
	if !strings.Contains(nama, "test") && !strings.Contains(nama, "dev") {
		t.Fatalf("menolak berjalan atas basis data %q — namanya tidak mengandung test/dev", nama)
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("sambung basis data: %v", err)
	}
	// t.Cleanup, BUKAN defer: defer berjalan SEBELUM cleanup, dan Serialize
	// melepas koneksinya di cleanup — menutup pool lebih dulu membuatnya
	// menggantung sampai timeout.
	t.Cleanup(pool.Close)
	testdb.Serialize(t, pool)

	ctx := context.Background()
	const ch = "ch-uji-visitor"
	if _, err := pool.Exec(ctx,
		`INSERT INTO chapters (id, name, display_name, area_name, city_name)
		 VALUES ($1,'ujivisitor','Uji Visitor','Uji','Uji') ON CONFLICT (id) DO NOTHING`, ch); err != nil {
		t.Fatalf("siapkan chapter: %v", err)
	}
	orang := map[string]string{
		"mem-uji-anggota": "active",
		"mem-uji-tamu":    "visitor",
	}
	for id, st := range orang {
		if _, err := pool.Exec(ctx,
			`INSERT INTO members (id, chapter_id, name, status, joined_date)
			 VALUES ($1,$2,$3,$4::member_status, CURRENT_DATE)
			 ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status`,
			id, ch, "Uji "+id, st); err != nil {
			t.Fatalf("siapkan member %s: %v", id, err)
		}
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, "DELETE FROM renewal_requests WHERE chapter_id = $1", ch)
		_, _ = pool.Exec(c, "DELETE FROM members WHERE chapter_id = $1", ch)
		_, _ = pool.Exec(c, "DELETE FROM chapters WHERE id = $1", ch)
	})

	repo := NewRepository(pool)
	// Lingkup admin: berjangkauan nasional. HARUS eksplisit — context tanpa
	// lingkup dianggap MUSTAHIL oleh scope.Chapter, dan Create akan menolak
	// dengan 403 alih-alih menguji apa yang dimaksud tes ini.
	ctx = scope.WithoutLimit(ctx)

	dibuat, dilewati, visitor, err := repo.Create(ctx,
		[]string{"mem-uji-anggota", "mem-uji-tamu"}, "2027-H1", "uji", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if dibuat != 1 {
		t.Errorf("dibuat = %d, seharusnya 1 (hanya yang berstatus active)", dibuat)
	}
	if visitor != 1 {
		t.Errorf("visitor = %d, seharusnya 1", visitor)
	}
	if dilewati != 0 {
		t.Errorf("dilewati = %d, seharusnya 0 — tamu tidak boleh disamakan dengan"+
			" permintaan yang sudah ada", dilewati)
	}

	// Yang menentukan bukan angkanya, tapi barisnya: tidak boleh ada satu pun
	// permintaan atas nama tamu itu di basis data.
	var ada int
	if err := pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM renewal_requests WHERE member_id = 'mem-uji-tamu'").Scan(&ada); err != nil {
		t.Fatalf("hitung permintaan: %v", err)
	}
	if ada != 0 {
		t.Errorf("ada %d permintaan renewal atas nama visitor", ada)
	}
}
