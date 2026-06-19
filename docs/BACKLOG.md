# BNI Finance Hub — Master Backlog

> **Source of truth** untuk semua user story, task, dan acceptance criteria.  
> Setiap fitur punya file detail di `docs/features/`.  
> Setiap epic punya file di `docs/epics/`.

---

## Status Legenda

| Status | Makna |
|---|---|
| `done` | Sudah live di produksi |
| `on-progress` | Sedang dikerjakan |
| `backlog` | Direncanakan, belum dimulai |
| `blocked` | Butuh keputusan / dependency eksternal |

---

## Feature Domains

| # | Domain | File | Status |
|---|---|---|---|
| 01 | Dashboard | [docs/features/01-dashboard.md](features/01-dashboard.md) | `done` |
| 02 | Invoice | [docs/features/02-invoice.md](features/02-invoice.md) | `done` |
| 03 | Member | [docs/features/03-members.md](features/03-members.md) | `done` |
| 04 | Chapter | [docs/features/04-chapters.md](features/04-chapters.md) | `done` |
| 05 | Pembayaran | [docs/features/05-payments.md](features/05-payments.md) | `done` |
| 06 | Laporan | [docs/features/06-reports.md](features/06-reports.md) | `done` |
| 07 | Notifikasi | [docs/features/07-notifications.md](features/07-notifications.md) | `done` |
| 08 | Profil | [docs/features/08-profile.md](features/08-profile.md) | `done` |
| 09 | Pengaturan | [docs/features/09-settings.md](features/09-settings.md) | `done` |
| 10 | Urgent | [docs/features/10-urgent.md](features/10-urgent.md) | `done` |
| 11 | Public Payment | [docs/features/11-public-payment.md](features/11-public-payment.md) | `done` |

---

## Backlog Tasks (Enhancement / Bug)

> Tambahkan task baru di sini. Saat task dikerjakan, buatkan epic di `docs/epics/`.

| ID | Task | Domain | Prioritas | Status |
|---|---|---|---|---|
| T-001 | Pagination di tabel invoice | Invoice | Medium | `done` |
| T-002 | Pagination di tabel member | Member | Medium | `done` |
| T-003 | Pagination di tabel pembayaran | Payments | Medium | `backlog` |
| T-004 | Sortable column di semua tabel | Global | Low | `backlog` |
| T-005 | BNI VM sync via Edge Function (produksi) | Settings | High | `backlog` |
| T-006 | Email notifikasi invoice ke member | Invoice | Medium | `backlog` |
| T-007 | Reminder otomatis overdue via WhatsApp API | Urgent | Medium | `backlog` |
| T-008 | Dashboard drill-down per chapter | Dashboard | Low | `backlog` |
| T-009 | Expiry VA / QRIS reminder di PublicPaymentPage | Public Payment | Low | `backlog` |
| T-010 | Custom domain Vercel | DevOps | Low | `backlog` |
| T-011 | Multi-role auth (chapter admin vs national admin) | Auth | High | `backlog` |
| T-012 | Dark mode | UI | Low | `backlog` |

---

## Cara Menggunakan Backlog Ini

### Memulai task baru
```
/agentic-start "<deskripsi task>"
```
Agent akan:
1. Mapping task ke feature domain di `docs/features/`
2. Mencari / membuat epic di `docs/epics/`
3. Scope file yang terdampak
4. Implementasi + gate (typecheck + build)
5. Update backlog & epic automation log

### Menambah task baru ke backlog
Edit bagian **Backlog Tasks** di file ini, tambah baris baru dengan ID berikutnya.

### Menandai task selesai
Ubah status di tabel Backlog Tasks ke `done` dan pastikan epic di `docs/epics/` juga diperbarui.
