# BNI Finance — Backend API (Go)

Backend **satu-satunya** untuk sistem ini: seluruh entitas (invoice, pembayaran,
member, chapter, pengaturan, jejak audit, dashboard) plus **autentikasi, unggah
berkas, dan halaman bayar publik**. Frontend Vite di `../src` berbicara hanya
dengan API ini.

Dibangun dengan **Go + pustaka standar** — satu-satunya dependensi eksternal
adalah driver Postgres `pgx/v5`. Hashing kata sandi (`crypto/pbkdf2`) dan token
sesi (HS256 di atas `crypto/hmac`) memakai stdlib, tanpa pustaka pihak ketiga.

> **Semuanya di Postgres.** Data di `db/init.sql`; autentikasi lewat tabel
> `users` + JWT; berkas unggahan di disk (`UPLOAD_DIR`); otorisasi lewat
> middleware `auth.RequireAuth` + `RequireAdmin`.

---

## 🚀 Menjalankan

Butuh Postgres 14+ yang berjalan lokal.

```bash
cp .env.example .env     # isi DATABASE_URL + JWT_SECRET
make db-reset            # buat database, terapkan skema, isi data contoh
make run                 # http://localhost:8080
```

`JWT_SECRET` wajib, minimal 32 karakter:

```bash
openssl rand -base64 48
```

Admin pertama dibuat otomatis dari `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD`,
**hanya** saat tabel `users` masih kosong.

Perintah database lain (`DB_NAME`, `DB_PORT`, `PSQL` bisa ditimpa):

```bash
make db-create   # buat database saja
make db-init     # terapkan ../db/init.sql — skema + akun awal + data contoh
make db-seed     # isi data contoh
make db-drop     # hapus database — minta konfirmasi
```

Perintah lain:

```bash
make build   # binary ke bin/api
make test    # unit test (integration test dilewati)
make check   # vet + test (gate sebelum commit)

# Integration test — menembak Postgres sungguhan. WAJIB database sekali pakai:
# test ini me-TRUNCATE tabel dan menolak nama yang tak mengandung 'test'/'dev'.
make test-integration TEST_DATABASE_URL=postgres://localhost/bni_finance_dev
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
    ├── auth/                  # akun lokal, PBKDF2, JWT, middleware peran
    ├── invoice/               # repository · service · handler
    ├── payment/               # repository · service · handler
    ├── member/                # repository · service · handler
    ├── chapter/               # repository · service · handler
    ├── settings/              # fee_settings + app_settings
    ├── audit/                 # jejak invoice (baca + catatan manual)
    ├── dashboard/             # agregat KPI, read-only
    ├── upload/                # bukti pembayaran di disk
    ├── publicpay/             # halaman bayar publik + Xendit
    ├── sync/                  # tarik member & chapter dari BNI VM
    ├── paperid/               # kirim invoice ke Paper.id + callback bayar
    ├── blackbox/              # perekam lalu lintas integrasi (in-memory)
    └── apidocs/               # spesifikasi OpenAPI + halaman /docs
```

Setiap resource didaftarkan lewat `api.Services`. Field yang `nil` tidak
didaftarkan — itulah yang membuat test bisa menyalakan sebagian saja.

| Lapis | Tanggung jawab |
|---|---|
| `handler` | Parsing HTTP, query filter, kode status |
| `service` | Aturan bisnis & validasi (transisi status, settle invoice) |
| `repository` | SQL murni, tanpa logika bisnis |

---

## 📘 Dokumentasi API

Spesifikasi **OpenAPI 3.1** lengkap ada di
[`internal/apidocs/openapi.yaml`](internal/apidocs/openapi.yaml) — 49 operasi,
38 skema, beserta aturan bisnis dan kode galat tiap endpoint.

Server menyajikannya sendiri (tanpa token):

