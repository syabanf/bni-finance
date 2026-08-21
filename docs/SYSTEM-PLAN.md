# BNI Finance System — Perencanaan Teknis

> Dokumen ini adalah rencana pembangunan sistem finance terpisah untuk BNI Grow Chapter Management.  
> Dibuat: 2026-06-15 | Status: Draft

---

## 1. Gambaran Umum

Sistem Finance BNI adalah aplikasi terpisah dari BNI Visitor Management (VM) yang mengelola:

- **Invoice Pendaftaran** — biaya yang dibayar visitor saat resmi bergabung menjadi member (berlaku 1 tahun)
- **Invoice Renewal** — biaya tahunan yang dibayar member aktif saat masa keanggotaan habis
- **Sinkronisasi data** — menarik data member dan chapter dari BNI VM via API
- **Integrasi Paper.id** — mengirim invoice ke Paper.id untuk mendapatkan link pembayaran, dan menerima notifikasi status pembayaran

### Prinsip Utama

| Prinsip | Penjelasan |
|---|---|
| Terpisah | Repo, domain, dan database berbeda dari BNI VM |
| Manual trigger | Invoice dibuat oleh national admin berdasarkan data jatuh tempo |
| Konfigurabel | Nominal biaya pendaftaran dan renewal bisa diatur per sistem |
| Audit trail | Semua perubahan status invoice tercatat lengkap |

---

## 2. Arsitektur Sistem

```
┌──────────────────────────────────────────────────────────┐
│               BNI Finance System                         │
│            finance.bni-vh.com (domain baru)              │
│                                                          │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────┐  │
│  │  Dashboard  │   │  Invoice     │   │  Settings    │  │
│  │  (overview) │   │  Management  │   │  (fee config)│  │
│  └─────────────┘   └──────────────┘   └──────────────┘  │
│                          │                               │
│              ┌───────────▼───────────┐                   │
│              │    Supabase Database   │                   │
│              │  (project terpisah)    │                   │
│              └───────────┬───────────┘                   │
│                          │                               │
└──────────────────────────┼───────────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
   ┌──────▼──────┐  ┌──────▼──────┐  ┌─────▼───────┐
   │  BNI VM API │  │  Paper.id   │  │  Webhook    │
   │  (pull data │  │  API        │  │  Receiver   │
   │  member &   │  │  (create    │  │  (payment   │
   │  chapter)   │  │  invoice +  │  │  callback   │
   │             │  │  pay link)  │  │  dari       │
   └─────────────┘  └─────────────┘  │  Paper.id)  │
                                     └─────────────┘
```

### Domain & Deployment

| Sistem | Domain | Repo |
|---|---|---|
| BNI Visitor Management | `*.bni-vh.com` | `bnigrowvisitor` (existing) |
| BNI Finance | `finance.bni-vh.com` | `bni-finance` (baru) |

---

## 3. Tech Stack

| Layer | Pilihan | Alasan |
|---|---|---|
| Framework | Next.js 15 (App Router) | Konsisten dengan BNI VM |
| Language | TypeScript | Type safety untuk data finansial |
| Database | Supabase (project baru) | Isolasi data finansial |
| Auth | Supabase Auth | Hanya national admin |
| Styling | Tailwind CSS v3 | Konsisten dengan BNI VM |
| Deployment | Vercel | Konsisten dengan BNI VM |
| PDF Invoice | `@react-pdf/renderer` atau `puppeteer` | Generate PDF invoice |
| HTTP Client | Native fetch | Untuk call BNI VM API & Paper.id |

---

## 4. Database Schema (Supabase)

### 4.1 Tabel Sync dari BNI VM

