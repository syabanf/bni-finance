package testdb_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// db/reset.sql tidak boleh menghapus akun nasional.
//
// Ini menjaga kegagalan yang sudah benar-benar terjadi, bukan yang dibayangkan.
// Versi sebelumnya memakai `truncate … cascade`; karena users.chapter_id
// menunjuk ke chapters, CASCADE ikut mengosongkan `users`. Postgres bahkan
// mengumumkannya — "truncate cascades to table users" — tapi tidak ada yang
// membaca NOTICE di tengah ratusan baris keluaran.
//
// Akibatnya: perintah untuk MENYIAPKAN lingkungan lokal justru mengunci orang
// di luar aplikasinya, sementara pesannya berkata "users tidak disentuh". Dan
// karena kerusakannya terlihat seperti "aplikasinya rusak", bukan "resetnya
// menghapus akun", ia bisa berulang berkali-kali tanpa pernah dikenali.
//
// Tes ini menjalankan BERKAS SQL YANG SAMA dengan `make db-reset`. Menyalin
// SQL-nya ke sini akan membuat salinan itu menyimpang, dan tesnya lalu menjaga
// sesuatu yang tidak lagi dijalankan siapa pun.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
func TestResetSQLTidakMenghapusAkunNasional(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL tidak diset — integration test dilewati")
	}
	nama := url[strings.LastIndex(url, "/")+1:]
	if !strings.Contains(nama, "test") && !strings.Contains(nama, "dev") {
		t.Fatalf("menolak berjalan atas basis data %q — namanya tidak mengandung test/dev", nama)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("sambung basis data: %v", err)
	}
	// t.Cleanup, BUKAN defer. Defer berjalan sebelum Cleanup, sedangkan
	// Serialize melepas koneksi pemegang kuncinya di Cleanup — jadi
	// `defer pool.Close()` menunggu koneksi yang baru akan dilepas sesudahnya,
	// dan tesnya menggantung sampai timeout alih-alih gagal. Cleanup berjalan
	// LIFO, sehingga yang didaftarkan Serialize dijalankan lebih dulu.
	t.Cleanup(pool.Close)
	testdb.Serialize(t, pool)

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("siapkan data: %v", err)
		}
	}
	exec(`INSERT INTO chapters (id, name, display_name)
	      VALUES ('ch-reset','reset','Chapter Reset') ON CONFLICT (id) DO NOTHING`)
	exec(`INSERT INTO users (id, email, password_hash, name, role, chapter_id, created_at)
	      VALUES (gen_random_uuid(),'nasional-reset@uji.invalid','x','Nasional','admin',NULL,now()),
	             (gen_random_uuid(),'berlingkup-reset@uji.invalid','x','ST','st','ch-reset',now())
	      ON CONFLICT (email) DO NOTHING`)
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM users WHERE email LIKE '%-reset@uji.invalid'`)
		_, _ = pool.Exec(c, `DELETE FROM chapters WHERE id = 'ch-reset'`)
	})

	jalankan := func(nama string) {
		t.Helper()
		sql, err := os.ReadFile(filepath.Join("..", "..", "..", "db", nama))
		if err != nil {
			t.Fatalf("baca db/%s: %v", nama, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("jalankan db/%s: %v", nama, err)
		}
	}

	// `make db-reset` menjalankan reset.sql LALU init.sql. Tes ini melakukan
	// keduanya, dengan pemeriksaan di antaranya.
	//
	// Bukan sekadar demi kesetiaan: reset.sql mengosongkan fee_settings, dan
	// init.sql-lah yang mengisinya kembali. Berhenti di reset.sql meninggalkan
	// basis data bersama ini tanpa pengaturan biaya — dan tes paket lain lalu
	// gagal dengan 404 yang tidak menyebut-nyebut berkas ini sama sekali.
	jalankan("reset.sql")
	t.Cleanup(func() { jalankan("init.sql") })

	var nasional, berlingkup int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE email = 'nasional-reset@uji.invalid'),
		       count(*) FILTER (WHERE email = 'berlingkup-reset@uji.invalid')
		FROM users`).Scan(&nasional, &berlingkup); err != nil {
		t.Fatalf("hitung user: %v", err)
	}

	if nasional != 1 {
		t.Errorf("akun nasional hilang setelah reset — CASCADE menyeret tabel users?")
	}
	// Yang berlingkup memang harus ikut hilang: chapter yang mengikatnya sudah
	// tidak ada, dan mengosongkan chapter_id-nya justru akan mengubah ST menjadi
	// pengguna nasional yang melihat seluruh chapter.
	if berlingkup != 0 {
		t.Errorf("akun berlingkup masih ada padahal chapternya sudah dihapus")
	}

	var chapters int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM chapters`).Scan(&chapters); err != nil {
		t.Fatalf("hitung chapter: %v", err)
	}
	if chapters != 0 {
		t.Errorf("chapters = %d setelah reset, seharusnya kosong", chapters)
	}
}
