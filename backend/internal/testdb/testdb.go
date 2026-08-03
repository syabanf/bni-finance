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
