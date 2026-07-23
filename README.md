# BNI Finance Hub

Sistem finance untuk **BNI Grow Chapter Management** — mengelola invoice pendaftaran &
renewal keanggotaan, sinkronisasi data dari BNI Visitor Management, pembayaran
(**Paper.id** atau **Xendit self-payment**), pelaporan keuangan, dan ekspor data.

Dibangun dengan **Vite + React + TypeScript + Tailwind CSS**, dapat dipasang sebagai
**PWA**, mengikuti [rencana teknis](./docs/bni-finance-system-plan.md) dan menerapkan
**clean architecture** (presentation → application → data) sehingga data layer mock dapat
ditukar dengan backend nyata (**REST API Go** / BNI VM API / Paper.id / Xendit) tanpa
mengubah UI.

> 📖 **Dokumentasi sistem lengkap (arsitektur, payment Xendit, edge functions, deploy):**
> [`docs/SYSTEM.md`](./docs/SYSTEM.md)
>
> Default berjalan di atas **mock repository** (data in-memory) — tanpa backend. Set
> `VITE_USE_MOCK=false` + `VITE_API_URL` untuk memakai backend Go di `backend/`.

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
- **Paper.id** — kirim invoice & terima pembayaran via webhook.
- **Xendit self-payment** — halaman pembayaran publik `/pay/:id` (Virtual Account / QRIS)
  tanpa perlu login. Mode aktif dipilih di **Metode Pembayaran**.

### Lainnya
- **Ekspor CSV & PDF** pada Invoice, Pembayaran, dan Laporan — PDF berlabel BNI (siap cetak /
  Save as PDF), CSV ber-BOM UTF-8 agar rapi di Excel.
- **Notifikasi** — feed tagihan terlambat / jatuh tempo / pembayaran diterima, dengan badge
  jumlah belum-dibaca pada lonceng.
- **Profil** — ubah nama & kata sandi (butuh kata sandi lama, mode non-mock).
- **PWA** — installable, navigasi bottom-tab di mobile, sadar safe-area.
- **Pengaturan Biaya** — konfigurasi nominal pendaftaran & renewal.
- **Sinkronisasi** — tarik manual data dari BNI VM. Berjalan di server, jadi
  tokennya tidak pernah ada di browser; member yang hilang dinonaktifkan, bukan
  dihapus, agar riwayat tagihan utuh.
- **Auth** — login berbasis JWT dengan dua peran (Admin / User); di mode mock
  memakai localStorage.

---

## 🚀 Menjalankan

```bash
npm install
npm run dev        # http://localhost:5173
```

**Mode mock** (default) — login dengan kredensial **apa pun** (mis. `admin@bni-finance.com`
/ `admin123`).

**Mode API** — jalankan `backend/` lebih dulu (lihat bagian Backend), lalu buat
`.env.local`:

```
VITE_USE_MOCK=false
VITE_API_URL=http://localhost:8080
```

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
│   ├── index.ts         #    Pilih implementasi (mock ↔ api) via VITE_USE_MOCK
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
    ├── pay/             #    Halaman pembayaran publik (Xendit, tanpa login)
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
const useMock = import.meta.env.VITE_USE_MOCK !== 'false'
export const services = useMock ? mockServices : apiServices // ← tukar di sini
```

---

## 🔌 Backend

Aplikasi berjalan penuh di atas **Postgres lokal + REST API Go** di `backend/`.
Tidak ada lagi ketergantungan Supabase.

| Kebutuhan | Dulu (Supabase) | Sekarang |
|---|---|---|
| Database | Supabase Postgres | Postgres lokal — `db/schema.sql` |
| Autentikasi | Supabase Auth | Tabel `users` + JWT (PBKDF2 + HS256, stdlib) |
| Otorisasi | RLS per baris | Middleware peran di backend |
| Penyimpanan berkas | Storage bucket | Berkas di disk (`UPLOAD_DIR`) |
| Halaman bayar publik | Edge Function | Endpoint `/public/**` |
| Webhook Xendit | Edge Function | Endpoint `/webhooks/xendit` |

```bash
# 1. database + backend
cd backend
cp .env.example .env      # isi DATABASE_URL + JWT_SECRET
make db-reset             # skema + data contoh
make run                  # http://localhost:8080

# 2. frontend (terminal lain)
cd ..
echo "VITE_USE_MOCK=false" >> .env.local
echo "VITE_API_URL=http://localhost:8080" >> .env.local
npm run dev               # http://localhost:5173
```

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
| Publik | `GET /api/v1/public/invoices/{id}` · `POST /webhooks/xendit` |
| Sinkronisasi | `POST /api/v1/sync` — tarik member & chapter dari BNI VM |

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
| Database | Postgres 14+ — skema di `db/schema.sql` |
| Autentikasi | JWT HS256 + PBKDF2, seluruhnya pustaka standar Go |
| Pembayaran | Paper.id · Xendit (Virtual Account / QRIS) |
| Data | Mock in-memory (default) ↔ REST API Go (`VITE_USE_MOCK=false`) |
| Hosting | Vercel |