| URL | Isi |
|---|---|
| `/docs` | Referensi yang bisa dibaca di browser, dengan pencarian |
| `/openapi.yaml` | Spesifikasi — arahkan Postman, Insomnia, atau generator klien ke sini |
| `/openapi.json` | Bentuk JSON-nya, yang dirender halaman `/docs` |

Ketiganya **tertanam di dalam binary**, dan halaman `/docs` sepenuhnya mandiri —
tanpa CDN, tanpa renderer eksternal. Dokumentasi yang butuh internet akan mati
justru ketika Anda sedang menelusuri masalah tanpa koneksi.

Setelah mengubah `openapi.yaml`:

```bash
make docs    # hasilkan ulang openapi.json
```

> **Dokumentasi ini diuji, bukan sekadar ditulis.** `internal/apidocs`
> membandingkan spesifikasi dengan rute yang benar-benar terdaftar — dibaca dari
> kode sumber dengan `go/ast`, karena `http.ServeMux` tidak bisa mendaftar
> polanya. Endpoint yang ditambahkan tanpa didokumentasikan **menggagalkan
> test**, begitu pula endpoint yang didokumentasikan tapi tidak ada. Test
> ketiga memastikan setiap rute tanpa autentikasi memang menandai
> `security: []`, sehingga tidak ada endpoint yang diam-diam terbuka.

---

## 🔌 Endpoint

Ringkasan di bawah untuk orientasi cepat; detail lengkapnya di `/docs`.

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

### Autentikasi

| Method | Path | Akses |
|---|---|---|
| `POST` | `/auth/login` | **publik** — mengembalikan token + profil |
| `GET` | `/auth/me` | login |
| `PATCH` | `/auth/me` | login — ubah nama |
| `PUT` | `/auth/password` | login — wajib kirim `currentPassword` |
| `GET·POST` | `/users` | admin |
| `PATCH` | `/users/{id}/role` | admin |
| `PUT` | `/users/{id}/password` | admin — reset kata sandi orang lain |
| `DELETE` | `/users/{id}` | admin |

### Sinkronisasi

| Method | Path | Akses |
|---|---|---|
| `POST` | `/sync` | admin — tarik member & chapter dari BNI Visitor Management |

Satu panggilan menyegarkan keduanya: BNI VM tidak punya endpoint chapter, jadi
chapter diturunkan dari daftar member. Token diambil dari
`app_settings.bni_vm_token` lebih dulu, lalu jatuh ke `BNI_VM_TOKEN`.

### Unggahan

| Method | Path | Akses |
|---|---|---|
| `POST` | `/uploads` | admin — multipart, field `file` |
| `GET` | `/uploads/{nama}` | **publik** — nama berkas dibuat acak |

