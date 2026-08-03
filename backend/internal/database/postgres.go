package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultMaxConns adalah ukuran pool bila DB_MAX_CONNS tidak diisi.
//
// Angka ini sebelumnya dipatok mati dan tidak bisa disetel. Beban 1000
// permintaan paralel menunjukkan 433 di antaranya harus MENUNGGU koneksi bebas
// — p50 16 ms tetapi p95 129 ms, dan selisih itu antre di pool, bukan query
// yang lambat. `db_pool_acquire_empty_total` di /metrics adalah angka yang
// menunjukkannya.
//
// Bawaannya tetap 10 supaya perilaku tidak berubah diam-diam. Nilai yang tepat
// tidak bisa ditebak dari sini: ia bergantung pada `max_connections` Postgres
// produksi dibagi jumlah instans backend. Yang bisa dilakukan kode adalah
// menyediakan tuasnya dan menunjukkan kapan perlu diputar.
const DefaultMaxConns = 10

// NewPool opens a pgx pool and verifies the connection so startup fails fast
// on a bad DATABASE_URL rather than on the first request.
//
// maxConns <= 0 memakai DefaultMaxConns.
func NewPool(ctx context.Context, url string, maxConns int) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL tidak valid: %w", err)
	}
	if maxConns <= 0 {
		maxConns = DefaultMaxConns
	}
	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("gagal terhubung ke database: %w", err)
	}
	return pool, nil
}
