# Feature: Pembayaran

**Route:** `/payments`  
**Status:** `done`  
**Domain:** Riwayat dan rekap seluruh pembayaran yang diterima.

---

## User Stories

### US-01 Melihat riwayat pembayaran
> Sebagai admin, saya ingin melihat semua pembayaran yang masuk dengan filter metode dan tanggal, agar bisa memonitor penerimaan kas.

**Acceptance Criteria:**
- [ ] Tabel: member, nomor invoice, nominal, metode pembayaran, waktu bayar
- [ ] Method badge per tipe (virtual_account, transfer bank, qris, paper_id, dll)
- [ ] Summary cards: Total Diterima (amount), Jumlah Transaksi (count), Bulan Ini (amount + count)
- [ ] Summary cards mengikuti filter aktif
- [ ] Search: nama member atau nomor invoice
- [ ] Filter metode: dropdown dari metode yang tersedia di data
- [ ] Filter tanggal bayar: range (dari–sampai) dengan reset
- [ ] Klik row → navigasi ke invoice detail
- [ ] Count display: "Menampilkan X dari Y pembayaran"

### US-02 Export pembayaran ke CSV dan PDF
> Sebagai admin, saya ingin mengekspor data pembayaran yang difilter, agar bisa dipakai untuk rekonsiliasi atau laporan.

**Acceptance Criteria:**
- [ ] Export CSV: BOM UTF-8, kolom: member, invoice, nominal, metode, waktu bayar
- [ ] Export PDF: dokumen berlabel BNI siap cetak
- [ ] Export menggunakan data terfilter

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `payments` | `PaymentWithInvoice[]` | Pembayaran + relasi invoice + member |
| `search` | `string` | Filter teks |
| `method` | `string` | Filter metode |
| `dueFrom, dueTo` | `string` | Filter range tanggal |

## Catatan
- Pembayaran bisa masuk via: Xendit webhook (otomatis), Paper.id webhook (simulasi), atau manual (admin)
- `proof_url` hanya ada untuk pembayaran manual yang disertai bukti upload