### Publik (tanpa token)

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/public/invoices/{id}` | Proyeksi sempit untuk halaman bayar |
| `POST` | `/public/invoices/{id}/payment` | Buat pembayaran Xendit (VA/QRIS) |
| `POST` | `/webhooks/xendit` | Callback Xendit — butuh `x-callback-token` |

### Paper.id

| Method | Path | Akses |
|---|---|---|
| `POST` | `/invoices/{id}/send` | admin — dorong invoice draft ke Paper.id, simpan link & PDF, status → `sent` |
| `POST` | `/webhooks/paperid` | **publik** — callback pembayaran (secret di URL callback) |

Dipakai saat Self Payment Mode **OFF**. `client_id`/`client_secret` hanya di
server. Callback dicocokkan ke invoice lewat `uuid` Paper.id lalu `number`, dan
bersifat idempoten. Endpoint lookup `GET /sales-invoices?number=…` tersedia di
Paper.id untuk rekonsiliasi manual bila sebuah pengiriman timeout di sisi kita
(nomornya bisa "terbakar" — lihat Aturan bisnis).

### Konsol Uji Paper.id

| Method | Path | Akses |
|---|---|---|
| `GET` | `/paperid/status` | admin — status konfigurasi (tanpa rahasia) |
| `POST` | `/paperid/test-invoice` | admin — uji buat invoice |
| `POST` | `/paperid/test-callback` | admin — simulasi callback pembayaran |

**`dryRun` default `true` pada keduanya.** Mengirim sungguhan membuat invoice
nyata di Paper.id dan memakai nomornya permanen; menjalankan callback sungguhan
menandai invoice lunas. Keduanya harus menjadi pilihan sadar, bukan efek samping
klik pertama.

Nomor uji selalu ber-prefix `TEST-` + stempel waktu + acak, jadi tidak pernah
menghabiskan nomor invoice sungguhan maupun bentrok satu sama lain. Konsol
**tidak menyentuh tabel invoice** — murni uji kontrak.

Simulasi callback menjalankan **handler webhook yang sebenarnya**, jadi yang
diuji jalur produksi, bukan tiruannya.

### Blackbox

| Method | Path | Akses |
|---|---|---|
| `GET` | `/blackbox` | admin — rekaman panggilan integrasi |
| `DELETE` | `/blackbox` | admin — kosongkan rekaman |

Perekam "kotak hitam": setiap panggilan **keluar** ke Paper.id/Xendit/BNI VM dan
setiap **callback masuk** tercatat dengan body request, endpoint, body response,
dan status berhasil/gagal. Ring buffer **di memori** (`BLACKBOX_SIZE`, default
200), terbaru dulu — hilang saat restart, memang disengaja: ini alat diagnosis,
bukan audit trail.

**Kredensial tidak pernah terekam** — hanya body yang dicatat, sedangkan
`client_id`, `client_secret`, dan bearer token ada di header.

Filter: `?integration=paper_id|xendit|bni_vm` · `?direction=outbound|inbound` ·
`?status=failed`.

### Lain-lain

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/healthz` | Cek kesehatan + ping database (tanpa autentikasi) |
| `GET` | `/docs` | Dokumentasi API di browser |
| `GET` | `/openapi.yaml` · `/openapi.json` | Spesifikasi OpenAPI 3.1 |

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
- **Sinkronisasi menonaktifkan, tidak menghapus.** Member yang hilang dari BNI
  VM diubah menjadi `inactive`. Menghapusnya akan melanggar foreign key begitu
  mereka punya invoice, dan membuang riwayat tagihan meski berhasil.
- **Snapshot kosong dari BNI VM ditolak** (502) sebelum menyentuh database —
  menerapkannya berarti menonaktifkan seluruh member sekaligus.
- **Nama tampilan chapter yang diubah lokal tidak ditimpa** sinkronisasi; hanya
  nama kanonik yang mengikuti BNI VM.
- **Menerbitkan invoice ke Paper.id** hanya untuk status `draft`, ditulis dalam
  satu transaksi (field Paper.id + status `sent` + audit). Callback pembayaran
  idempoten. Timeout klien 60 detik: bila sebuah create timeout di sisi kita
  tapi sebenarnya berhasil di Paper.id, nomornya jadi tak bisa dipakai ulang —
  pengiriman ulang membalas **409** yang jelas, bukan kegagalan generik.

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

**Otorisasi ada di sini, dan tidak bisa di tempat lain.** Backend ini menyambung
ke Postgres sebagai SATU peran tepercaya, jadi database melihat satu identitas
saja dan tidak bisa menegakkan apa pun per pengguna. Tiga tingkat akses:

| Tingkat | Isi |
|---|---|
| Publik | `/auth/login`, `/public/**`, `/webhooks/xendit`, `/uploads/{nama}`, `/healthz` |
| Login | Seluruh **pembacaan** `/api/**` |
| Admin | Seluruh **penulisan** + `/users/**` |

Cek di UI (tombol yang disembunyikan) adalah kenyamanan, bukan kontrol akses —
batas sebenarnya `auth.RequireAuth` dan `auth.RequireAdmin`.

- **Kata sandi** di-hash PBKDF2-HMAC-SHA256, 600.000 iterasi, salt acak per baris.
  Jumlah iterasi disimpan di dalam hash sehingga bisa dinaikkan tanpa
  membatalkan kata sandi lama.
