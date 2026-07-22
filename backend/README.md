# BNI Finance — Backend API (Go)

REST API untuk **seluruh entitas** sistem — invoice, pembayaran, member, chapter,
pengaturan, jejak audit, dan ringkasan dashboard — terpisah dari aplikasi Vite di
`../src`. Menggunakan **Postgres yang sama** dengan Supabase (`../supabase/schema.sql`),
jadi data antara frontend dan backend ini konsisten.

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
| `0004_performance_indexes.sql` | Indeks untuk `ORDER BY`/filter pada invoice & pembayaran |
| `0005_app_settings_and_indexes.sql` | **Membuat tabel `app_settings`** — sebelumnya tidak pernah didefinisikan di mana pun, padahal 0002 sudah meng-`insert` ke sana dan tiga edge function membacanya. Plus indeks untuk member, chapter, dan audit log |

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
    ├── api/                   # perakitan rute + rantai middleware
    ├── config/                # env + loader .env sederhana
    ├── database/              # pool pgx + ping saat start (fail fast)
    ├── domain/                # model, validasi, aturan transisi status
    ├── httpx/                 # envelope JSON, error → status, middleware
    ├── invoice/               # repository · service · handler
    ├── payment/               # repository · service · handler
    ├── member/                # repository · service · handler
    ├── chapter/               # repository · service · handler
    ├── settings/              # fee_settings + app_settings
    ├── audit/                 # jejak invoice (baca + catatan manual)
    └── dashboard/             # agregat KPI, read-only
```

Setiap resource didaftarkan lewat `api.Services`. Field yang `nil` tidak
didaftarkan — itulah yang membuat test bisa menyalakan sebagian saja.

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

### Member

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/members` | Daftar + filter |
| `POST` | `/members` | Tambah member (`id` opsional — dibuatkan bila kosong) |
| `GET` | `/members/renewal-due` | Keanggotaan yang habis dalam `days` hari ke depan |
| `GET` | `/members/{id}` | Detail |
| `PATCH` | `/members/{id}` | Ubah sebagian field |
| `DELETE` | `/members/{id}` | Hapus |

**Query filter `GET /members`:** `chapterId` · `status` · `q` (nama/email/perusahaan) ·
`renewalFrom` · `renewalTo` · `limit` · `offset`

Setiap member dibaca beserta chapter-nya (`LEFT JOIN`), jadi bentuknya sama dengan
`MemberWithChapter` di frontend — tidak perlu request kedua.

### Chapter

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/chapters` | Daftar + filter `q` · `cityName` · `areaName` |
| `POST` | `/chapters` | Tambah chapter |
| `GET` | `/chapters/{id}` | Detail |
| `PATCH` | `/chapters/{id}` | Ubah sebagian field |
| `DELETE` | `/chapters/{id}` | Hapus |

### Pengaturan

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/fee-settings` | Biaya pendaftaran & perpanjangan yang berlaku |
| `PATCH` | `/fee-settings` | Ubah biaya |
| `GET` | `/app-settings` | Semua konfigurasi key/value |
| `GET` | `/app-settings/{key}` | Satu konfigurasi |
| `PUT` | `/app-settings/{key}` | Simpan/ubah (upsert) |
| `DELETE` | `/app-settings/{key}` | Hapus |

### Jejak audit

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/invoices/{id}/audit` | Timeline invoice, terbaru dulu |
| `POST` | `/invoices/{id}/audit` | Tambah catatan manual (`notes` wajib) |

### Dashboard

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/dashboard/summary` | Agregat KPI — bentuknya persis `DashboardSummary` di frontend |

`?months=6` mengatur panjang grafik tren bulanan (1–24).

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
- **Jejak audit ditulis otomatis** pada transaksi yang sama dengan perubahan yang
  dicatatnya, baik lewat `PATCH /invoices/{id}` maupun lewat pelunasan dari
  `POST /payments`. Timeline tidak mungkin melenceng dari invoice-nya.
- **Chapter dan member yang masih dirujuk tidak bisa dihapus** — 409 dengan
  jumlah data yang menahannya, bukan pelanggaran foreign key yang membingungkan.
- **Biaya tidak boleh negatif**, dan `PATCH /fee-settings` tanpa field apa pun
  ditolak alih-alih diam-diam tidak melakukan apa-apa.

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
- **`app_settings` menyimpan token BNI VM.** Karena itu key yang namanya
  mengandung `token`, `secret`, `password`, `apikey`, `credential`, atau `private`
  **selalu terbaca tersamar** (`••••••`) — bersifat write-only lewat API ini.
  Nilai aslinya tetap utuh di database; hanya jalur keluarnya yang ditutup.
  Mengirim balik nilai tersamar ditolak 400, supaya tidak ada yang menimpa token
  asli dengan tanda bintang.

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
| Throughput | ~22.300 req/s |
| Latensi | p50 1,9 ms · p95 7,6 ms · p99 14,4 ms · max 27,2 ms |
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
- `GET /dashboard/summary` menjalankan lima query agregat tanpa cache. Murah pada
  ukuran data sekarang; bila nanti berat, itu kandidat pertama untuk di-cache.
- Angka `trend` membandingkan 30 hari terakhir dengan 30 hari sebelumnya.
  Pertumbuhan dari nol dilaporkan `+100%`, bukan tak hingga.

---

## ✅ Status pengujian

Terverifikasi di mesin ini: `go vet` bersih, `gofmt` rapi, seluruh test lulus
dengan `-race` — transisi status, validasi input, round-trip `Date` YYYY-MM-DD,
guard API key & CORS, CRUD member & chapter lewat rantai handler asli,
penyamaran kredensial `app_settings`, guard hapus 409, `/members/renewal-due`
yang tidak terbayangi `/members/{id}`, perhitungan `trend`, daftar kosong yang
tetap `[]`, stress test 16.000 request, dan pelunasan paralel pada
invoice yang sama. Binary build serta gagal-cepat dengan pesan jelas saat
`DATABASE_URL` kosong/salah.

**Belum diuji:** jalur yang menyentuh database (query & transaksi) — tidak ada
Postgres lokal di mesin ini, dan menguji ke database produksi akan menulis data nyata.
Jalankan `make run` terhadap database dev untuk memvalidasi ujung-ke-ujung.