```sql
-- Mirror data chapter dari BNI VM (read-only, diperbarui saat sync)
CREATE TABLE chapters_sync (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  display_name  TEXT NOT NULL,
  area_name     TEXT,
  city_name     TEXT,
  synced_at     TIMESTAMPTZ DEFAULT NOW()
);

-- Mirror data member dari BNI VM (read-only, diperbarui saat sync)
CREATE TABLE members_sync (
  id            TEXT PRIMARY KEY,
  chapter_id    TEXT REFERENCES chapters_sync(id),
  name          TEXT NOT NULL,
  email         TEXT,
  phone         TEXT,
  status        TEXT,           -- 'active', 'inactive', dll dari BNI VM
  joined_date   DATE,
  synced_at     TIMESTAMPTZ DEFAULT NOW()
);
```

### 4.2 Konfigurasi Biaya

```sql
-- Konfigurasi nominal biaya (satu record global, bisa ditambah per chapter jika perlu)
CREATE TABLE fee_settings (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  registration_fee    NUMERIC(12,2) NOT NULL,   -- biaya pendaftaran visitor → member
  renewal_fee         NUMERIC(12,2) NOT NULL,   -- biaya renewal tahunan member
  currency            TEXT DEFAULT 'IDR',
  notes               TEXT,
  updated_by          UUID,                     -- admin yang mengubah
  updated_at          TIMESTAMPTZ DEFAULT NOW(),
  created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default (akan diisi admin saat setup awal)
-- INSERT INTO fee_settings (registration_fee, renewal_fee) VALUES (500000, 500000);
```

### 4.3 Invoice

```sql
CREATE TYPE invoice_type AS ENUM ('registration', 'renewal');
CREATE TYPE invoice_status AS ENUM ('draft', 'sent', 'paid', 'overdue', 'cancelled');

CREATE TABLE invoices (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi
  member_id             TEXT NOT NULL REFERENCES members_sync(id),
  chapter_id            TEXT NOT NULL REFERENCES chapters_sync(id),

  -- Tipe & nominal
  type                  invoice_type NOT NULL,
  amount                NUMERIC(12,2) NOT NULL,
  currency              TEXT DEFAULT 'IDR',

  -- Tanggal
  due_date              DATE NOT NULL,       -- = tanggal invoice diterbitkan
  period_start          DATE NOT NULL,       -- awal masa berlaku keanggotaan
  period_end            DATE NOT NULL,       -- period_start + 1 tahun

  -- Status
  status                invoice_status DEFAULT 'draft',

  -- Paper.id
  paper_id_invoice_id   TEXT,               -- ID invoice di Paper.id
  paper_id_invoice_url  TEXT,               -- URL halaman invoice Paper.id
  paper_id_payment_url  TEXT,               -- URL link pembayaran
  paper_id_sent_at      TIMESTAMPTZ,        -- kapan dikirim ke Paper.id

  -- Pembayaran
  paid_at               TIMESTAMPTZ,
  paid_amount           NUMERIC(12,2),

  -- Metadata
  notes                 TEXT,
  created_by            UUID,               -- admin yang membuat invoice
  cancelled_by          UUID,
  cancelled_at          TIMESTAMPTZ,
  cancel_reason         TEXT,

  created_at            TIMESTAMPTZ DEFAULT NOW(),
  updated_at            TIMESTAMPTZ DEFAULT NOW()
);

-- Index untuk performa
CREATE INDEX idx_invoices_member_id ON invoices(member_id);
CREATE INDEX idx_invoices_chapter_id ON invoices(chapter_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
```

### 4.4 Pembayaran (dari Webhook Paper.id)

```sql
CREATE TABLE payments (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  invoice_id            UUID NOT NULL REFERENCES invoices(id),

  amount                NUMERIC(12,2) NOT NULL,
  paid_at               TIMESTAMPTZ NOT NULL,
  payment_method        TEXT,               -- transfer, virtual_account, dll
  
  -- Data dari Paper.id
  paper_id_payment_id   TEXT UNIQUE,
  paper_id_status       TEXT,
  raw_webhook           JSONB,              -- raw payload dari Paper.id

  created_at            TIMESTAMPTZ DEFAULT NOW()
);
```

### 4.5 Audit Log

