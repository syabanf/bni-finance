// Package cache adalah Redis yang dibagi seluruh replika API.
//
// Tiga kegunaan, dan ketiganya baru benar bila disimpan di luar proses:
//
//	cache baca       daftar chapter dan biaya keanggotaan
//	pembatas login   hitungan gagal per email, supaya batasnya tidak
//	                 berlipat dengan jumlah replika
//	kunci sekali     tugas berkala yang cukup dijalankan satu replika
//
// *Redis bernilai nil berarti Redis tidak dikonfigurasi. Setiap metode tetap
// aman dipanggil: baca selalu meleset, tulis tidak melakukan apa pun, dan kunci
// selalu didapat. Itu perilaku satu instance tanpa Redis, sama seperti sebelum
// paket ini ada.
//
// Redis yang mati di tengah jalan diperlakukan sama: galatnya dicatat lalu
// pemanggil membaca langsung dari Postgres. Cache yang hilang membuat aplikasi
// lebih lambat; cache yang menggagalkan permintaan membuatnya mati.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// batasWaktu membatasi setiap panggilan Redis. Cache yang menunggu lebih lama
// dari query Postgres yang digantikannya tidak ada gunanya.
const batasWaktu = 300 * time.Millisecond

// awalan memisahkan kunci aplikasi ini bila Redis-nya dipakai bersama.
const awalan = "bni:"

// jeda adalah lama Redis dilewati setelah satu kegagalan. Tanpa jeda, setiap
// permintaan selama Redis mati menunggu batasWaktu lebih dulu dan menulis satu
// baris peringatan; diukur di Docker, itu menambah sekitar 280 ms per
// permintaan dan membanjiri log.
const jeda = 5 * time.Second

// errJeda dikembalikan ke pemanggil yang perlu tahu Redis sedang dilewati.
var errJeda = errors.New("redis sedang dilewati setelah gagal")

type Redis struct {
	rdb *redis.Client
	log *slog.Logger
	// lewatiSampai menyimpan UnixNano sampai kapan Redis tidak dipanggil.
	lewatiSampai atomic.Int64
}

// siap melaporkan Redis boleh dipanggil: dikonfigurasi dan tidak sedang jeda.
func (r *Redis) siap() bool {
	return r != nil && time.Now().UnixNano() >= r.lewatiSampai.Load()
}

// Buka menyambung ke url (redis://[:sandi@]host:port/db). url kosong
// mengembalikan nil: Redis memang tidak dipakai.
//
// Server yang belum bisa dihubungi saat start TIDAK menggagalkan Buka. Klien
// menyambung ulang sendiri, dan API yang menolak start karena cache-nya belum
// siap berarti satu layanan opsional bisa mematikan seluruh aplikasi.
func Buka(ctx context.Context, url string, log *slog.Logger) (*Redis, error) {
	if url == "" {
		return nil, nil
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("REDIS_URL tidak valid: %w", err)
	}
	opt.DialTimeout = time.Second
	opt.ReadTimeout = batasWaktu
	opt.WriteTimeout = batasWaktu
	r := &Redis{rdb: redis.NewClient(opt), log: log}
	if err := r.Ping(ctx); err != nil {
		log.Warn("redis belum bisa dihubungi, API tetap jalan dan membaca langsung dari database", "error", err)
	}
	return r, nil
}

func (r *Redis) Ping(ctx context.Context) error {
	if r == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, batasWaktu)
	defer cancel()
	return r.rdb.Ping(ctx).Err()
}

func (r *Redis) Close() error {
	if r == nil {
		return nil
	}
	return r.rdb.Close()
}

// catat menjedakan Redis bila err adalah kegagalan sungguhan. Peringatan
// hanya ditulis sekali per jeda.
func (r *Redis) catat(op, key string, err error) {
	if err == nil || errors.Is(err, redis.Nil) {
		return
	}
	lama := r.lewatiSampai.Swap(time.Now().Add(jeda).UnixNano())
	if time.Now().UnixNano() >= lama {
		r.log.Warn("redis gagal, memakai database selama jeda", "op", op, "key", key, "jeda", jeda.String(), "error", err)
	}
}

