# BNI Finance Hub

Sistem finance untuk **BNI Grow Chapter Management** — mengelola invoice pendaftaran &
renewal keanggotaan, sinkronisasi data dari BNI Visitor Management, pembayaran
(**Paper.id**), pelaporan keuangan, dan ekspor data.

Dibangun dengan **Vite + React + TypeScript + Tailwind CSS**, dapat dipasang sebagai
**PWA**, mengikuti [rencana teknis](./docs/bni-finance-system-plan.md) dan menerapkan
**clean architecture** (presentation → application → data) sehingga data layer mock dapat
ditukar dengan backend nyata (**REST API Go** / BNI VM API / Paper.id) tanpa
mengubah UI.

> 📖 **Ringkasan sistem dalam satu dokumen** — arsitektur, alur bisnis, konfigurasi,
> pengujian, deploy, dan jebakan yang pernah menggigit: [`docs/OVERVIEW.md`](./docs/OVERVIEW.md)
>
> Default berjalan di atas **mock repository** (data in-memory) — tanpa backend.
> Berpindah ke backend Go dilakukan lewat **tombol _Sumber Data_** di halaman
> Pengaturan atau di bawah form login, bukan lewat env.

---

## ✨ Fitur

### Inti
- **Dashboard** — KPI (total invoice, dibayar, outstanding, overdue) dengan **drill-down**,
  donut status pembayaran, invoice terbaru, peringatan renewal, dan statistik per chapter.
- **Invoice** — daftar dengan **filter** (status / tipe / chapter / jatuh tempo / pencarian)
  dan **summary card yang mengikuti filter**, buat invoice (auto-fill biaya + periode),
  detail lengkap dengan **audit trail**, siklus hidup _Kirim → Tandai Lunas → Batalkan_,
  **kirim WhatsApp**, serta **bulk send** (ke Paper.id, atau Email/WhatsApp sesuai mode
  pembayaran).
- **Renewal Due** — deteksi member yang masa keanggotaannya berakhir (≤30 hari / lewat),
  dengan **bulk-select** & **bulk-generate** invoice renewal.
- **Pembayaran** — riwayat pembayaran dengan filter (metode / waktu bayar / pencarian) dan
  summary mengikuti filter, plus **pencatatan pembayaran manual** + unggah bukti.
- **Laporan Keuangan** — ringkasan per periode (Bulan Ini / Bulan Lalu / Tahun Ini / Kustom):
  KPI, tren bulanan _ditagih vs diterima_, rincian per chapter, per tipe, dan metode
  pembayaran.
- **Member & Chapter** — data hasil sinkronisasi BNI VM, riwayat invoice per member, filter
  kota & jatuh tempo.

### Pembayaran
- **Paper.id** — menerbitkan invoice mendorongnya ke Paper.id (invoice + link
  bayar ter-hosting), pembayaran masuk lewat webhook. Kredensial di server;
  aktif saat Self Payment Mode OFF.
- **Pengiriman ke member** — kanal Email/WhatsApp diatur di **Metode Pembayaran**
  dan dibaca **server** dari `app_settings` (`paperid_send_email`,
  `paperid_send_whatsapp`). Sengaja bukan pilihan per-klik: penerbitan terjadi
  massal (Terbitkan, aksi massal, "buat + kirim"), dan bila tiap pemanggil harus
  menyertakan flagnya, satu jalur yang lupa akan berhenti mengantar ke member
  sambil tetap melaporkan sukses. Bawaannya **nyala** — hanya nilai `false` yang
  mematikan, karena invoice yang tidak pernah sampai ke member bukan hasil yang
  lebih ringan melainkan kegagalan diam-diam. Member tanpa email dilewati untuk
  kanal email; WhatsApp tetap jalan.
  tanpa perlu login. Mode aktif dipilih di **Metode Pembayaran**.

### Lainnya
- **Ekspor CSV & PDF** pada Invoice, Pembayaran, dan Laporan — PDF berlabel BNI (siap cetak /
  Save as PDF), CSV ber-BOM UTF-8 agar rapi di Excel.