```sql
CREATE TABLE invoice_audit_log (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  invoice_id    UUID NOT NULL REFERENCES invoices(id),
  action        TEXT NOT NULL,         -- 'created', 'sent', 'paid', 'cancelled', dll
  old_status    invoice_status,
  new_status    invoice_status,
  actor_id      UUID,
  notes         TEXT,
  created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 5. Alur Kerja (User Flow)

### 5.1 Invoice Pendaftaran (Registration)

```
Admin login ke Finance System
  ↓
Halaman "Invoice Baru" → pilih type: Registration
  ↓
Sistem tarik daftar member dari BNI VM
  (filter: yang belum punya invoice registration aktif)
  ↓
Admin pilih member/visitor yang akan dibuatkan invoice
  ↓
Sistem auto-fill:
  - Nominal = fee_settings.registration_fee
  - due_date = hari ini
  - period_start = hari ini
  - period_end = hari ini + 365 hari
  ↓
Admin review → klik "Buat Invoice"
  ↓
Invoice tersimpan dengan status 'draft'
  ↓
Admin klik "Kirim ke Paper.id"
  ↓
Sistem call Paper.id API → dapat payment_url
  ↓
Invoice status berubah ke 'sent'
  ↓
Admin share payment_url ke member (manual via WA)
  ↓
Member bayar via Paper.id
  ↓
Paper.id kirim webhook ke Finance System
  ↓
Sistem update invoice status → 'paid', catat paid_at
```

### 5.2 Invoice Renewal

```
Admin buka halaman "Renewal Due"
  ↓
Sistem tampilkan daftar member yang:
  - Punya invoice registration/renewal sebelumnya
  - period_end <= 30 hari ke depan (atau sudah lewat)
  ↓
Admin pilih satu atau bulk select
  ↓
Sistem generate invoice renewal:
  - Nominal = fee_settings.renewal_fee
  - due_date = hari ini
  - period_start = period_end invoice sebelumnya + 1 hari
  - period_end = period_start + 365 hari
  ↓
(alur sama seperti registration di atas)
```

### 5.3 Penerimaan Webhook dari Paper.id

```
Paper.id POST ke: https://finance.bni-vh.com/api/webhooks/paper-id
  ↓
Sistem verifikasi signature webhook (API key / HMAC)
  ↓
Parse payload: paper_id_invoice_id, status, paid_at, amount
  ↓
Lookup invoice berdasarkan paper_id_invoice_id
  ↓
Jika status = 'paid':
  - Update invoices.status = 'paid'
  - Update invoices.paid_at
  - Insert ke tabel payments
  - Insert ke invoice_audit_log
  ↓
Return 200 OK ke Paper.id
```

---

## 6. API Design

### 6.1 API yang Dibutuhkan dari BNI VM (ditambahkan ke repo existing)

Endpoint baru di BNI VM, diproteksi dengan API key header `X-Finance-API-Key`:

```
GET  /api/finance/members
     → Mengembalikan daftar member aktif dengan:
       id, chapter_id, name, email, phone, status, joined_date

GET  /api/finance/chapters
     → Daftar chapter aktif:
       id, name, display_name, area_name, city_name

GET  /api/finance/members/:id
     → Detail satu member
```

### 6.2 API Internal Finance System

```
# Auth
POST /api/auth/login
POST /api/auth/logout

# Sync
POST /api/sync/members      → pull dari BNI VM, update members_sync
POST /api/sync/chapters     → pull dari BNI VM, update chapters_sync

# Fee Settings
GET  /api/settings/fees
PUT  /api/settings/fees

# Invoices
GET  /api/invoices                    → list dengan filter status, type, chapter, date
POST /api/invoices                    → buat invoice baru (draft)
GET  /api/invoices/:id
PUT  /api/invoices/:id/send           → kirim ke Paper.id
PUT  /api/invoices/:id/cancel
GET  /api/invoices/:id/audit          → riwayat perubahan status

# Renewal Due
GET  /api/invoices/renewal-due        → member yang mendekati/sudah jatuh tempo

# Payments
GET  /api/payments                    → list semua pembayaran

# Webhook dari Paper.id
POST /api/webhooks/paper-id           → endpoint callback Paper.id

