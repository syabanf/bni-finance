# BNI Finance Hub — Dokumentasi Sistem

Dokumentasi teknis menyeluruh untuk **BNI Finance Hub**: platform pengelolaan invoice
keanggotaan BNI (pendaftaran & renewal) dengan **pembayaran mandiri (self payment)** via
Xendit, sinkronisasi data dari BNI Visitor Management, dan halaman pembayaran publik untuk
member.

> Status: berjalan di atas **backend nyata (Supabase)** dengan `VITE_USE_MOCK=false`,
> ter-deploy di **Vercel**, payment gateway **Xendit** (mode test/development).

---

## 1. Ringkasan

| Aspek | Detail |
|---|---|
| Tipe | SPA (Single Page Application) |
| Frontend | Vite 5 + React 18 + TypeScript + Tailwind CSS 3 + React Router 6 |
| Backend | Supabase (PostgreSQL + PostgREST + Edge Functions Deno) |
| Payment Gateway | Xendit — Virtual Account (BCA/BNI/Mandiri/BRI) & QRIS |
| Sumber data member | BNI Visitor Management API (`bni-vh.com`) |
| Hosting | Vercel (`https://bni-finance-five.vercel.app`) |
| Ikon | lucide-react · QR: `qrcode.react` |

---

## 2. Arsitektur

Dependensi mengalir satu arah: **presentation → application → data**. Halaman hanya
bergantung pada _interface_ repository, bukan implementasi konkret.

```
src/
├── app/                 # Composition root: router & providers
│   ├── router.tsx       #   termasuk route publik /pay/:id (tanpa auth)
│   └── Providers.tsx    #   AuthProvider + ToastProvider
├── types/               # Domain models (Invoice, Member, Chapter, Payment, …)
├── services/            # Data layer
│   ├── types.ts         #   Repository INTERFACES (kontrak)
│   ├── index.ts         #   Pilih implementasi: mock ↔ supabase
│   ├── mock/            #   Implementasi in-memory
│   └── supabase/        #   Implementasi nyata (Supabase)
│       ├── invoiceRepository.ts
│       ├── memberRepository.ts
│       ├── settingsRepository.ts   # getAppSetting / setAppSetting
│       └── paymentGateway.ts        # createXenditPayment, isSelfPaymentMode
├── hooks/               # Application layer (useAsync)
├── components/
│   ├── ui/              # Design system (Button, Card, BniLogo, …)
│   └── layout/          # Sidebar, Topbar, AppLayout (auth guard)
└── features/            # Presentation — satu folder per domain
    ├── auth/  dashboard/  invoices/  members/  chapters/
    ├── payments/  settings/  urgent/
    └── pay/             # Halaman pembayaran publik member (PublicPaymentPage)
```

Pemilihan backend di satu baris:

```ts
// services/index.ts
const useMock = import.meta.env.VITE_USE_MOCK !== 'false'
export const services = useMock ? mockServices : supabaseServices
```

---

## 3. Fitur Utama

- **Dashboard** — KPI (total, lunas, outstanding, overdue), statistik per chapter.
- **Invoice** — daftar + filter (status/tipe/chapter/cari), detail + audit trail,
  siklus: _Draft → Terbitkan (Outstanding) → Lunas / Batal_.
  - **Preview & Download PDF** invoice (render HTML self-contained, cetak browser).
  - **Link Pembayaran** (Salin / Kirim WhatsApp) saat invoice Outstanding.
- **Perlu Tindakan (Urgent)** — overdue, renewal jatuh tempo, perlu invoice pendaftaran.
- **Member & Chapter** — hasil sinkronisasi BNI VM; kolom **Due Date** (renewal_date).
- **Pengaturan Biaya** — nominal pendaftaran & renewal.
- **Metode Pembayaran** (menu sidebar) — toggle **Self Payment Mode** (ON Xendit / OFF Paper.id).
- **Konfigurasi Timing Invoice** — draft H-N sebelum renewal; jatuh tempo H+N setelah terbit.
- **Sinkronisasi Data** — pull manual dari BNI VM (hanya berjalan di dev via proxy Vite).

---

## 4. Self Payment Mode (Xendit)

Toggle di **Sidebar → Metode Pembayaran** (`app_settings.self_payment_mode`).

| Mode | Perilaku |
|---|---|
| **ON** | Member bayar mandiri via Xendit (VA / QRIS), auto-Lunas via webhook. |
| **OFF** | Integrasi Paper.id (simulasi link pembayaran), tandai Lunas manual. |

### Alur pembayaran (mode ON)

```
Admin: Draft → Terbitkan (Outstanding) → Salin/Kirim link /pay/:id
                                   │
Member buka  /pay/:id (publik, tanpa login)
   1. Lihat informasi invoice
   2. Download Invoice (PDF)
   3. Pilih metode → VA (BCA/BNI/Mandiri/BRI) atau QRIS
        └─ Edge Function `xendit-create-payment` (secret key di server)
             └─ Xendit buat VA/QR → simpan ke invoices.xendit_*
   4. Member bayar di m-banking / scan QRIS
        └─ Xendit kirim callback → Edge Function `xendit-webhook`
             ├─ verifikasi x-callback-token
             ├─ invoice → status = paid (+ catat di payments)
             └─ member.renewal_date dimajukan ke period_end (+1 tahun)
   5. Halaman /pay auto-poll (7 dtk) → tampil "Pembayaran Berhasil"
```

### Batasan
- **QRIS maks Rp 10.000.000** per transaksi (limit Xendit). Untuk nominal di atas itu,
  QRIS otomatis dinonaktifkan — gunakan Virtual Account.
