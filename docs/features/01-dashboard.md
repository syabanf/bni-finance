# Feature: Dashboard

**Route:** `/`  
**Status:** `done`  
**Domain:** Ringkasan eksekutif keuangan chapter BNI.

---

## User Stories

### US-01 Melihat KPI keuangan
> Sebagai admin, saya ingin melihat 4 KPI utama (Total Invoice, Sudah Dibayar, Outstanding, Overdue) beserta trend-nya, agar bisa mengetahui kesehatan keuangan chapter secara cepat.

**Acceptance Criteria:**
- [ ] Terdapat 4 KPI card: Total Invoice, Sudah Dibayar, Outstanding, Overdue
- [ ] Setiap card menampilkan count & total amount
- [ ] Terdapat indikator trend (naik/turun) dibanding periode sebelumnya
- [ ] KPI card bisa diklik → navigasi ke invoice list dengan filter status sesuai
- [ ] Loading skeleton tampil selama data dimuat

### US-02 Melihat distribusi status pembayaran
> Sebagai admin, saya ingin melihat donut chart distribusi status pembayaran, agar bisa memvisualisasikan proporsi invoice lunas vs belum.

**Acceptance Criteria:**
- [ ] Donut chart tampil dengan warna per status (draft, sent, paid, overdue, cancelled)
- [ ] Klik segment donut → filter invoice list berdasarkan status yang diklik
- [ ] Legend menampilkan label status + jumlah

### US-03 Melihat statistik per chapter
> Sebagai admin, saya ingin melihat breakdown invoice per chapter dalam tabel, agar bisa membandingkan performa antar chapter.

**Acceptance Criteria:**
- [ ] Tabel menampilkan: chapter name, total invoice, paid, outstanding, overdue, total amount
- [ ] Klik baris chapter → navigasi ke invoice list filter chapter + status
- [ ] Kolom sortable (opsional backlog)

### US-04 Melihat invoice terbaru
> Sebagai admin, saya ingin melihat 6 invoice terbaru di dashboard, agar bisa langsung tahu aktivitas terkini.

**Acceptance Criteria:**
- [ ] Daftar 6 invoice terbaru dengan: nomor, member, status badge, amount, due date
- [ ] Link "Lihat semua" → navigasi ke invoice list
- [ ] Klik row → navigasi ke invoice detail

### US-05 Peringatan renewal & urgent
> Sebagai admin, saya ingin melihat notifikasi singkat jumlah member yang akan renewal dan invoice overdue, agar bisa segera mengambil tindakan.

**Acceptance Criteria:**
- [ ] Banner/callout tampil jika ada overdue atau renewal due dalam 30 hari
- [ ] Menampilkan jumlah item urgent
- [ ] Link ke halaman Urgent (`/urgent`)

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `summary` | `DashboardSummary` | KPI, donut, monthly, chapterStats |
| `recent` | `InvoiceWithRelations[]` | 6 invoice terbaru |

## Komponen Terkait
- `DashboardPage.tsx`
- `StatCard`, `DonutChart`, `SummaryCard` (ui primitives)