# Dashboard stats
GET  /api/dashboard/summary           → total invoice, paid, outstanding, dll
```

---

## 7. Integrasi Paper.id

> Note: Akses API Paper.id belum diperoleh. Section ini berdasarkan Paper.id Open API docs.

### 7.1 Endpoint Paper.id yang Digunakan

```
# Membuat invoice
POST https://api.paper.id/v1/invoices
  Body: {
    customer_name, customer_email, customer_phone,
    items: [{ name, qty, price }],
    due_date,
    notes
  }
  Response: { invoice_id, invoice_url, payment_url }

# Cek status invoice (opsional, selain webhook)
GET https://api.paper.id/v1/invoices/:invoice_id
```

### 7.2 Webhook dari Paper.id

Paper.id akan POST ke endpoint kita saat pembayaran berhasil:

```json
{
  "event": "payment.success",
  "invoice_id": "PAPER_ID_INVOICE_ID",
  "paid_amount": 500000,
  "paid_at": "2026-06-15T10:30:00Z",
  "payment_method": "virtual_account",
  "payment_id": "PAY_XXXXX"
}
```

### 7.3 Environment Variables yang Dibutuhkan

```env
PAPER_ID_API_KEY=...
PAPER_ID_API_URL=https://api.paper.id/v1
PAPER_ID_WEBHOOK_SECRET=...       # untuk verifikasi signature webhook

BNI_VM_API_URL=https://bni-vh.com
BNI_VM_FINANCE_API_KEY=...        # API key dari BNI VM

SUPABASE_URL=...                  # project Supabase baru
SUPABASE_ANON_KEY=...
SUPABASE_SERVICE_ROLE_KEY=...
```

---

## 8. Halaman UI

### 8.1 Struktur Halaman

```
/login                        → Login national admin

/dashboard                    → Overview: total invoice, paid, outstanding,
                                grafik bulanan, daftar invoice terbaru

/invoices                     → List semua invoice (filter: status, type, chapter, bulan)
/invoices/new                 → Buat invoice baru
/invoices/renewal-due         → Daftar member yang mau renewal / sudah overdue
/invoices/:id                 → Detail invoice + audit log

/members                      → Daftar member hasil sync dari BNI VM
/members/:id                  → Riwayat invoice satu member

