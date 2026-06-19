# Feature: Notifikasi

**Route:** `/notifications`  
**Status:** `done`  
**Domain:** Feed notifikasi tagihan dan pembayaran untuk admin.

---

## User Stories

### US-01 Melihat notifikasi terbaru
> Sebagai admin, saya ingin melihat notifikasi tagihan overdue, renewal jatuh tempo, dan pembayaran diterima, agar tidak ada yang terlewat.

**Acceptance Criteria:**
- [ ] Daftar notifikasi dengan: icon per tipe, judul, deskripsi, timestamp relatif
- [ ] Tipe notifikasi: `overdue` (merah), `due-soon` (amber), `payment` (hijau)
- [ ] Label kontekstual: "Telat N hari", "N hari lagi", formatted datetime
- [ ] Klik notifikasi → navigasi ke halaman terkait (invoice detail, dll)
- [ ] Empty state jika tidak ada notifikasi

### US-02 Badge unread di ikon notifikasi
> Sebagai admin, saya ingin melihat badge jumlah notifikasi belum dibaca di ikon lonceng, agar tahu ada notifikasi baru tanpa harus membuka halamannya.

**Acceptance Criteria:**
- [ ] Badge merah dengan count tampil di ikon notifikasi di topbar/sidebar
- [ ] Badge hilang setelah halaman notifikasi dibuka (mark all read otomatis)
- [ ] Highlight visual pada item yang belum dibaca saat halaman pertama dibuka

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `items` | `AppNotification[]` | Dari `NotificationsContext` |
| `highlight` | `Set<string> \| null` | Item yang unread saat halaman dibuka |

## NotificationType
```ts
type NotificationType = 'overdue' | 'due-soon' | 'payment'
```

## Catatan
- Notifikasi di-generate dari data invoice (overdue, due-soon) dan payments (payment received)
- Tidak persisten ke database — dibuild dari data yang sudah ada saat context load
