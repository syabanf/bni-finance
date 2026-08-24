# BNI Finance Hub — Ringkasan Sistem

Satu dokumen untuk memahami keseluruhan sistem: apa isinya, cara menjalankannya, di mana
batas keamanannya, dan hal-hal yang pernah menggigit di lapangan.

Dokumen ini **ringkasan**, bukan pengganti. Rujukan mendalam ada di bagian
[Peta dokumen](#peta-dokumen) di bawah.

---

## 1. Apa ini

Platform pengelolaan **invoice keanggotaan BNI** — pendaftaran dan
perpanjangan tahunan. Data member ditarik dari BNI Visitor Management, invoice diterbitkan
ke **Paper.id**, pembayaran masuk lewat webhook, dan
seluruhnya terekam dengan jejak audit.

| Lapisan | Pilihan |
|---|---|
| Frontend | Vite 5 · React 18 · TypeScript · Tailwind 3 · React Router 6 · PWA |
| Backend | Go 1.25, pustaka standar (satu dependensi: `pgx/v5`) |
| Database | Postgres 14+ — skema di [`db/init.sql`](../db/init.sql) |
| Autentikasi | JWT HS256 + PBKDF2, seluruhnya stdlib Go |
| Pembayaran | Paper.id |
| Hosting | Vercel (frontend) · backend server-side terpisah |

---

## 2. Dua sumber data

Aplikasi bisa berjalan **tanpa backend sama sekali**. Pilihannya lewat tombol
_Sumber Data_ — ada di halaman Pengaturan **dan** di bawah form login.

| Mode | Data | Untuk |
|---|---|---|
| **Data Contoh** | Store in-memory di browser | Demo, pengembangan UI, eksplorasi |
| **Backend API** | Postgres lewat REST API Go | Pengembangan nyata, staging, produksi |

Pilihan tersimpan di browser, jadi berpindah tidak perlu menjalankan ulang dev server.
`VITE_USE_MOCK` hanya menentukan nilai **awal** untuk browser yang belum pernah memilih.

> Pemilih itu sengaja juga ada di halaman login. Kalau backend mati saat mode API, login
> adalah satu-satunya layar yang bisa dijangkau — tanpa tombol di sana Anda terkunci.

**Seluruh fitur hidup di kedua mode**, termasuk Konsol API dan Blackbox. Di Data Contoh,
59 endpoint dijawab di browser oleh store mock dan panggilan integrasi direkam oleh
perekam blackbox versi browser dengan bentuk rekaman yang identik.

---

## 3. Menjalankan

### Data Contoh — tanpa backend

```bash
npm install
npm run setup      # buat .env.local + backend/.env, rahasia dibangkitkan
npm run dev        # http://localhost:5173
```

Masuk dengan kredensial apa pun, atau tekan tombol **Masuk Cepat**.

`npm run setup` menyalin kedua `.env.example` lalu membangkitkan `JWT_SECRET`,
token callback Paper.id, dan kata sandi admin awal. Kredensial pihak ketiga
dibiarkan kosong — fitur terkait menjawab 503 dengan pesan jelas sampai diisi.
Berkas yang sudah ada tidak pernah ditimpa.

### Backend API

```bash
# 1. database + backend
cd backend
cp .env.example .env      # isi DATABASE_URL + JWT_SECRET
make db-reset             # skema + data contoh
make run                  # http://localhost:8080

# 2. frontend (terminal lain)
cd ..
echo "VITE_API_URL=http://localhost:8080" >> .env.local
npm run dev
```

Lalu pilih **Backend API** pada pemilih _Sumber Data_. Admin pertama dibuat otomatis dari
`SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` saat tabel `users` masih kosong.

---

## 4. Arsitektur

Dependensi mengalir satu arah: **presentation → application → data**. Halaman tidak pernah
tahu dari mana data berasal — hanya bergantung pada _interface_ repository.

```
src/
├── app/            Composition root: router & providers
├── types/          Domain models (Invoice, Member, Chapter, Payment, …)
├── services/       Data layer
│   ├── types.ts        Repository INTERFACES (kontrak)
│   ├── dataSource.ts   Sumber data aktif, bisa ditukar runtime
│   ├── index.ts        Pilih implementasi berdasarkan dataSource
│   ├── mock/           In-memory (seed, store, mockApi, blackbox)
│   └── api/            HTTP → backend Go
├── hooks/          Application layer (useAsync, …)
├── components/     ui/ (design system) · layout/
└── features/       Presentation — satu folder per domain

backend/            REST API Go
db/                 Skema Postgres + data contoh
```

Aturan yang dijaga: halaman **tidak pernah** mengimpor dari `services/mock/` atau
`services/api/` secara langsung — selalu lewat `@/services` atau `@/services/appSettings`.
Melanggarnya pernah membuat halaman Pengaturan menembak backend saat mode mock.

---

## 5. Alur bisnis invoice

```
Draft ──Terbitkan──▶ Outstanding ──pembayaran──▶ Lunas
  │                       │
  └──────Batalkan─────────┴──▶ Dibatalkan
```

Yang terjadi saat **Terbitkan** bergantung pada `self_payment_mode`:

Invoice didorong ke Paper.id, link bayar & PDF disimpan, kanal Email/WhatsApp
dijalankan, status → `sent`. Pembayaran mandiri lewat Xendit sudah dihapus —
member membayar di halaman Paper.id.

Invoice dan baris audit pertamanya ditulis dalam **satu transaksi** — tidak ada invoice
tanpa jejak.

### Pengiriman ke member

Kanal Email/WhatsApp adalah **kebijakan server**, dibaca dari `app_settings`, bukan pilihan
per-klik. Alasannya: penerbitan terjadi massal lewat tiga jalur (tombol Terbitkan, aksi
massal, "buat + kirim"). Bila tiap pemanggil harus menyertakan flagnya, satu jalur yang
lupa akan berhenti mengantar ke member sambil tetap melaporkan sukses.

Member tanpa alamat email dilewati untuk kanal email; WhatsApp tetap jalan.

---

## 6. API

**59 endpoint**, terbagi per tag:

| Tag | Jml | Tag | Jml |
|---|---|---|---|
| Auth | 11 | Pengaturan | 6 |
| Invoice | 7 | Member | 6 |
| Pembayaran | 5 | Chapter | 5 |
| Paper.id | 5 | Sistem | 5 |
| Publik | 3 | Unggahan | 2 |
| Blackbox | 2 | Dashboard | 1 |
| Sinkronisasi | 1 | | |

Respons memakai **camelCase** identik dengan tipe di `src/types`, jadi bisa dikonsumsi
klien TypeScript tanpa lapisan pemetaan.

### Tiga tingkat akses

| Tingkat | Cakupan |
|---|---|
| **Publik** | Login, halaman bayar, webhook, dokumentasi, health |
| **Login** | Semua pembacaan |
| **Admin** | Semua penulisan |

Cek peran di UI hanya menyembunyikan tombol — batas sebenarnya ada di backend.

### Dokumentasi & alat

| Alat | Lokasi | Isi |
|---|---|---|
| Halaman `/docs` | Backend, mis. `http://localhost:8080/docs` | Referensi lengkap, tanpa CDN |
| `/openapi.yaml` · `/openapi.json` | Backend | Spesifikasi OpenAPI 3.1 |
| **Konsol API** | Aplikasi → Alat Teknis | Postman built-in; body berupa form berlabel |
| **Blackbox** | Aplikasi → Alat Teknis | Rekaman tiap panggilan integrasi, dua arah |
| **`/metrics`** | Backend | Metrik Prometheus — lihat bagian di bawah |

Daftar endpoint Konsol API **dibangkitkan dari spesifikasi** (`npm run api-collection`,
ikut jalan pada `npm run build`), jadi tidak bisa melenceng dari route sungguhan.

Spesifikasinya sendiri dijaga lima tes di `internal/apidocs`, yang memindai sumber dengan
`go/ast` dan membandingkannya dengan `openapi.yaml`:

| Tes | Menangkap |
|---|---|
| `TestEveryRouteIsDocumented` | Route terdaftar tapi tidak terdokumentasi |
| `TestNoPhantomEndpoints` | Terdokumentasi tapi tidak dilayani |
| `TestSpecJSONMatchesYAML` | `openapi.json` basi — lupa `make docs` |
| `TestPublicOperationsAreExplicit` | Rute publik yang tidak menandai `security: []`, dan sebaliknya |
| `TestCORSAllowsEveryRegisteredMethod` | Method yang dipakai router tapi hilang dari allow-list CORS |

---

### Metrik Prometheus

`GET /metrics` menyajikan format eksposisi teks, siap di-scrape.

| Metrik | Tipe | Label |
|---|---|---|
| `http_requests_total` | counter | `method`, `route`, `status` |
| `http_request_duration_seconds` | histogram | `method`, `route` |
| `http_requests_in_flight` | gauge | — |
| `integration_calls_total` | counter | `integration`, `direction`, `outcome` |
| `integration_call_duration_seconds` | histogram | `integration`, `direction` |
| `db_pool_conns_*`, `db_pool_acquire_*` | gauge | — |
| `go_goroutines`, `go_memstats_*`, `go_gc_cycles_total` | gauge | — |

Ditulis tanpa `prometheus/client_golang` — pustaka itu menarik ~10 paket
transitif ke modul yang justru dijaga hanya punya satu dependensi langsung.
Yang dibutuhkan cuma counter, histogram, dan format eksposisi. Konsekuensinya
tidak ada exemplar, native histogram, atau pushgateway; semua scraper membaca
format teks, jadi tak satu pun dari itu diperlukan untuk di-scrape.

**Label `route` memakai POLA route, bukan URL.** `/api/v1/invoices/{id}` adalah
satu series; URL berarti satu series per invoice, dan Prometheus menyimpan
setiap series yang pernah dilihatnya di memori — lalu lintas normal akan
menghabiskan heap scraper sampai mati. Path yang tak cocok route mana pun
dikumpulkan ke `other`, karena path 404 dipilih oleh pemanggil.

**Tidak ada data bisnis di label.** Tidak ada id member, nomor invoice, email,
atau nominal — dijamin konstruksi, bukan konfigurasi, dan dijaga tes.

Metrik integrasi diturunkan dari perekam blackbox, satu-satunya titik yang
sudah dilewati keempat integrasi. Blackbox menyimpan N panggilan terakhir;
counter ini bertahan lebih lama dan bisa memicu alert, yang ring buffer tidak
bisa.

---

## 7. Konfigurasi

### `app_settings` — runtime, bisa diubah dari UI

| Kunci | Arti | Bawaan |
|---|---|---|
| `invoice_draft_days_before` | Draft renewal disusun H-N | `30` |
| `invoice_due_days_after` | Jatuh tempo = terbit + N hari | `30` |
| `paperid_send_email` | Paper.id mengantar invoice via email | `true` |
| `paperid_send_whatsapp` | Paper.id mengantar via WhatsApp | `true` |

Kunci yang belum ada bernilai kosong. Untuk kanal kirim, **hanya `false` yang mematikan** —
invoice yang tidak pernah sampai ke member bukan hasil yang lebih ringan, melainkan
kegagalan diam-diam.

### Variabel lingkungan

**Frontend** — satu-satunya yang boleh ada di sisi klien:

| Berkas | Status | Isi |
|---|---|---|
| `.env.example` | ter-commit | Templat, disalin oleh `npm run setup` |
| `.env.local` | di-gitignore | Env kerja Anda |
| `.env.production` | **ter-commit** | Dimuat `vite build`; tertanam ke bundel publik |

```
VITE_API_URL=http://localhost:8080
VITE_USE_MOCK=true          # hanya nilai awal
```

> `.env.production` pernah menyetel `VITE_USE_MOCK=false` tanpa `VITE_API_URL`
> — tanpa VITE_API_URL. Pengunjung pertama mendarat di mode Backend API yang
> menunjuk ke origin Vercel sendiri, dan karena semua path diarahkan ke
> `index.html`, permintaan API dijawab HTML berstatus 200. Sekarang bawaannya
> Data Contoh, jadi deployment langsung bisa dipakai.

**Backend** (`backend/.env`) — seluruhnya server-side:

```
DATABASE_URL, JWT_SECRET               # wajib
PAPER_ID_CLIENT_ID, PAPER_ID_CLIENT_SECRET, PAPER_ID_CALLBACK_TOKEN
BNI_VM_URL, BNI_VM_TOKEN
SEED_ADMIN_EMAIL, SEED_ADMIN_PASSWORD
AUTH_QUICK_LOGIN                       # kosong = mati
METRICS_TOKEN                          # kosong = /metrics terbuka
```

> ⚠️ **Vite menyisipkan setiap nilai `VITE_*` ke bundel JavaScript publik.** Rahasia apa pun
> di sana bisa dibaca siapa saja yang membuka devtools. `DATABASE_URL` juga menyambung
> sebagai peran tepercaya yang melewati seluruh otorisasi aplikasi — backend hanya boleh
> berjalan sebagai layanan server-side.

### Masuk cepat

`AUTH_QUICK_LOGIN` adalah **daftar email**, bukan saklar on/off. Saklar berarti sekali
dinyalakan di produksi, *semua* akun bisa dimasuki tanpa kata sandi. Kosong = fitur mati
dan endpointnya menjawab 404, sehingga produksi tidak memberi petunjuk bahwa rute itu ada.
Server mencatat peringatan di setiap start selama fitur ini menyala.

---

## 8. Pengujian

```bash
# Frontend
npm run typecheck
npm run build

# Backend — unit, tanpa database
cd backend && make test

# Backend — integration, menembak Postgres sungguhan
make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
```

> ⚠️ Integration test **meng-`TRUNCATE` tabel bisnis** dan menolak database yang namanya
> tidak mengandung `test`/`dev`. Muat ulang `db/init.sql` setelahnya.
> `-p 1` wajib — paket-paketnya berbagi satu database dan Go menjalankan paket berbeda
> secara paralel.

### Uji beban — k6

```bash
brew install k6
BASE_URL=http://127.0.0.1:8123 ADMIN_EMAIL=… ADMIN_PASSWORD=… k6 run loadtest/api.js
```

Tiga skenario berjalan bersamaan, meniru lalu lintas nyata: `baca` (admin
menjelajah, ramping 0→30 VU), `tulis` (penerbitan invoice 10/dtk — jalur yang
dulu punya balapan penomoran), dan `publik` (halaman bayar tanpa token, yang
sekaligus memeriksa tidak ada kontak member yang bocor).

**Threshold-nya menggagalkan run** (exit ≠ 0), jadi skrip ini bisa jadi gerbang
CI: <1% error, p95 per skenario, nol nomor invoice ganda, dan >99% check lulus.
Angkanya sengaja longgar — batas "jelas rusak", bukan target performa; batas
ketat milik pengukuran di perangkat produksi.

Login terjadi sekali di `setup()`, bukan per-VU: 100 VU yang login sendiri-
sendiri berarti 100 PBKDF2 600 ribu iterasi, dan yang terukur jadi hashing.

> Skenario `tulis` benar-benar membuat invoice. Jalankan hanya ke database
> sekali pakai, dan muat ulang `db/init.sql` setelahnya.

**Tes E2E** (`internal/api/e2e_test.go`) menelusuri seluruh perjalanan lewat HTTP: masuk →
tolak tanpa token → tolak peran user untuk menulis → daftarkan chapter & member → terbitkan
draft → periksa audit → dorong ke Paper.id sungguhan → pastikan member yang ganti nomor
tetap bisa ditagih → periksa blackbox tak memuat kredensial → tolak callback bertoken salah
→ lunasi lewat callback → ulangi tanpa menggandakan → cocokkan dashboard → pastikan halaman
publik tak membocorkan kontak.

Leg Paper.id butuh `PAPER_ID_CLIENT_ID`/`_SECRET`; tanpa itu langkah kirim **dilewati,
bukan dipalsukan** — palsu di sana tidak membuktikan apa pun tentang integrasi yang justru
paling sering patah.

---

## 9. Jebakan yang pernah menggigit

Kumpulan hal yang tidak terlihat dari membaca kode, dan masing-masing pernah memakan waktu.

**Nomor invoice dibakar permanen oleh Paper.id.** Sekali sebuah nomor dipakai, percobaan
berikutnya dijawab 403 *"invoice number sudah dipakai"* selamanya. Penomoran lokal dihitung
dari `COUNT(*)`, jadi setelah database di-truncate atau di-restore dari backup lama,
penomorannya mulai ulang dan bentrok. Gejalanya: 409 pada setiap penerbitan sampai
penomorannya melewati angka yang sudah terbakar.

**`customer.id` terikat kontak di Paper.id.** Paper.id mengunci id itu ke nama/email/telepon
saat pertama melihatnya, lalu menolak id yang sama dengan kontak berbeda —
*"Failed partner doesn't match"*, sebuah 400 yang tidak menyebut kontak sama sekali. Karena
sinkronisasi BNI VM memperbarui nomor telepon secara rutin, referensinya kini memuat digest
pendek dari kontak: berubah kontak → id berubah → Paper.id mendaftarkan customer baru
alih-alih menolak invoice.

**Method HTTP harus ada di allow-list CORS.** `PUT` pernah hilang dari
`Access-Control-Allow-Methods`, sehingga menyimpan setting apa pun dan mengganti kata sandi
tidak pernah bisa dari browser — sementara `curl` dan seluruh handler test lolos, karena
keduanya tidak melakukan preflight. Sekarang dijaga tes yang memindai router.

**`self_payment_mode` membypass Paper.id sepenuhnya.** Kalau menyala, tombol Terbitkan
menjadikan invoice `sent` tanpa pernah memanggil Paper.id. Bukan bug — tapi mudah
membingungkan saat mengira integrasinya rusak.

**Paper.id staging belum tentu benar-benar mengantar.** Respons `201` dengan `status_send`
dikonfirmasi hanya berarti permintaannya diterima. Kalau pesan tidak masuk, tunjukkan
rekaman Blackbox ke Paper.id.

---

## 10. Deploy

**Frontend** — Vercel. `npm run build` menjalankan generator koleksi API, type-check, lalu
build PWA ke `dist/`. `vercel.json` mengatur SPA rewrite + header keamanan.

**Backend** — layanan server-side terpisah dengan Postgres. Setelah upgrade, pastikan kunci
`app_settings` yang baru ada:

```sql
insert into app_settings (key, value) values
  ('paperid_send_email','true'),
  ('paperid_send_whatsapp','true'),
  ('invoice_due_days_after','30')
on conflict (key) do nothing;
```

Halaman `/docs`, `/openapi.yaml`, dan `/openapi.json` dilayani backend, bukan Vercel.

---

## Peta dokumen

| Dokumen | Status | Isi |
|---|---|---|
| [`README.md`](../README.md) | ✅ berlaku | Fitur lengkap, arsitektur frontend, cara menjalankan |
| [`backend/README.md`](../backend/README.md) | ✅ berlaku | Endpoint, aturan bisnis, keamanan, performa, hasil stress test |
| [`db/init.sql`](../db/init.sql) | ✅ berlaku | Skema database — sumber kebenaran |
| `/docs` di backend | ✅ berlaku | Referensi API, dibangkitkan dari `openapi.yaml` |
| [`docs/SYSTEM-PLAN.md`](./SYSTEM-PLAN.md) | 📜 historis | Rencana teknis awal |
| [`docs/BACKLOG.md`](./BACKLOG.md) · [`docs/epics/`](./epics) · [`docs/features/`](./features) | 📜 historis | User story & acceptance criteria per fitur. `EPIC-001` dan `11-public-payment` menggambarkan pembayaran mandiri Xendit yang sudah dihapus. |
| [`docs/AGENTIC-WORKFLOW.md`](./AGENTIC-WORKFLOW.md) · [`AGENTS.md`](../AGENTS.md) | ✅ berlaku | Cara kerja agen pada repo ini |
