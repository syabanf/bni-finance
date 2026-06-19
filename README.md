# BNI Finance Hub

Sistem finance untuk **BNI Grow Chapter Management** — mengelola invoice pendaftaran &
renewal keanggotaan, sinkronisasi data dari BNI Visitor Management, pembayaran
(**Paper.id** atau **Xendit self-payment**), pelaporan keuangan, dan ekspor data.

Dibangun dengan **Vite + React + TypeScript + Tailwind CSS**, dapat dipasang sebagai
**PWA**, mengikuti [rencana teknis](./docs/bni-finance-system-plan.md) dan menerapkan
**clean architecture** (presentation → application → data) sehingga data layer mock dapat
ditukar dengan backend nyata (**Supabase** / BNI VM API / Paper.id / Xendit) tanpa
mengubah UI.

> 📖 **Dokumentasi sistem lengkap (arsitektur, payment Xendit, edge functions, deploy):**
> [`docs/SYSTEM.md`](./docs/SYSTEM.md)
>
> Default berjalan di atas **mock repository** (data in-memory) — tanpa backend. Set
> `VITE_USE_MOCK=false` (+ kredensial Supabase) untuk memakai backend nyata; ter-deploy
> di **Vercel** dengan payment **Xendit** (mode test).

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
- **Profil** — ubah nama & kata sandi (Supabase Auth pada mode produksi).
- **PWA** — installable, navigasi bottom-tab di mobile, sadar safe-area.
- **Pengaturan Biaya** — konfigurasi nominal pendaftaran & renewal.
- **Sinkronisasi** — trigger manual pull data dari BNI VM.
- **Auth** — login National Admin (mock localStorage atau Supabase Auth).

---

## 🚀 Menjalankan

```bash
npm install
npm run dev        # http://localhost:5173
```

**Mode mock** (default) — login dengan kredensial **apa pun** (mis. `admin@bni-finance.com`
/ `admin123`).

**Mode Supabase** — buat `.env.local` lalu set:

```
VITE_USE_MOCK=false
VITE_SUPABASE_URL=https://xxxx.supabase.co
VITE_SUPABASE_ANON_KEY=eyJ...
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
│   ├── index.ts         #    Pilih implementasi (mock ↔ supabase) via VITE_USE_MOCK
│   ├── mock/            #    Implementasi in-memory (seed, store, repositories)
│   └── supabase/        #    Implementasi Supabase (Postgres + Auth + Storage)
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

supabase/                # SQL schema, migrations, RLS, seed, & Edge Functions
└── functions/           #    xendit-create-payment · xendit-webhook ·
                         #    auto-create-invoices · get-public-invoice
```

### Kenapa repository pattern?

Setiap halaman memanggil `invoiceService`, `memberService`, dst. dari `@/services` — yang
tipenya adalah _interface_ di `services/types.ts`. Mengganti backend cukup dengan memilih
implementasi lain di `services/index.ts`:

```ts
// services/index.ts
const useMock = import.meta.env.VITE_USE_MOCK !== 'false'
export const services = useMock ? mockServices : supabaseServices // ← tukar di sini
```

---

## 🔌 Backend (Supabase)

Implementasi Supabase tersedia di `services/supabase/` dan aktif saat `VITE_USE_MOCK=false`.

| Bagian | Mekanisme |
|---|---|
| Database & Auth | Supabase Postgres + Supabase Auth (`supabase/schema.sql`, `supabase/rls.sql`) |
| Pembayaran mandiri | Edge Function `xendit-create-payment` + halaman publik `/pay/:id` |
| Webhook pembayaran | Edge Function `xendit-webhook` (verifikasi `x-callback-token`) |
| Invoice otomatis | Edge Function `auto-create-invoices` |
| Bukti pembayaran manual | Supabase Storage bucket `payment-proofs` |
| Sync member & chapter | `ChapterRepository` / `MemberRepository` → BNI VM API |

> ⚠️ Kunci **service-role** dan **password database** bersifat **rahasia** — jangan pernah
> dimasukkan ke variabel `VITE_*` karena akan ter-bundle ke klien. Hanya `VITE_SUPABASE_URL`
> dan `VITE_SUPABASE_ANON_KEY` (publik) yang boleh ada di sisi klien.

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
| Backend (opsional) | Supabase — Postgres, Auth, Storage, Edge Functions |
| Pembayaran | Paper.id · Xendit (Virtual Account / QRIS) |
| Data | Mock in-memory (default) ↔ Supabase (`VITE_USE_MOCK=false`) |
| Hosting | Vercel |
