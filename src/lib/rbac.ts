import type { UserRole } from '@/types'

/**
 * RBAC berbasis kemampuan (capability), bukan cakupan data.
 *
 * - **Admin** — kontrol penuh, nasional.
 * - **ST**    — Secretary/Treasurer. Sama seperti admin untuk invoice dan
 *   pembayaran, tapi HANYA di chapternya sendiri. Tidak mengubah pengaturan
 *   dan tidak menjalankan sinkronisasi — keduanya nasional.
 * - **MC**    — Membership Committee. Melihat dan mengekspor; menjawab
 *   permintaan konfirmasi renewal (menyusul di alur B).
 * - **User**  — hanya melihat & mengekspor, nasional.
 *
 * Berkas ini membatasi APA yang bisa dilakukan, bukan BARIS MANA yang terlihat.
 *
 * Lingkup per-chapter ditegakkan di BACKEND, di dalam query — lihat
 * backend/internal/scope. TIDAK ada row-level security yang menegakkannya:
 * backend menyambung ke Postgres sebagai satu peran tepercaya, jadi basis data
 * melihat satu identitas saja. Berkas ini menyembunyikan tombol; yang benar-benar
 * menahan data adalah query di sisi server.
 */

export const ROLE_LABEL: Record<UserRole, string> = {
  admin: 'Administrator',
  st: 'Secretary/Treasurer',
  mc: 'Membership Committee',
  user: 'User',
}

export type Permission =
  /** Buat invoice baru (termasuk bulk-generate renewal). */
  | 'invoice:create'
  /** Terbitkan / kirim ulang / tandai lunas / batalkan invoice. */
  | 'invoice:manage'
  /** Catat pembayaran manual + unggah bukti. */
  | 'payment:record'
  /** Ubah pengaturan biaya & metode pembayaran. */
  | 'settings:manage'
  /** Jalankan sinkronisasi data dari BNI VM. */
  | 'sync:run'
  /** Ekspor Excel / CSV / PDF. */
  | 'export:data'

const ROLE_PERMISSIONS: Record<UserRole, readonly Permission[]> = {
  admin: [
    'invoice:create',
    'invoice:manage',
    'payment:record',
    'settings:manage',
    'sync:run',
    'export:data',
  ],
  // ST punya kemampuan yang sama dengan admin atas invoice dan pembayaran —
  // yang membedakan bukan daftar ini, melainkan LINGKUPNYA di backend. Yang
  // sengaja TIDAK diberikan: settings:manage dan sync:run, karena keduanya
  // mengubah hal yang berlaku nasional, bukan hanya untuk satu chapter.
  st: ['invoice:create', 'invoice:manage', 'payment:record', 'export:data'],
  mc: ['export:data'],
  user: ['export:data'],
}

export function can(role: UserRole | null | undefined, permission: Permission): boolean {
  if (!role) return false
  return ROLE_PERMISSIONS[role]?.includes(permission) ?? false
}

export function isAdmin(role: UserRole | null | undefined): boolean {
  return role === 'admin'
}