- **Notifikasi** — feed tagihan terlambat / jatuh tempo / pembayaran diterima, dengan badge
  jumlah belum-dibaca pada lonceng.
- **Profil** — ubah nama & kata sandi (butuh kata sandi lama, mode non-mock).
- **PWA** — installable, navigasi bottom-tab di mobile, sadar safe-area.
- **Pengaturan Biaya** — konfigurasi nominal pendaftaran & renewal.
- **Konsol API** — halaman admin bergaya Postman untuk seluruh 59 endpoint.
  Daftar endpointnya **dibangkitkan dari spesifikasi OpenAPI backend**
  (`npm run api-collection`), jadi tidak bisa melenceng dari route sungguhan.
  Ada **Isi Otomatis** yang mengambil id dan tanggal yang benar-benar ada lewat
  sumber data yang sedang aktif — termasuk memenuhi prasyarat tiap endpoint
  (mis. `/invoices/{id}/send` diberi invoice berstatus draft). Endpoint Paper.id
  ikut di dalamnya, termasuk konsol uji yang menampilkan payload persis yang
  dikirim ke Paper.id (default **dry-run**) dan simulasi callback pembayaran
  yang menjalankan handler webhook sungguhan.
  Pada mode **Data Contoh** seluruh endpoint dijawab di browser oleh store mock,
  jadi konsol tetap jalan tanpa backend; pada mode **Backend API** setiap
  permintaan yang mengubah data dikonfirmasi lebih dulu.
- **Blackbox Integrasi** — halaman admin berisi rekaman tiap panggilan ke/dari
  Paper.id dan BNI VM: request JSON, endpoint yang dihubungi, response
  JSON, dan status berhasil/gagal. Berguna saat integrasi bermasalah.
- **Sinkronisasi** — tarik manual data dari BNI VM. Berjalan di server, jadi
  tokennya tidak pernah ada di browser; member yang hilang dinonaktifkan, bukan
  dihapus, agar riwayat tagihan utuh.
- **Auth** — login berbasis JWT dengan dua peran (Admin / User); di mode mock
  memakai localStorage.
- **Masuk cepat** — tombol satu klik di halaman login, tersedia di **kedua**
  mode. Di Data Contoh memakai akun mock. Di Backend API, akun yang boleh
  ditentukan server lewat `AUTH_QUICK_LOGIN` (daftar email, kosong = mati) dan
  **kata sandinya tidak pernah dikirim ke browser** — sebelumnya jalur ini
  memakai `VITE_DEMO_PASSWORD`, yang Vite tanam ke bundel JS publik. Sengaja
  daftar email alih-alih saklar on/off: saklar berarti sekali dinyalakan di
  produksi, semua akun jadi bisa dimasuki tanpa kata sandi. Server mencatat
  peringatan di setiap start selama fitur ini menyala.

---

## 🚀 Menjalankan

```bash
npm install
npm run setup      # buat .env.local + backend/.env, rahasia dibangkitkan
npm run dev        # http://localhost:5173
```

`npm run setup` menyalin kedua berkas `.env.example` lalu mengisi yang bisa
diisi sendiri: `JWT_SECRET`, token callback Paper.id, dan kata sandi admin awal
— semuanya dibangkitkan acak, karena itu satu-satunya cara menaruh rahasia yang
benar-benar dipakai tanpa pernah menulisnya ke berkas yang ikut ter-commit.
Kredensial pihak ketiga dibiarkan kosong; fitur yang bersangkutan menjawab 503
dengan pesan yang jelas selama belum diisi. Berkas yang sudah ada tidak pernah
ditimpa.

**Mode mock** (default) — login dengan kredensial **apa pun** (mis. `admin@bni-finance.com`
/ `admin123`), atau pakai tombol **Login Cepat**.

**Berpindah sumber data** dilakukan lewat **tombol**, bukan env: ada pemilih
_Sumber Data_ di halaman **Pengaturan** dan di bawah **form login**. Pilihannya
tersimpan di browser, jadi tidak perlu menjalankan ulang dev server.

`VITE_USE_MOCK` di `.env.local` hanya menentukan nilai **awal** untuk browser
yang belum pernah memilih. Untuk mode API, jalankan `backend/` lebih dulu dan
set alamatnya:

```
VITE_API_URL=http://localhost:8080
```

> Pemilih itu sengaja juga ada di halaman login: kalau backend mati saat mode
> API, login adalah satu-satunya layar yang bisa dijangkau — tanpa tombol di
> sana Anda akan terkunci.

Skrip lain:

```bash
npm run build      # type-check + build produksi (PWA) ke dist/
npm run preview    # preview hasil build
npm run typecheck  # type-check tanpa emit
```

---

## 🏗️ Arsitektur

Dependensi mengalir satu arah: **presentation → application → data**. Halaman tidak pernah
mengetahui dari mana data berasal — hanya bergantung pada _interface_ repository.

```
src/
├── app/                 # Composition root: router & providers
│   ├── router.tsx
│   └── Providers.tsx
│
├── types/               # 🟦 Domain models (Invoice, Member, Chapter, Payment, …)
│
├── services/            # 🟩 Data layer
│   ├── types.ts         #    Repository INTERFACES (kontrak)
│   ├── dataSource.ts    #    Sumber data aktif (mock ↔ api), bisa ditukar runtime
│   ├── index.ts         #    Pilih implementasi repository berdasarkan dataSource
│   ├── appSettings.ts   #    Titik komposisi untuk konfigurasi key/value
│   ├── mock/            #    Implementasi in-memory (seed, store, repositories)
│   └── api/             #    Implementasi HTTP → backend Go
│
├── hooks/               # 🟨 Application layer (useAsync, …)
├── lib/                 # Helpers (csv, pdfReport, status, whatsapp, date, format, …)
│
├── components/
│   ├── ui/              # Design system primitives (Button, Card, Badge, Table, …)
│   └── layout/          # Sidebar, Topbar, BottomNav, AppLayout
│
└── features/            # 🟧 Presentation — satu folder per domain
    ├── auth/  dashboard/  invoices/  members/  chapters/
    ├── payments/  reports/  notifications/  profile/
    ├── urgent/  settings/  misc/

db/                      # Skema Postgres + data contoh
backend/                 # REST API Go (lihat backend/README.md)
```

### Kenapa repository pattern?

Setiap halaman memanggil `invoiceService`, `memberService`, dst. dari `@/services` — yang
tipenya adalah _interface_ di `services/types.ts`. Mengganti backend cukup dengan memilih
implementasi lain di `services/index.ts`:

```ts
// services/index.ts
const useMock = isMockMode()                                  // ← dari dataSource.ts
export const services = useMock ? mockServices : apiServices  // ← tukar di sini
```

Halaman **tidak pernah** mengimpor dari `services/mock/` atau `services/api/`
secara langsung — selalu lewat `@/services` atau `@/services/appSettings`.

---

## 🔌 Backend

Aplikasi berjalan penuh di atas **Postgres lokal + REST API Go** di `backend/`.
Tidak ada lagi ketergantungan Supabase.

| Kebutuhan | Dulu (Supabase) | Sekarang |
|---|---|---|
| Database | Supabase Postgres | Postgres lokal — `db/init.sql` |
| Autentikasi | Supabase Auth | Tabel `users` + JWT (PBKDF2 + HS256, stdlib) |
| Otorisasi | RLS per baris | Middleware peran di backend |
| Penyimpanan berkas | Storage bucket | Berkas di disk (`UPLOAD_DIR`) |
| Halaman bayar publik | Edge Function | Endpoint `/public/**` |

```bash
# 1. database + backend
cd backend
cp .env.example .env      # isi DATABASE_URL + JWT_SECRET
make db-reset             # skema + data contoh
make run                  # http://localhost:8080

# 2. frontend (terminal lain)
cd ..
echo "VITE_API_URL=http://localhost:8080" >> .env.local
npm run dev               # http://localhost:5173
```

Lalu pilih **Backend API** pada pemilih _Sumber Data_ di halaman login.

Masuk dengan `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` dari `backend/.env` —
admin pertama dibuat otomatis saat tabel `users` masih kosong.

