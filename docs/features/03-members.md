# Feature: Member

**Route:** `/members`, `/members/:id`  
**Status:** `done`  
**Domain:** Data member hasil sinkronisasi dari BNI Visitor Management.

---

## User Stories

### US-01 Melihat daftar member dengan filter dan pagination
> Sebagai admin, saya ingin melihat daftar semua member dengan filter dan pagination, agar bisa dengan mudah menemukan member yang saya cari.

**Acceptance Criteria:**
- [ ] Tabel menampilkan: nama + company/bidang usaha, chapter, email, status badge, due date
- [ ] Search: nama, ID, atau email (live filter)
- [ ] Filter chapter: dropdown semua chapter
- [ ] Filter due date: range tanggal (dari–sampai) dengan tombol reset
- [ ] Toggle "Ada due date": sembunyikan member tanpa renewal date
- [ ] Summary cards atas (Total, Active, Pending, Inactive) berfungsi sebagai filter status
- [ ] Pagination 25 per halaman dengan kontrol prev/next dan nomor halaman
- [ ] Footer: "X–Y dari Z member"
- [ ] Reset ke halaman 1 otomatis saat filter berubah
- [ ] Query param `?chapter=id` support (dari link lain)

### US-02 Melihat detail member
> Sebagai admin, saya ingin melihat profil lengkap member beserta riwayat invoice-nya, agar bisa memahami status keanggotaan member secara menyeluruh.

**Acceptance Criteria:**
- [ ] Profil: avatar, nama, ID, status badge, email, phone, chapter, kota, tanggal bergabung
- [ ] Quick stats: Total Dibayar (amount), Outstanding (count)
- [ ] Periode aktif (period_end invoice terakhir non-cancelled)
- [ ] Tabel riwayat invoice: nomor, tipe, nominal, status badge, periode
- [ ] Klik invoice → navigasi ke invoice detail
- [ ] Tombol "Buat Invoice" → navigasi ke create invoice dengan member pre-selected

### US-03 Export data member
> Sebagai admin, saya ingin mengekspor data member yang sedang difilter ke CSV, agar bisa dilaporkan atau dianalisis.

**Acceptance Criteria:**
- [ ] Tombol Export CSV di header halaman
- [ ] Export menggunakan data yang sedang difilter (bukan semua)
- [ ] Kolom: Nama, ID, Chapter, Email, Telepon, Status, Due Date
- [ ] File CSV dengan BOM UTF-8

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `members` | `MemberWithChapter[]` | Data member dengan chapter |
| `chapters` | `Chapter[]` | Untuk dropdown filter |
| `search` | `string` | Live search |
| `chapterId` | `string` | Filter chapter |
| `memberStatus` | `MemberStatus \| 'all'` | Filter via summary cards |
| `hideNoDueDate` | `boolean` | Sembunyikan tanpa renewal date |
| `dueFrom, dueTo` | `string` | Date range filter |
| `page` | `number` | Halaman aktif (1-indexed) |

## Komponen Terkait
- `MemberListPage.tsx`, `MemberDetailPage.tsx`
- `Avatar`, `MemberStatusBadge` (ui)

## Catatan
- Data member adalah **read-only** — source of truth ada di BNI Visitor Management
- Update via sync manual di `/settings/sync`
- `renewalDate` di member = snapshot dari BNI VM; nilai aktual keanggotaan di `invoices.period_end`
