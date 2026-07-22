# BNI Finance — Backend API (Go)

REST API untuk **CRUD Invoice & Pembayaran**, terpisah dari aplikasi Vite di `../src`.
Menggunakan **Postgres yang sama** dengan Supabase (`../supabase/schema.sql`), jadi data
antara frontend dan backend ini konsisten.

Dibangun dengan **Go + pustaka standar** (`net/http` routing Go 1.22) — satu-satunya
dependensi eksternal adalah driver Postgres `pgx/v5`.

---

## ⚠️ Prasyarat database

Backend ini membaca kolom yang ditambahkan oleh migrasi. Jalankan di Supabase SQL
Editor **sebelum** menjalankan server, sesuai urutan:

| Migrasi | Kenapa dibutuhkan |
|---|---|
| `0002_xendit_self_payment.sql` | Kolom `xendit_*` pada `invoices` |
| `0003_manual_payment.sql` | Kolom `proof_url` & `note` pada `payments` — **di-SELECT setiap query pembayaran**; tanpa ini semua endpoint `/payments` error |
| `0004_performance_indexes.sql` | Indeks untuk `ORDER BY`/filter (lihat Performa) |

---

## 🚀 Menjalankan

```bash
cp .env.example .env     # lalu isi DATABASE_URL
make run                 # http://localhost:8080
```

Perintah lain:

```bash
make build   # binary ke bin/api
make test    # unit test
make check   # vet + test (gate sebelum commit)
```

---

## 🏗️ Struktur

Berlapis satu arah **handler → service → repository**, sejalan dengan repository
pattern di frontend.

```
backend/
├── cmd/api/main.go            # entrypoint: wiring, middleware, graceful shutdown
└── internal/
    ├── config/                # env + loader .env sederhana
    ├── database/              # pool pgx + ping saat start (fail fast)
    ├── domain/                # model, validasi, aturan transisi status
    ├── httpx/                 # envelope JSON, error → status, middleware
    ├── invoice/               # repository · service · handler
    └── payment/               # repository · service · handler
```

| Lapis | Tanggung jawab |
|---|---|
| `handler` | Parsing HTTP, query filter, kode status |
| `service` | Aturan bisnis & validasi (transisi status, settle invoice) |
| `repository` | SQL murni, tanpa logika bisnis |

---

## 🔌 Endpoint

Base URL: `/api/v1`. Semua respons JSON **camelCase**, sama seperti tipe di
`../src/types`, sehingga bisa langsung dipakai klien TypeScript.

### Invoice

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/invoices` | Daftar + filter (lihat tabel di bawah) |
| `POST` | `/invoices` | Buat invoice (status awal `draft`) |
| `GET` | `/invoices/{id}` | Detail |
| `PATCH` | `/invoices/{id}` | Ubah sebagian field |
| `DELETE` | `/invoices/{id}` | Hapus |

**Query filter `GET /invoices`:**
`status` (termasuk `outstanding` = sent+overdue) · `type` · `chapterId` · `memberId` ·
`q` (nomor invoice) · `dueFrom` · `dueTo` · `issuedFrom` · `issuedTo` · `limit` (maks 200) · `offset`

### Pembayaran

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/payments` | Daftar + filter |
| `POST` | `/payments` | Catat pembayaran (default **melunasi** invoice) |
| `GET` | `/payments/{id}` | Detail |
| `PATCH` | `/payments/{id}` | Ubah sebagian field |
| `DELETE` | `/payments/{id}` | Hapus |

**Query filter `GET /payments`:** `invoiceId` · `method` · `paidFrom` · `paidTo` · `limit` · `offset`

### Lain-lain
`GET /healthz` — cek kesehatan + ping database (tanpa autentikasi).

---

## 📐 Aturan bisnis yang ditegakkan

Ini bukan CRUD telanjang — beberapa aturan domain dijaga di lapis service:

- **Transisi status invoice** mengikuti siklus di UI:
  `draft → sent → paid`, bisa dibatalkan sebelum lunas. `paid` dan `cancelled`
  bersifat final. Lompatan seperti `draft → paid` ditolak **409**.
- **Invoice lunas/batal tidak bisa diubah** nominal maupun periodenya.
- **Invoice dengan pembayaran tidak bisa dihapus** — 409, arahkan untuk membatalkan.
- **Mencatat pembayaran = melunasi invoice**, dalam **satu transaksi** dengan
  `SELECT … FOR UPDATE`, sehingga tidak mungkin ada pembayaran tercatat pada invoice
  yang masih belum lunas. Kirim `"settleInvoice": false` untuk melewati pelunasan.
