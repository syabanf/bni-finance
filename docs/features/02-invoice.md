# Feature: Invoice

**Route:** `/invoices`, `/invoices/new`, `/invoices/:id`, `/invoices/renewal-due`  
**Status:** `done`  
**Domain:** Pembuatan, penerbitan, dan pengelolaan invoice keanggotaan.

---

## User Stories

### US-01 Melihat daftar invoice dengan filter
> Sebagai admin, saya ingin memfilter invoice berdasarkan status, tipe, chapter, dan tanggal jatuh tempo, agar bisa fokus pada invoice yang relevan.

**Acceptance Criteria:**
- [ ] Filter: status tab (all/draft/outstanding/overdue/paid/cancelled), tipe dropdown, chapter dropdown, due date range
- [ ] Search by nomor invoice atau nama member
- [ ] Summary cards (total, outstanding, overdue, lunas) mengikuti filter aktif
- [ ] Count per status tab diperbarui mengikuti filter lain (bukan status)
- [ ] Filter bisa dikombinasikan
- [ ] Filter tersedia via query param (`?status=overdue&type=renewal&chapter=id`)
- [ ] Tombol reset/clear untuk date range filter

### US-02 Membuat invoice baru
> Sebagai admin, saya ingin membuat invoice pendaftaran atau renewal untuk member tertentu, agar tagihan keanggotaan tercatat di sistem.

**Acceptance Criteria:**
- [ ] Pilih tipe: registration atau renewal
- [ ] Daftar member terupdate sesuai tipe:
  - Registration: member yang belum ada invoice aktif
  - Renewal: semua member
- [ ] Amount otomatis terisi dari fee settings
- [ ] Periode otomatis dihitung: periodStart = hari ini (atau period_end invoice terakhir), periodEnd = +1 tahun
- [ ] Preview ringkasan sebelum submit (member, tipe, nominal, periode)
- [ ] Dua aksi: "Simpan Draft" atau "Buat & Terbitkan"
- [ ] Redirect ke invoice detail setelah berhasil

### US-03 Menerbitkan invoice (draft → sent)
> Sebagai admin, saya ingin menerbitkan invoice draft, agar member mendapat tagihan dan link pembayaran bisa dibagikan.

**Acceptance Criteria:**
- [ ] Tombol "Terbitkan" hanya muncul jika status = `draft`
- [ ] Setelah terbit: status berubah ke `sent`, due_date diset (today + N hari dari settings)
- [ ] Mode Xendit ON: tidak ada integrasi Paper.id, link `/pay/:id` tersedia
- [ ] Mode Xendit OFF: simulasi Paper.id ID + URL tersimpan di invoice
- [ ] Audit log entry `sent` tercatat
- [ ] Konfirmasi dialog sebelum aksi

### US-04 Melihat detail invoice
> Sebagai admin, saya ingin melihat semua detail invoice termasuk audit trail, agar bisa melacak riwayat perubahan status.

**Acceptance Criteria:**
- [ ] Tampil: nomor, tipe, amount, status, due date, periode, currency, timestamps
- [ ] Info member: nama, email, phone, chapter, kota
- [ ] Audit trail timeline (urutan kronologis, actor name, notes)
- [ ] Riwayat pembayaran (payments received) dengan method, amount, proof link
- [ ] Cancel reason tampil jika status = `cancelled`

### US-05 Preview & Download Invoice PDF
> Sebagai admin, saya ingin melihat preview dan mendownload invoice sebagai PDF, agar bisa mencetak atau mengirimkan dokumen resmi ke member.

**Acceptance Criteria:**
- [ ] Tombol "Preview Invoice" membuka modal dengan tampilan invoice yang identik dengan versi cetak
- [ ] Tombol "Download PDF" membuka dialog print browser (Save as PDF)
- [ ] Header invoice menampilkan logo BNI + "BNI Indonesia"
- [ ] Invoice berisi: nomor, nama member, chapter, amount, periode, due date
- [ ] Tampilan konsisten antara preview modal dan hasil PDF

### US-06 Berbagi link pembayaran
> Sebagai admin, saya ingin menyalin link pembayaran dan mengirimkannya via WhatsApp, agar member bisa langsung membayar.

**Acceptance Criteria:**
- [ ] Link `/pay/:id` tampil di detail invoice jika status = `sent` atau `overdue`
- [ ] Tombol "Salin Link" → copy ke clipboard dengan feedback visual (teks berubah "Tersalin!")
- [ ] Tombol "Kirim via WA" → buka WhatsApp Web/app dengan pesan template + link
- [ ] Hanya tampil saat self payment mode ON atau status memungkinkan

### US-07 Tandai Lunas manual
> Sebagai admin, saya ingin menandai invoice sebagai lunas secara manual, agar bisa mencatat pembayaran offline atau transfer langsung.