- **Token** HS256, hanya algoritma itu yang diterima — `alg: none` dan token yang
  payload-nya ditukar ditolak (ada test-nya). `JWT_SECRET` minimal 32 karakter.
- **Login gagal** memberi pesan identik untuk email tak dikenal dan kata sandi
  salah, dan tetap menghitung satu hash, agar akun tidak bisa dienumerasi.
- **Ganti kata sandi sendiri wajib menyertakan kata sandi lama** — token yang
  dicuri saja tidak cukup untuk mengambil alih akun.
- **Admin terakhir tidak bisa dihapus atau diturunkan** — itu keadaan yang tak
  bisa dipulihkan lewat API.
- `ALLOWED_ORIGINS` membatasi CORS; `*` hanya untuk development.
- **Token BNI VM tidak pernah sampai ke browser.** Dulu halaman mengambilnya
  dari `app_settings` lalu memanggil API lewat proxy; sekarang seluruh
  sinkronisasi berjalan di server. Key-nya mengandung kata `token`, jadi API
  membacanya tersamar.
- `DATABASE_URL` memakai kredensial database langsung dan **melewati seluruh
  otorisasi aplikasi**. Backend harus berjalan server-side; jangan pernah
  diekspos apa adanya ke browser.
- **Unggahan** memakai allowlist tipe (JPG/PNG/WebP/HEIC/PDF), dibatasi ukuran,
  dan namanya dibuat server — nama dari klien adalah jalan menuju path traversal.
  Direktori tidak bisa dijelajahi; URL sengaja tak tertebak karena halaman bayar
  publik harus bisa menampilkannya.
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

**Unit test** (in-memory, `-race` bersih): transisi status, validasi input,
round-trip `Date` YYYY-MM-DD, guard API key & CORS, CRUD member & chapter lewat
rantai handler asli, penyamaran kredensial `app_settings`, guard hapus 409,
`/members/renewal-due` yang tidak terbayangi `/members/{id}`, perhitungan
`trend`, daftar kosong yang tetap `[]`, stress test 16.000 request, dan
pelunasan paralel pada invoice yang sama.

**Integration test** (Postgres 17 lokal, `-race` bersih): seluruh query dieksekusi
database sungguhan — termasuk sinkronisasi, yang diuji menonaktifkan member yang
hilang tanpa kehilangan invoice mereka, dan tidak menimpa nama tampilan chapter
yang sudah diubah lokal — — setiap cabang filter, LEFT JOIN member–chapter, jejak audit
`created → sent → paid` lintas paket, lima agregat dashboard, dan 32 pelunasan
paralel atas satu invoice yang berakhir `paid` dengan **tepat satu** entri audit
`paid` serta `paidAmount` yang tidak menumpuk.

**Batas keamanan** diuji tersendiri: seluruh rute terproteksi menolak request
tanpa token (didaftar satu per satu, bukan sampel), token kedaluwarsa/asing/rusak
ditolak, peran `user` mendapat 403 pada setiap penulisan tapi 200 pada setiap
pembacaan, dan rute publik tetap terbuka.

**Dokumentasi** dijaga tiga test: setiap rute terdaftar harus ada di
spesifikasi, setiap operasi terdokumentasi harus benar-benar dilayani, dan
setiap rute tanpa autentikasi harus menandainya secara eksplisit. Ketiganya
sudah dibuktikan gagal saat drift-nya sengaja dibuat.

> Unit test tidak bisa menangkap SQL yang salah — bagi Go, query hanyalah string.
> Dua bug nyata lolos persis lewat celah itu (`FILTER` menempel pada `coalesce()`,
> dan parameter yang disimpulkan Postgres sebagai `TEXT` karena disambung menjadi
> interval). `make test-integration` ada untuk menjaganya; keduanya sudah
> diverifikasi gagal bila bug-nya dikembalikan.
