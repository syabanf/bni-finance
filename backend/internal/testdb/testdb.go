// Package testdb menserialkan akses integration test ke database bersama.
//
// Tiga paket (api, sync, paperid) masing-masing mem-TRUNCATE tabel bisnis di
// awal tesnya. Makefile menjalankan mereka dengan `-p 1`, tetapi Makefile
// bukan satu-satunya pintu: `go test ./...` dengan TEST_DATABASE_URL masih
// terpasang di shell menjalankan ketiganya PARALEL terhadap satu database —
// dan mereka saling menghapus data di tengah run yang lain.
//
// Kegagalannya acak karena bergantung interleaving: dua run berurutan dengan
// perintah yang sama bisa menghasilkan "ketiga paket gagal bersamaan" lalu
// "semua hijau". Itu persis tanda tangan satu kegagalan yang selama lima run
// tidak pernah bisa direproduksi — dan lognya sudah tertimpa sebelum terbaca.
//
// Kuncinya di database, bukan di flag, supaya SEMUA cara invokasi terlindungi:
// advisory lock sesi pada satu koneksi yang dipegang sepanjang paket berjalan.
// Proses tes kedua tidak gagal — ia MENUNGGU gilirannya.
package testdb

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// lockKey satu untuk semua paket integration — nilainya bebas asal sama.
// hashtext('bni-integration-tests') dihitung Postgres supaya kuncinya terbaca
// di pg_locks tanpa perlu menebak angka.
const lockQuery = "SELECT pg_advisory_lock(hashtext('bni-integration-tests'))"

// Serialize memegang advisory lock sesi sampai tes (beserta seluruh subtesnya)
// selesai. Panggil SEBELUM TRUNCATE pertama.
//
// TUTUP POOL LEWAT t.Cleanup, JANGAN defer. Defer berjalan lebih dulu daripada
// Cleanup, sedangkan koneksi pemegang kunci baru dilepas di Cleanup — sehingga
// `defer pool.Close()` menunggu koneksi yang justru akan dilepas sesudahnya.
// Tesnya lalu MENGGANTUNG sampai timeout alih-alih gagal, dan jejaknya menunjuk
// ke Pool.Close, bukan ke penyebabnya. Cleanup berjalan LIFO, jadi yang
// didaftarkan Serialize dijalankan lebih dulu:
//
//	t.Cleanup(pool.Close)
//	testdb.Serialize(t, pool)
//
// Lock sesi menempel pada koneksi, dan koneksi dari pool bisa dikembalikan —
// karena itu satu koneksi diambil khusus dan ditahan sampai Cleanup. Menutup
// koneksi melepas kuncinya, jadi tidak ada jalur yang meninggalkan kunci
// menggantung, bahkan bila tes panik.
func Serialize(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()

	// Batas waktu longgar: paket lain yang sedang memegang kunci bisa sedang
	// menunggu Paper.id staging, yang terukur sampai 52 detik per panggilan.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("testdb: ambil koneksi untuk kunci: %v", err)
	}
	if _, err := conn.Exec(ctx, lockQuery); err != nil {
		conn.Release()
		t.Fatalf("testdb: ambil advisory lock: %v", err)
	}
	t.Cleanup(func() {
		// Unlock eksplisit lalu lepaskan; menutup pool (Cleanup pemanggil)
		// tetap menjadi jaring pengaman bila unlock gagal.
		_, _ = conn.Exec(context.Background(),
			"SELECT pg_advisory_unlock(hashtext('bni-integration-tests'))")
		conn.Release()
	})
}

// Periode mendaftarkan periode konfirmasi renewal untuk tes.
//
// renewal_requests.period menunjuk ke renewal_periods, jadi tes yang mencatat
// permintaan konfirmasi harus mendaftarkan periodenya lebih dulu. Rentang
// tanggalnya tidak berarti apa-apa bagi tes.
//
// Periode uji yang tidak lagi dipakai permintaan mana pun dihapus saat tes
// selesai, dan sekali lagi pada panggilan pertama tiap proses tes. Yang kedua
// menangkap sisa run sebelumnya: tes integrasi API mengosongkan permintaan
// lewat TRUNCATE di run berikutnya, bukan di cleanup-nya sendiri.
func Periode(t testing.TB, pool *pgxpool.Pool, ids ...string) {
	t.Helper()
	sisaRunLalu.Do(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM renewal_periods p
			WHERE p.name LIKE 'uji %' AND NOT EXISTS (SELECT 1 FROM renewal_requests r WHERE r.period = p.id)`)
	})
	for _, id := range ids {
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO renewal_periods (id, name, start_date, end_date)
			VALUES ($1, 'uji ' || $1, DATE '2000-01-01', DATE '2000-01-01')
			ON CONFLICT (id) DO NOTHING`, id); err != nil {
			t.Fatalf("testdb: daftarkan periode %s: %v", id, err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM renewal_periods p
			WHERE p.id = ANY($1) AND p.name LIKE 'uji %'
			  AND NOT EXISTS (SELECT 1 FROM renewal_requests r WHERE r.period = p.id)`, ids)
	})
}

var sisaRunLalu sync.Once