- **Invoice yang sudah dibatalkan menolak pembayaran** (409).
- **Nomor invoice dibuat otomatis** (`INV-2026-013`) bila tidak dikirim.

---

## 📦 Contoh

```bash
# Buat invoice
curl -X POST localhost:8080/api/v1/invoices \
  -H 'Content-Type: application/json' \
  -d '{
    "memberId": "mem-001",
    "chapterId": "ch-garuda",
    "type": "renewal",
    "amount": 1500000,
    "dueDate": "2026-07-01",
    "periodStart": "2026-07-01",
    "periodEnd": "2027-07-01"
  }'

# Terbitkan (draft → sent)
curl -X PATCH localhost:8080/api/v1/invoices/<id> \
  -H 'Content-Type: application/json' -d '{"status":"sent"}'

# Catat pembayaran (otomatis melunasi invoice)
curl -X POST localhost:8080/api/v1/payments \
  -H 'Content-Type: application/json' \
  -d '{"invoiceId":"<id>","amount":1500000,"paymentMethod":"bank_transfer"}'

# Daftar invoice outstanding di satu chapter
curl 'localhost:8080/api/v1/invoices?status=outstanding&chapterId=ch-garuda&limit=20'
```

Bentuk respons daftar:

```json
{
  "data": [ { "id": "…", "number": "INV-2026-013", "…": "…" } ],
  "meta": { "total": 47, "limit": 20, "offset": 0 }
}
```

Bentuk error: `{ "error": "pesan" }`.

---

## 🔒 Keamanan

- Isi `API_KEY` untuk mewajibkan `Authorization: Bearer <key>` pada seluruh `/api/**`.
  Dibandingkan **constant-time**. Dikosongkan = terbuka (khusus development).
- `ALLOWED_ORIGINS` membatasi CORS; `*` hanya untuk development.
- `DATABASE_URL` memakai kredensial database langsung — **melewati RLS Supabase**.
  Karena itu backend ini harus dijalankan sebagai layanan tepercaya (server-side),
  jangan pernah diekspos langsung ke browser tanpa `API_KEY`.
- Semua query memakai parameter binding (`$1`, `$2`) — tidak ada perangkaian string
  dari input pengguna.

---

## ⚡ Performa

Stress test menembak **wiring HTTP asli** (router + middleware + handler +
validasi) di atas store in-memory, jadi yang diukur adalah lapisan API — bukan
Postgres.

```bash
make stress        # ukur throughput & latensi
make stress-race   # deteksi data race di bawah beban
```

Hasil di mesin pengembangan (Apple Silicon, 64 worker, 16.000 request campuran
55% list / 20% get / 15% create invoice / 5% create payment / 5% list payment):

| Metrik | Hasil |
|---|---|
| Throughput | ~25.300 req/s |
| Latensi | p50 1,6 ms · p95 6,4 ms · p99 13,3 ms · max 32,6 ms |
| Error | 0 dari 16.000 (tanpa 4xx maupun 5xx) |
| Data race | Tidak ada (`-race` bersih) |

> Angka ini **batas atas** lapisan HTTP. Dengan Postgres sungguhan, throughput
> akan ditentukan database — karena itu indeks di migrasi 0004 penting.

### Catatan skala yang diketahui

- Setiap request daftar menjalankan `COUNT(*)` untuk `meta.total`. Nyaman untuk
  paginasi, tapi jadi mahal pada tabel besar; pertimbangkan perkiraan hitungan
  bila baris sudah ratusan ribu.
- Filter `q` memakai `ILIKE '%…%'` — membutuhkan indeks trigram (disediakan di
  migrasi 0004), dan hanya mencari **nomor invoice**, belum nama member seperti
  di UI.
- Kolom `amount` bertipe `integer` (maks ~Rp 2,14 miliar per baris). Cukup untuk
  iuran keanggotaan; ubah ke `bigint` bila kelak ada tagihan lebih besar.

---

## ✅ Status pengujian

Terverifikasi di mesin ini: `go vet` bersih, `gofmt` rapi, seluruh test lulus
dengan `-race` — transisi status, validasi input, round-trip `Date` YYYY-MM-DD,
guard API key & CORS, stress test 16.000 request, dan pelunasan paralel pada
invoice yang sama. Binary build serta gagal-cepat dengan pesan jelas saat
`DATABASE_URL` kosong/salah.

**Belum diuji:** jalur yang menyentuh database (query & transaksi) — tidak ada
Postgres lokal di mesin ini, dan menguji ke database produksi akan menulis data nyata.
Jalankan `make run` terhadap database dev untuk memvalidasi ujung-ke-ujung.
