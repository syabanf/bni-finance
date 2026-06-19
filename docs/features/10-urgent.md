# Feature: Perlu Tindakan (Urgent)

**Route:** `/urgent`  
**Status:** `done`  
**Domain:** Tindakan prioritas: invoice overdue, renewal jatuh tempo, dan pendaftaran baru.

---

## User Stories

### US-01 Mengelola invoice overdue
> Sebagai admin, saya ingin melihat semua invoice yang sudah lewat jatuh tempo, agar bisa segera mengirim pengingat atau mencatatnya.

**Acceptance Criteria:**
- [ ] Tabel overdue: member, chapter, nominal, due date, badge "Telat N hari"
- [ ] Badge merah untuk semua overdue
- [ ] Per row aksi: Download PDF invoice, Kirim pengingat via WhatsApp, Lihat detail
- [ ] Count badge merah di judul section
- [ ] Filter chapter (berlaku global untuk semua section)
- [ ] Empty state jika tidak ada overdue

### US-02 Bulk generate renewal invoice
> Sebagai admin, saya ingin memilih beberapa member yang keanggotaannya akan berakhir dan membuat invoice renewal sekaligus, agar proses penagihan lebih efisien.

**Acceptance Criteria:**
- [ ] Tabel renewal due: member, chapter, tanggal berakhir, sisa hari (badge)
- [ ] Sub-section: "Sangat Mendesak (≤ 7 hari)" dan "Dalam 30 Hari"
- [ ] Badge merah untuk ≤ 7 hari atau sudah lewat, amber untuk 8–30 hari
- [ ] Checkbox multi-select per row + select all
- [ ] Bulk action bar: "Buat X Invoice Renewal", total amount
- [ ] Loading state selama generate invoice
- [ ] Setelah berhasil: hapus dari daftar atau tampil konfirmasi
- [ ] Link ke profil member per row

### US-03 Bulk generate invoice pendaftaran
> Sebagai admin, saya ingin melihat member yang belum punya invoice aktif dan membuat invoice pendaftaran mereka sekaligus.

**Acceptance Criteria:**
- [ ] Tabel eligible: member, chapter, tanggal bergabung, tombol aksi per row
- [ ] Checkbox multi-select + select all
- [ ] Bulk action bar: "Buat X Invoice Pendaftaran"
- [ ] Loading state selama generate
- [ ] Empty state jika tidak ada member eligible

### US-04 Filter per chapter (global)
> Sebagai admin, saya ingin memfilter semua section di halaman Urgent berdasarkan chapter, agar bisa fokus mengelola satu chapter tertentu.

**Acceptance Criteria:**
- [ ] Dropdown filter chapter di atas semua section
- [ ] Filter berlaku sekaligus untuk overdue, renewal due, dan eligible
- [ ] Total count urgent (badge di header) juga mengikuti filter chapter

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `overdue` | `InvoiceWithRelations[]` | Invoice overdue |
| `renewalDue` | `RenewalDueMember[]` | Member mendekati renewal |
| `eligible` | `MemberWithChapter[]` | Member eligible pendaftaran |
| `fees` | `FeeSettings` | Untuk kalkulasi bulk amount |
| `chapterId` | `string` | Filter global |
| `selectedRenewal, selectedEligible` | `Set<string>` | Multi-select per section |
| `generatingRenewal, generatingEligible` | `boolean` | Loading state bulk |

## Aturan Bisnis
- Member `eligible for registration`: belum ada invoice dengan status bukan `cancelled`
- `renewalDue`: member dengan `period_end` invoice terakhir dalam 30 hari ke depan
- Renewal invoice: `period_start = last_invoice.period_end + 1 hari`, `period_end = +1 tahun`