**Acceptance Criteria:**
- [ ] Tombol "Tandai Lunas" hanya tersedia untuk status `sent` atau `overdue`
- [ ] Dialog konfirmasi sebelum aksi
- [ ] Invoice status berubah ke `paid`, `paid_at` dan `paid_amount` tersimpan
- [ ] Payment record otomatis diinsert dengan method `paper_id`
- [ ] Audit log entry `paid` tercatat

### US-08 Catat pembayaran manual dengan bukti
> Sebagai admin, saya ingin mencatat pembayaran manual lengkap dengan nominal, metode, tanggal, catatan, dan upload bukti transfer, agar riwayat pembayaran terdokumentasi dengan baik.

**Acceptance Criteria:**
- [ ] Form: nominal (editable), tanggal bayar, metode pembayaran, catatan (opsional), upload bukti (opsional)
- [ ] Upload file bukti ke Supabase Storage bucket `payment-proofs`
- [ ] Invoice berubah ke `paid` setelah disimpan
- [ ] Payment record tersimpan dengan: amount, paid_at, method, proof_url, note
- [ ] Audit log entry `paid` dengan keterangan metode + catatan

### US-09 Membatalkan invoice
> Sebagai admin, saya ingin membatalkan invoice yang salah dibuat, agar tidak mengganggu laporan keuangan.

**Acceptance Criteria:**
- [ ] Tombol "Batalkan" tersedia selama status bukan `paid` atau `cancelled`
- [ ] Modal input alasan pembatalan (required)
- [ ] Invoice status berubah ke `cancelled`, `cancelled_at`, `cancel_reason`, `cancelled_by` tersimpan
- [ ] Audit log entry `cancelled` dengan alasan tercatat

### US-10 Bulk kirim invoice
> Sebagai admin, saya ingin mengirim atau membagikan beberapa invoice sekaligus, agar efisien dalam mengelola tagihan massal.

**Acceptance Criteria:**
- [ ] Checkbox select per row, select all
- [ ] Bulk action bar muncul saat ada yang dipilih: count + total amount
- [ ] Mode Xendit OFF: bulk send ke Paper.id (draft → sent); bulk resend (sent/overdue)
- [ ] Mode Xendit ON: bulk share via Email atau WhatsApp
- [ ] Progress/loading state selama proses bulk
- [ ] Toast notifikasi sukses/gagal

### US-11 Export invoice ke CSV dan PDF
> Sebagai admin, saya ingin mengekspor data invoice yang sedang difilter ke CSV atau PDF, agar bisa dianalisis atau dilaporkan.

**Acceptance Criteria:**
- [ ] Tombol Export CSV: download file CSV dengan BOM UTF-8 (agar rapi di Excel)
- [ ] Tombol Export PDF: buka dokumen PDF berlabel BNI siap cetak
- [ ] Export menggunakan data yang sedang difilter (bukan semua data)
- [ ] Kolom CSV: nomor, member, chapter, tipe, nominal, status, jatuh tempo, lunas pada

### US-12 Daftar renewal due (akan habis masa berlaku)
> Sebagai admin, saya ingin melihat daftar member yang keanggotaannya akan habis dalam 30 hari, agar bisa langsung generate invoice renewal secara massal.

**Acceptance Criteria:**
- [ ] Tabel menampilkan: member, chapter, tanggal berakhir, sisa hari (badge warna)
- [ ] Badge merah untuk sisa ≤ 7 hari atau sudah lewat
- [ ] Badge amber untuk sisa 8–30 hari
- [ ] Checkbox multi-select + select all
- [ ] Bulk generate renewal invoice → create draft sekaligus
- [ ] Total amount renewal ditampilkan di bulk action bar

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `invoices` | `InvoiceWithRelations[]` | Daftar invoice + relasi member/chapter |
| `invoice` | `InvoiceWithRelations` | Detail invoice aktif |
| `audit` | `AuditLogEntry[]` | Timeline perubahan status |
| `payments` | `PaymentWithInvoice[]` | Riwayat pembayaran per invoice |
| `selfPayment` | `boolean` | Mode Xendit ON/OFF |
| `selected` | `Set<string>` | Invoice terpilih untuk bulk action |

## Komponen Terkait
- `InvoiceListPage.tsx`, `InvoiceDetailPage.tsx`, `InvoiceNewPage.tsx`, `RenewalDuePage.tsx`
- `InvoiceTable.tsx`, `InvoicePreview.tsx`, `PaymentPanel.tsx`
- `invoiceDocument.ts` (PDF renderer)

## Aturan Bisnis
- `overdue`: status `sent` + `due_date < today` → auto-update saat `list()` dipanggil
- `due_date`: hari terbit + `invoice_due_days_after` (dari app_settings, default 30)
- `period_end`: `period_start` + 1 tahun
- QRIS dinonaktifkan jika `amount > 10.000.000` (limit Xendit)
