# BNI Finance System

Sistem finance untuk **BNI Grow Chapter Management** — mengelola invoice pendaftaran &
renewal keanggotaan, sinkronisasi data dari BNI Visitor Management, dan integrasi
pembayaran via Paper.id.

Dibangun dengan **Vite + React + TypeScript + Tailwind CSS** mengikuti
[rencana teknis](./docs/bni-finance-system-plan.md) dan menerapkan **clean architecture**
(presentation → application → data) sehingga data layer mock dapat ditukar dengan backend
nyata (Supabase / BNI VM API / Paper.id) tanpa mengubah UI.

> Catatan: aplikasi ini berjalan di atas **mock repository** (data in-memory) secara
> default. Tidak ada backend yang dibutuhkan untuk menjalankannya.

---

## ✨ Fitur

- **Dashboard** — KPI (total invoice, dibayar, outstanding, overdue), donut status
  pembayaran, invoice terbaru, dan peringatan renewal.
- **Invoice** — daftar dengan filter (status / tipe / chapter / pencarian), buat invoice
  (pendaftaran & renewal) dengan auto-fill biaya + periode, detail invoice lengkap dengan
  **audit trail**, serta aksi siklus hidup: _Kirim ke Paper.id → Tandai Lunas → Batalkan_.
- **Renewal Due** — deteksi member yang masa keanggotaannya berakhir (≤30 hari / lewat),
  dengan **bulk-select** & **bulk-generate** invoice renewal.
- **Member & Chapter** — data hasil sinkronisasi dari BNI VM, riwayat invoice per member.
- **Pembayaran** — riwayat pembayaran (disimulasikan dari webhook Paper.id).
- **Pengaturan Biaya** — konfigurasi nominal pendaftaran & renewal.
- **Sinkronisasi** — trigger manual pull data dari BNI VM.
- **Auth** — login National Admin (mock, session di localStorage).

---

## 🚀 Menjalankan

```bash
npm install
npm run dev        # http://localhost:5173
```

Login dengan kredensial **apa pun** (mis. `admin@bni-finance.com` / `admin123`).

Skrip lain:

```bash
npm run build      # type-check + build produksi ke dist/
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
│   ├── index.ts         #    Composition root — pilih implementasi (mock ↔ real)
│   └── mock/            #    Implementasi in-memory (seed, store, repositories)
│
├── hooks/               # 🟨 Application layer (useAsync, …)
│
├── components/
│   ├── ui/              # Design system primitives (Button, Card, Badge, Table, …)
│   └── layout/          # Sidebar, Topbar, AppLayout
│
└── features/            # 🟧 Presentation — satu folder per domain
    ├── auth/
    ├── dashboard/
    ├── invoices/
    ├── members/
    ├── chapters/
    ├── payments/
    └── settings/
```

### Kenapa repository pattern?

Setiap halaman memanggil `invoiceService`, `memberService`, dst. dari `@/services` — yang
tipenya adalah _interface_ di `services/types.ts`. Mengganti backend cukup dengan menambah
implementasi baru dan mengubah satu baris di `services/index.ts`:

```ts
// services/index.ts
const useMock = import.meta.env.VITE_USE_MOCK !== 'false'
export const services = useMock ? mockServices : supabaseServices // ← tukar di sini
```

---

## 🔌 Integrasi ke Backend Nyata (peta ke rencana)

| Bagian rencana | Status di repo ini | Cara mengaktifkan |
|---|---|---|
| Sync member & chapter dari BNI VM | Mock (`mock/*Repository.sync`) | Implement `ChapterRepository` / `MemberRepository` yang fetch `GET /api/finance/*` |
| Buat & kirim invoice | Mock (`invoiceRepository.create/send`) | Implement `InvoiceRepository.send` memanggil Paper.id `POST /v1/invoices` |
| Webhook pembayaran Paper.id | Disimulasikan via `markPaid()` | Pindahkan ke endpoint server `POST /api/webhooks/paper-id` |
| Auth National Admin | Mock (localStorage) | Implement `AuthRepository` dengan Supabase Auth |
| Audit log | In-memory (`invoice_audit_log`) | Map ke tabel Supabase yang sama |

Env var tersedia di [`.env.example`](./.env.example).

---

## 🎨 Design System

- **Warna brand**: merah BNI (`brand.500 = #e2231a`) + skala netral `ink`.
- **Font**: Inter.
- Primitives di `components/ui` (Button, Card, Badge, Table, Modal, Toast, StatCard,
  DonutChart, dll.) menjaga konsistensi visual di seluruh aplikasi.

---

## 🧱 Tech Stack

| Layer | Pilihan |
|---|---|
| Build tool | Vite 5 |
| UI | React 18 + TypeScript |
| Routing | React Router 6 (data router) |
| Styling | Tailwind CSS 3 |
| Ikon | lucide-react |
| Data (saat ini) | Mock in-memory repositories |