// Ingat mengembalikan nilai tersimpan di key, atau memanggil ambil lalu
// menyimpan hasilnya selama ttl. Galat dari ambil tidak disimpan.
func Ingat[T any](ctx context.Context, r *Redis, key string, ttl time.Duration, ambil func() (T, error)) (T, error) {
	if !r.siap() {
		return ambil()
	}
	ctx, cancel := context.WithTimeout(ctx, batasWaktu)
	defer cancel()
	if b, err := r.rdb.Get(ctx, awalan+key).Bytes(); err == nil {
		var v T
		if json.Unmarshal(b, &v) == nil {
			return v, nil
		}
	} else {
		r.catat("get", key, err)
	}

	v, err := ambil()
	if err != nil {
		return v, err
	}
	// siap() diperiksa ulang: Get yang gagal barusan sudah menjedakan Redis, dan
	// Set ke Redis yang sama hanya menambah satu batasWaktu lagi.
	if b, err := json.Marshal(v); err == nil && r.siap() {
		// Context baru: ambil() bisa menghabiskan batas waktu yang tadi.
		simpan, batal := context.WithTimeout(context.Background(), batasWaktu)
		defer batal()
		r.catat("set", key, r.rdb.Set(simpan, awalan+key, b, ttl).Err())
	}
	return v, nil
}

// Hapus membuang key, dipanggil setelah data di baliknya berubah. Galatnya
// dicatat dan dikembalikan untuk pemanggil yang perlu jatuh ke cadangan.
func (r *Redis) Hapus(ctx context.Context, key string) error {
	if r == nil {
		return nil
	}
	if !r.siap() {
		return errJeda
	}
	ctx, cancel := context.WithTimeout(ctx, batasWaktu)
	defer cancel()
	err := r.rdb.Del(ctx, awalan+key).Err()
	r.catat("del", key, err)
	return err
}

// Generasi membaca penghitung versi key. Cache yang kuncinya memuat generasi
// gugur seluruhnya saat NaikkanGenerasi dipanggil, tanpa perlu tahu kunci
// mana saja yang pernah dibuat.
func (r *Redis) Generasi(ctx context.Context, key string) int64 {
	if !r.siap() {
		return 0
	}
	ctx, cancel := context.WithTimeout(ctx, batasWaktu)
	defer cancel()
	n, err := r.rdb.Get(ctx, awalan+key).Int64()
	r.catat("get", key, err)
	return n
}

func (r *Redis) NaikkanGenerasi(ctx context.Context, key string) {
	if !r.siap() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, batasWaktu)
	defer cancel()
	r.catat("incr", key, r.rdb.Incr(ctx, awalan+key).Err())
}

// Sekali memegang key selama ttl dan melaporkan apakah pemanggil ini yang
// mendapatkannya. Replika lain yang memanggil dalam rentang itu mendapat
// false. Tanpa Redis selalu true, karena hanya ada satu instance.
//
// Redis yang galat juga menjawab true. Tugas yang dijalankan dua kali lebih
// ringan daripada tugas yang tidak pernah jalan, dan tugas yang memakai kunci
// ini harus aman diulang.
func (r *Redis) Sekali(ctx context.Context, key string, ttl time.Duration) bool {
	if !r.siap() {
		return true
	}
	ctx, cancel := context.WithTimeout(ctx, batasWaktu)
	defer cancel()
	ok, err := r.rdb.SetNX(ctx, awalan+key, time.Now().UTC().Format(time.RFC3339), ttl).Result()
	if err != nil {
		r.catat("setnx", key, err)
		return true
	}
	return ok
}

// Tambah menaikkan penghitung key. Masa hidupnya ttl sejak kenaikan PERTAMA;
// kenaikan berikutnya tidak memperpanjangnya.
func (r *Redis) Tambah(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if !r.siap() {
		return 0, errJeda
	}
	ctx, cancel := context.WithTimeout(ctx, batasWaktu)
	defer cancel()
	pipe := r.rdb.TxPipeline()
	incr := pipe.Incr(ctx, awalan+key)
	pipe.ExpireNX(ctx, awalan+key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		r.catat("incr", key, err)
		return 0, err
	}
	return incr.Val(), nil
}

// Hitung membaca penghitung key beserta sisa masa hidupnya.
func (r *Redis) Hitung(ctx context.Context, key string) (int64, time.Duration, error) {
	if !r.siap() {
		return 0, 0, errJeda
	}
	ctx, cancel := context.WithTimeout(ctx, batasWaktu)
	defer cancel()
	pipe := r.rdb.Pipeline()
	get := pipe.Get(ctx, awalan+key)
	ttl := pipe.PTTL(ctx, awalan+key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		r.catat("get", key, err)
		return 0, 0, err
	}
	n, _ := get.Int64()
	return n, ttl.Val(), nil
}