- `external_id` Xendit dibuat unik per pembuatan (`{number}-{method}-{rand}`) agar tidak
  bentrok saat ganti metode/bank.

### Pengaman webhook
Hanya event **pembayaran nyata** yang diproses (ada `payment_id` / status sukses).
Callback **FVA created** (saat VA dibuat, belum dibayar) **diabaikan** — jadi jangan
mendaftarkan FVA created di Xendit (cukup **FVA paid** + **QR payment**).

---

## 5. Database (Supabase)

Tabel inti: `chapters`, `members`, `fee_settings`, `invoices`, `payments`,
`invoice_audit_log`, `app_settings`.

### Kolom Xendit (migration `0002_xendit_self_payment.sql`)

`invoices`: `payment_provider`, `xendit_external_id`, `xendit_payment_id`,
`xendit_payment_method` (`va`|`qris`), `xendit_va_bank`, `xendit_va_number`,
`xendit_qris_string`, `xendit_payment_status` (`PENDING`|`PAID`|`EXPIRED`), `xendit_expires_at`.

`payments`: `xendit_payment_id`, `xendit_status`.

### `app_settings` (key-value runtime)

| key | nilai | fungsi |
|---|---|---|
| `self_payment_mode` | `'true'` / `'false'` | aktifkan gateway Xendit |
| `invoice_draft_days_before` | angka (default 30) | buat draft H-N sebelum renewal |
| `invoice_due_days_after` | angka (default 30) | jatuh tempo H+N setelah terbit |
| `bni_vm_token` | string | token BNI VM (jika diset di DB) |

> Setelah perubahan DDL, jalankan `select pg_notify('pgrst', 'reload schema');`.

---

## 6. Edge Functions (Supabase, Deno)

| Function | Trigger | Fungsi | verify_jwt |
|---|---|---|---|
| `xendit-create-payment` | dipanggil app | Buat VA / QRIS via Xendit, simpan ke invoice | ya |
| `xendit-webhook` | callback Xendit | Auto-Lunas + renewal +1 tahun | **no** (`--no-verify-jwt`) |
| `auto-create-invoices` | pg_cron harian | Buat draft renewal otomatis (H-N) | ya |

### Secrets (server-side, tidak pernah di browser)
```bash
npx supabase secrets set XENDIT_SECRET_KEY=xnd_development_...
npx supabase secrets set XENDIT_CALLBACK_TOKEN=...
```

### Deploy
```bash
npx supabase functions deploy xendit-create-payment --project-ref <ref>
npx supabase functions deploy xendit-webhook --no-verify-jwt --project-ref <ref>
npx supabase functions deploy auto-create-invoices --project-ref <ref>
```

### Webhook URL (daftarkan di Xendit → Settings → Developers → Webhooks)
```
https://<project-ref>.supabase.co/functions/v1/xendit-webhook
```
Aktifkan untuk **Virtual Account paid (FVA paid)** & **QR Code payment**. **Jangan** FVA created.

---

## 7. Environment Variables

```
VITE_USE_MOCK=false                  # pakai backend Supabase
VITE_SUPABASE_URL=https://<ref>.supabase.co
VITE_SUPABASE_ANON_KEY=<anon key>    # publik, aman di browser
VITE_BNI_VM_URL=<url BNI VM>         # dev only (proxy Vite /api/bni-vm)
VITE_BNI_VM_TOKEN=<token>            # JANGAN deploy ke produksi publik
```

> **Keamanan:** Service Role key & Xendit Secret Key **tidak boleh** ada di frontend —
> hanya anon key. Secret hidup di Supabase Edge Function. `.env.local` di-gitignore.

---

## 8. Menjalankan & Deploy

### Lokal
```bash
npm install
npm run dev         # http://localhost:5173
npm run typecheck   # type-check
npm run build       # build produksi ke dist/
```

### Produksi (Vercel)
Env produksi (`VITE_USE_MOCK`, `VITE_SUPABASE_URL`, `VITE_SUPABASE_ANON_KEY`) diset di
Vercel; `vercel.json` mengatur SPA rewrite + header keamanan (CSP, HSTS, dll).

```bash
vercel --prod        # build + deploy
```

URL produksi (alias publik): **https://bni-finance-five.vercel.app**
Halaman pembayaran member: `https://bni-finance-five.vercel.app/pay/<invoice-id>`

> Sinkronisasi BNI VM **tidak berjalan di produksi** (mengandalkan proxy dev Vite).
> Untuk produksi, sync perlu dipindah ke Edge Function server-side.

---

## 9. Routing

| Path | Akses | Halaman |
|---|---|---|
| `/login` | publik | Login |
| `/pay/:id` | **publik** | Halaman pembayaran member |
| `/dashboard` | auth | Dashboard |
| `/invoices`, `/invoices/:id`, `/invoices/new` | auth | Invoice |
| `/members`, `/chapters`, `/payments` | auth | Data |
| `/urgent` | auth | Perlu Tindakan |
| `/settings`, `/settings/payment`, `/settings/sync` | auth | Pengaturan |

---

## 10. Catatan Operasional

- Invoice **overdue** otomatis di-set saat `due_date < hari ini` (di `list()`/`renewalDue()`).
- Renewal otomatis maju **+1 tahun** ke `period_end` saat pembayaran sukses.
- Logo bank di tombol VA = wordmark SVG terstilasi (`BankLogo`). Logo BNI = `BniLogo`.
  Untuk PNG resmi, taruh di `public/` dan ganti komponennya dengan `<img>`.
