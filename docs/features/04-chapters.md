# Feature: Chapter

**Route:** `/chapters`  
**Status:** `done`  
**Domain:** Data chapter/cabang BNI hasil sinkronisasi.

---

## User Stories

### US-01 Melihat daftar chapter beserta statistik
> Sebagai admin, saya ingin melihat semua chapter dengan jumlah member dan outstanding amount-nya, agar bisa memantau performa tiap chapter.

**Acceptance Criteria:**
- [ ] Card grid (responsive) menampilkan per chapter: nama, area, kota, jumlah member, outstanding amount
- [ ] Outstanding amount = total amount invoice dengan status `sent` atau `overdue`
- [ ] Klik outstanding amount → navigasi ke invoice list filter chapter + outstanding
- [ ] Klik "Lihat member" → navigasi ke member list filter chapter
- [ ] Filter kota: dropdown unique kota dari data
- [ ] Count total chapter ditampilkan

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `chapters` | `Chapter[]` | Daftar chapter |
| `members` | `MemberWithChapter[]` | Untuk hitung member count per chapter |
| `invoices` | `InvoiceWithRelations[]` | Untuk hitung outstanding per chapter |
| `city` | `string` | Filter kota |

## Catatan
- Data chapter adalah **read-only** — sync dari BNI VM via `/settings/sync`