/settings                     → Konfigurasi nominal biaya (registration & renewal fee)
/settings/sync                → Trigger sync manual dari BNI VM
```

### 8.2 Dashboard Overview (KPI Cards)

| Kartu | Data |
|---|---|
| Total Invoice (bulan ini) | Count & jumlah Rp |
| Sudah Dibayar | Count & jumlah Rp |
| Outstanding | Count & jumlah Rp |
| Overdue | Count (merah) |
| Renewal Due (30 hari) | Count (kuning) |

---

## 9. Build Phases

### Phase 1 — Fondasi (Minggu 1-2)

- [ ] Init repo `bni-finance` (Next.js 15 + TypeScript + Tailwind)
- [ ] Setup Supabase project baru, jalankan migration schema
- [ ] Auth: login national admin (Supabase Auth)
- [ ] Tambahkan endpoint `/api/finance/*` di repo BNI VM (members, chapters)
- [ ] Implementasi sync: pull data dari BNI VM → simpan ke `members_sync` & `chapters_sync`
- [ ] Halaman: Login, Dashboard skeleton, Members list

### Phase 2 — Invoice Core (Minggu 3-4)

- [ ] Halaman buat invoice (registration & renewal)
- [ ] Halaman list invoice dengan filter
- [ ] Halaman detail invoice + audit log
- [ ] Halaman renewal-due (deteksi member yang jatuh tempo)
- [ ] Bulk select & bulk generate invoice
- [ ] Settings: konfigurasi registration_fee & renewal_fee
- [ ] Fee settings CRUD

### Phase 3 — Paper.id Integration (Minggu 5-6)

> Tunggu hingga API key Paper.id tersedia

- [ ] Implementasi `POST /api/paper-id/create-invoice` (kirim invoice ke Paper.id)
- [ ] Simpan `paper_id_invoice_id`, `paper_id_payment_url` ke database
- [ ] Tampilkan payment_url di halaman detail invoice (siap di-share manual via WA)
- [ ] Implementasi webhook receiver `POST /api/webhooks/paper-id`
- [ ] Verifikasi signature webhook
- [ ] Update status invoice saat pembayaran diterima
- [ ] Insert ke tabel `payments`

### Phase 4 — Polish & Laporan (Setelah Phase 3)

- [ ] Export laporan PDF / Excel
- [ ] Filter laporan per chapter, per periode
- [ ] Notifikasi in-app saat ada pembayaran masuk
- [ ] Tampilan riwayat pembayaran per member
- [ ] Deteksi otomatis invoice yang overdue (cron job / background task)

---

## 10. Keamanan

| Aspek | Implementasi |
|---|---|
| Auth | Supabase Auth, hanya role `national_admin` |
| API key BNI VM | Disimpan di env var, tidak pernah di-expose ke client |
| Webhook Paper.id | Verifikasi HMAC signature setiap request masuk |
| Database | RLS (Row Level Security) aktif di Supabase |
| Logging | Semua aksi invoice tercatat di `invoice_audit_log` |
| HTTPS | Wajib di semua endpoint (Vercel default) |

---

## 11. Item yang Masih Perlu Dikonfirmasi

| # | Item | Status |
|---|---|---|
| 1 | Akses API Paper.id (API key, docs endpoint, format webhook) | ⏳ Menunggu |
| 2 | Nominal default biaya registration & renewal | ❓ Perlu input |
| 3 | Format nomor invoice (contoh: INV-2026-001) | ❓ Perlu input |
| 4 | Apakah perlu notif WA otomatis di masa depan? | ❓ TBD |
| 5 | Apakah data member di-sync realtime atau manual trigger? | ✅ Manual trigger |
| 6 | Akses siapa saja ke Finance System? | ✅ National admin saja |

---

## 12. Struktur Repo (Preview)

```
bni-finance/
├── src/
│   ├── app/
│   │   ├── (auth)/
│   │   │   └── login/
│   │   ├── (dashboard)/
│   │   │   ├── layout.tsx
│   │   │   ├── dashboard/
│   │   │   ├── invoices/
│   │   │   │   ├── page.tsx          (list)
│   │   │   │   ├── new/page.tsx      (buat baru)
│   │   │   │   ├── renewal-due/page.tsx
│   │   │   │   └── [id]/page.tsx     (detail)
│   │   │   ├── members/
│   │   │   └── settings/
│   │   └── api/
│   │       ├── auth/
│   │       ├── sync/
│   │       ├── invoices/
│   │       ├── settings/
│   │       └── webhooks/
│   │           └── paper-id/
│   ├── components/
│   │   ├── layout/   (Sidebar, Topbar)
│   │   ├── invoices/ (InvoiceTable, InvoiceForm, dll)
│   │   └── ui/       (Skeleton, Badge, Modal)
│   ├── lib/
│   │   ├── supabase.ts
│   │   ├── paper-id.ts     (Paper.id API client)
│   │   ├── bni-vm.ts       (BNI VM API client)
│   │   └── invoice.ts      (business logic)
│   └── hooks/
│       └── useInvoices.ts
├── supabase/
│   └── migrations/         (schema SQL)
├── .env.local.example
└── README.md
```

---

## 13. Ketergantungan Antar Phase

```
Phase 1 (Fondasi)
    │
    ├── Butuh: endpoint /api/finance/* di BNI VM ditambahkan
    │
    ▼
Phase 2 (Invoice Core)
    │
    ├── Bisa jalan tanpa Paper.id (simpan sebagai draft)
    │
    ▼
Phase 3 (Paper.id) ← BLOCKER: butuh API key Paper.id
    │
    ▼
Phase 4 (Polish)
```

---

*Dokumen ini akan diperbarui seiring progress pembangunan.*
*Kontak teknis: Ilham Kurniawan | BNI Grow Platform*
