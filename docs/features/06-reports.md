# Feature: Laporan Keuangan

**Route:** `/reports`  
**Status:** `done`  
**Domain:** Analitik keuangan berbasis periode waktu.

---

## User Stories

### US-01 Melihat laporan keuangan per periode
> Sebagai admin, saya ingin melihat ringkasan keuangan untuk periode tertentu, agar bisa mengevaluasi kinerja penagihan dan penerimaan.

**Acceptance Criteria:**
- [ ] Preset periode: Bulan Ini, Bulan Lalu, Tahun Ini, Semua, Custom
- [ ] Custom range: input tanggal dari–sampai (muncul hanya saat preset = Custom)
- [ ] KPI cards: Total Ditagih, Total Diterima, Outstanding, Collection Rate (%)
- [ ] Semua KPI mengikuti periode yang dipilih

### US-02 Melihat tren bulanan
> Sebagai admin, saya ingin melihat grafik tren invoice diterbitkan vs dibayar per bulan, agar bisa mengidentifikasi pola pembayaran.

**Acceptance Criteria:**
- [ ] Bar chart: sumbu X = bulan, dua bar per bulan (Diterbitkan vs Diterima)
- [ ] Data 12 bulan terakhir atau sesuai range
- [ ] Label nominal pada bar
- [ ] Loading skeleton selama data dimuat

### US-03 Melihat breakdown per chapter
> Sebagai admin, saya ingin melihat kinerja tiap chapter dalam periode, agar bisa mengidentifikasi chapter dengan collection rate rendah.

**Acceptance Criteria:**
- [ ] Tabel: chapter, jumlah invoice, total ditagih, total diterima, outstanding, collection rate (%)
- [ ] Klik chapter row → navigasi ke invoice list filter chapter
- [ ] Sortable per kolom (backlog)

### US-04 Melihat breakdown per tipe & metode pembayaran
> Sebagai admin, saya ingin melihat porsi registration vs renewal dan metode pembayaran yang digunakan, agar bisa memahami komposisi penerimaan.

**Acceptance Criteria:**
- [ ] Donut chart tipe: registration vs renewal (count & %)
- [ ] Daftar metode pembayaran: nama metode, jumlah transaksi, total amount

### US-05 Export laporan
> Sebagai admin, saya ingin mengekspor laporan dalam periode aktif ke CSV dan PDF.

**Acceptance Criteria:**
- [ ] Export CSV: data per chapter (kolom lengkap)
- [ ] Export PDF: dokumen laporan berlabel BNI dengan ringkasan KPI + tabel chapter

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `invoices` | `InvoiceWithRelations[]` | Semua invoice untuk kalkulasi |
| `payments` | `PaymentWithInvoice[]` | Semua payment untuk kalkulasi |
| `preset` | `Preset` | Periode aktif |
| `customFrom, customTo` | `string` | Hanya aktif saat preset = custom |

## Computed
- `range` — dari preset atau custom date → `{ from: string, to: string }`
- `inRange(date)` — filter helper per invoice/payment
- `report` — kalkulasi lengkap: KPI, per-chapter, per-type, monthly trend, payment methods
