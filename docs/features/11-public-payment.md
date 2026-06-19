# Feature: Halaman Pembayaran Publik

**Route:** `/pay/:id`  
**Status:** `done`  
**Domain:** Halaman pembayaran mandiri untuk member (tanpa login).

---

## User Stories

### US-01 Melihat informasi invoice
> Sebagai member yang menerima link pembayaran, saya ingin melihat detail invoice yang harus saya bayar, agar saya yakin tentang tagihan yang saya terima.

**Acceptance Criteria:**
- [ ] Halaman accessible tanpa login
- [ ] Tampil preview invoice (identik dengan versi PDF)
- [ ] Informasi: nomor invoice, nama member, chapter, tipe, nominal, periode, due date
- [ ] Status-aware rendering:
  - Draft: banner warning "Invoice belum diterbitkan"
  - Sent/Overdue: tampilkan opsi pembayaran
  - Paid: screen sukses "Pembayaran Berhasil" + periode aktif
  - Cancelled: screen "Invoice Dibatalkan"
  - Not found: screen "Invoice Tidak Ditemukan"

### US-02 Download invoice PDF
> Sebagai member, saya ingin mendownload invoice sebagai PDF, agar bisa menyimpan bukti tagihan.

**Acceptance Criteria:**
- [ ] Tombol "Download Invoice (PDF)" tersedia di bawah preview
- [ ] Membuka dialog print browser (Save as PDF)
- [ ] Invoice PDF identik dengan tampilan di layar

### US-03 Bayar via Xendit Virtual Account
> Sebagai member, saya ingin memilih bank dan mendapatkan nomor Virtual Account, agar bisa transfer melalui m-banking saya.

**Acceptance Criteria:**
- [ ] Hanya tampil jika Self Payment Mode ON
- [ ] Step 1: pilih bank (BCA / BNI / Mandiri / BRI) dengan logo
- [ ] Step 2: setelah bank dipilih → buat VA via Xendit
- [ ] Nomor VA tampil dengan tombol salin
- [ ] Informasi ekspiry VA
- [ ] Tombol "Ganti metode / bank" (warna merah) → kembali ke picker awal
- [ ] Loading state saat VA sedang dibuat

### US-04 Bayar via QRIS
> Sebagai member, saya ingin membayar menggunakan QRIS, agar bisa scan dengan aplikasi m-banking atau dompet digital saya.

**Acceptance Criteria:**
- [ ] Opsi QRIS tersedia di picker (selain bank VA)
- [ ] QRIS dinonaktifkan (disabled + tooltip) jika amount > Rp 10.000.000
- [ ] Jika dipilih: tampilkan QR code yang bisa discan
- [ ] Tombol "Ganti metode / bank" untuk kembali ke picker

### US-05 Status pembayaran otomatis (auto-refresh)
> Sebagai member, saya ingin halaman ini otomatis mendeteksi pembayaran saya, tanpa perlu refresh manual.

**Acceptance Criteria:**
- [ ] Auto-poll setiap 7 detik jika invoice masih payable (sent/overdue) dan belum paid
- [ ] Saat invoice berubah ke `paid`: tampilkan screen "Pembayaran Berhasil"
- [ ] Tidak ada flicker loader saat auto-poll (loader penuh hanya di load pertama)
- [ ] Polling berhenti setelah pembayaran berhasil

### US-06 Fallback Paper.id (mode OFF)
> Sebagai member dengan mode Paper.id, saya ingin diarahkan ke halaman pembayaran Paper.id, agar bisa menyelesaikan pembayaran.

**Acceptance Criteria:**
- [ ] Jika Self Payment Mode OFF: tampilkan kartu Paper.id dengan tombol "Bayar via Paper.id"
- [ ] Tombol membuka Paper.id payment URL di tab baru

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `invoice` | `InvoiceWithRelations \| null` | Data invoice dari URL param `:id` |
| `selfPayment` | `boolean` | Mode aktif |
| `forced` | `'success' \| 'failed' \| null` | Dari query param `?status=` |

## Alur Lengkap (Self Payment Mode ON)

```
Member buka /pay/:id
  1. Preview invoice ditampilkan
  2. Download PDF (opsional)
  3. Pilih metode → VA (pilih bank) atau QRIS
       └─ createXenditPayment() → Edge Function → Xendit API
  4. Tampilkan VA number atau QR code
  5. Member bayar di m-banking / scan QRIS
       └─ Xendit callback → xendit-webhook Edge Function
            ├─ Verifikasi x-callback-token
            ├─ Update invoice → paid
            ├─ Insert ke payments
            ├─ Audit log
            └─ Update members.renewal_date
  6. Auto-poll 7s → deteksi paid → screen sukses
```

## Catatan Keamanan
- Halaman menggunakan **anon key** Supabase (bukan service role)
- Secret key Xendit **tidak pernah** expose ke client
- RLS Supabase: invoice bisa dibaca anon (public read untuk `/pay`)