> ⚠️ `DATABASE_URL` dan `JWT_SECRET` adalah **rahasia**. Jangan pernah dimasukkan
> ke variabel `VITE_*`, karena Vite menyisipkannya ke bundel JavaScript publik.
> Hanya `VITE_API_URL` yang boleh ada di sisi klien.

---

## ⚙️ REST API (Go)

`backend/` berisi REST API yang melayani seluruh aplikasi, ditulis dengan Go +
pustaka standar (satu dependensi: driver `pgx/v5`).

| Resource | Endpoint |
|---|---|
| Invoice | `GET·POST /api/v1/invoices` · `GET·PATCH·DELETE /api/v1/invoices/{id}` |
| Pembayaran | `GET·POST /api/v1/payments` · `GET·PATCH·DELETE /api/v1/payments/{id}` |
| Member | `GET·POST /api/v1/members` · `/members/{id}` · `/members/renewal-due` |
| Chapter | `GET·POST /api/v1/chapters` · `/chapters/{id}` |
| Pengaturan | `/api/v1/fee-settings` · `/api/v1/app-settings/{key}` |
| Jejak audit | `GET·POST /api/v1/invoices/{id}/audit` |
| Dashboard | `GET /api/v1/dashboard/summary` |
| Autentikasi | `POST /api/v1/auth/login` · `/auth/me` · `/auth/password` · `/users/**` |
| Unggahan | `POST /api/v1/uploads` · `GET /uploads/{nama}` |
| Publik | *(tidak ada — seluruh endpoint memerlukan token)* |
| Sinkronisasi | `POST /api/v1/sync` — tarik member & chapter dari BNI VM |
| Paper.id | `POST /api/v1/invoices/{id}/send` · `POST /webhooks/paperid` |
| Blackbox | `GET·DELETE /api/v1/blackbox` — rekaman lalu lintas integrasi |
| Konsol uji | `/api/v1/paperid/status` · `/paperid/test-invoice` · `/paperid/test-callback` |

Respons memakai **camelCase** yang identik dengan tipe di `src/types`, jadi bisa
dikonsumsi klien TypeScript tanpa lapisan pemetaan.

**Dokumentasi API** tersedia langsung dari server yang berjalan:

| URL | Isi |
|---|---|
| `http://localhost:8080/docs` | Referensi lengkap di browser, dengan pencarian |
| `/openapi.yaml` · `/openapi.json` | Spesifikasi OpenAPI 3.1 untuk Postman atau generator klien |

Detail lain — aturan bisnis, keamanan, dan hasil stress test — ada di
[`backend/README.md`](backend/README.md).

Tiga tingkat akses: **publik** (login, halaman bayar, webhook), **login**
(semua pembacaan), **admin** (semua penulisan). Cek peran di UI hanya
menyembunyikan tombol — batas sebenarnya ada di backend.

> ⚠️ `DATABASE_URL` menyambung sebagai peran tepercaya dan melewati seluruh
> otorisasi aplikasi. Jalankan backend hanya sebagai layanan server-side.

---

## 🎨 Design System

- **Warna brand**: merah BNI (`brand.500 = #e2231a`) + skala netral `ink`.
- **Font**: Inter.
- Primitives di `components/ui` (Button, Card, Badge, Table, Modal, Toast, StatCard,
  SummaryCard, DonutChart, ExportMenu, dll.) menjaga konsistensi visual di seluruh aplikasi.

---

## 🧱 Tech Stack

| Layer | Pilihan |
|---|---|
| Build tool | Vite 5 (+ vite-plugin-pwa) |
| UI | React 18 + TypeScript |
| Routing | React Router 6 (data router) |
| Styling | Tailwind CSS 3 |
| Ikon | lucide-react |
| Ekspor | CSV (BOM UTF-8) + PDF (dokumen cetak berlabel BNI) |
| Backend | Go 1.25 (`net/http`) + `pgx/v5` — lihat `backend/` |
| Database | Postgres 14+ — skema di `db/init.sql` |
| Autentikasi | JWT HS256 + PBKDF2, seluruhnya pustaka standar Go |
| Pembayaran | Paper.id |
| Data | Mock in-memory (default) ↔ REST API Go (`VITE_USE_MOCK=false`) |
| Hosting | Vercel |
