# Feature: Pengaturan

**Route:** `/settings`, `/settings/payment`, `/settings/sync`  
**Status:** `done`  
**Domain:** Konfigurasi biaya, metode pembayaran, dan sinkronisasi data.

---

## User Stories

### US-01 Mengatur biaya keanggotaan
> Sebagai admin, saya ingin mengatur nominal biaya pendaftaran dan renewal, agar invoice baru otomatis menggunakan biaya yang benar.

**Acceptance Criteria:**
- [ ] Input nominal pendaftaran (registration fee)
- [ ] Input nominal renewal fee
- [ ] Textarea catatan/keterangan (opsional)
- [ ] Preview kartu: tampilan nominal untuk tiap tipe
- [ ] Tombol Simpan aktif hanya jika ada perubahan
- [ ] Timestamp "Terakhir diperbarui" setelah simpan
- [ ] Perubahan hanya berlaku untuk invoice yang dibuat setelah perubahan

### US-02 Mengatur timing pembuatan invoice
> Sebagai admin, saya ingin mengatur berapa hari sebelum renewal invoice draft dibuat, dan berapa hari setelah terbit invoice jatuh tempo, agar siklus penagihan sesuai kebijakan chapter.

**Acceptance Criteria:**
- [ ] Input: Draft H-N sebelum renewal (default 30)
- [ ] Input: Jatuh tempo H+N setelah terbit (default 30)
- [ ] Hanya tampil jika `VITE_USE_MOCK=false` (Supabase mode)
- [ ] Simpan ke `app_settings` tabel (key-value)
- [ ] Info konteks: konfigurasi ini mempengaruhi invoice baru saja

### US-03 Mengatur mode pembayaran (Xendit vs Paper.id)
> Sebagai admin, saya ingin mengaktifkan atau menonaktifkan Self Payment Mode, agar bisa beralih antara payment gateway Xendit dan integrasi Paper.id.

**Acceptance Criteria:**
- [ ] Toggle switch: Self Payment Mode ON/OFF
- [ ] Mode ON: member bayar mandiri via Xendit (VA/QRIS), link `/pay/:id` aktif
- [ ] Mode OFF: integrasi Paper.id, link Paper.id tersimpan di invoice
- [ ] Kartu perbandingan mode: fitur dan keterbatasan tiap mode
- [ ] Toggle rollback jika save gagal
- [ ] Info warning tentang konfigurasi server (XENDIT_SECRET_KEY di Edge Function)
- [ ] Hanya tersedia jika `VITE_USE_MOCK=false`

### US-04 Sinkronisasi data dari BNI Visitor Management
> Sebagai admin, saya ingin men-trigger sync data member dan chapter dari BNI VM, agar data di sistem selalu up-to-date.

**Acceptance Criteria:**
- [ ] Informasi sumber data: BNI Visitor Management (nama + status)
- [ ] Konfigurasi token (jika Supabase mode): input Base URL + API Token (password field + show/hide)
- [ ] Tombol simpan konfigurasi token
- [ ] Card sync Member: jumlah record, last sync timestamp, tombol Sync
- [ ] Card sync Chapter: jumlah record, last sync timestamp, tombol Sync
- [ ] Loading state saat sync berjalan
- [ ] Hasil sync: count baru, timestamp diperbarui
- [ ] Peringatan: sync hanya berjalan di dev (proxy Vite), tidak di produksi

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `fees` | `FeeSettings` | Biaya keanggotaan |
| `selfPayment` | `boolean` | Mode Xendit ON/OFF |
| `draftDaysBefore` | `number` | Timing draft invoice |
| `dueDaysAfter` | `number` | Timing jatuh tempo |
| `token, apiUrl` | `string` | Konfigurasi BNI VM |
| `syncing` | `'members' \| 'chapters' \| null` | Proses sync aktif |

## app_settings keys

| Key | Default | Fungsi |
|---|---|---|
| `self_payment_mode` | `'false'` | Aktifkan Xendit |
| `invoice_draft_days_before` | `'30'` | Draft H-N sebelum renewal |
| `invoice_due_days_after` | `'30'` | Jatuh tempo H+N setelah terbit |
| `bni_vm_token` | — | Token BNI VM |
