import type { UserRole } from '@/types'

/**
 * RBAC berbasis kemampuan (capability), bukan cakupan data.
 *
 * - **Admin**  — kontrol penuh: buat/kirim/batalkan invoice, catat pembayaran
 *   manual, ubah pengaturan, jalankan sinkronisasi.
 * - **User**   — hanya melihat & mengekspor. Semua aksi yang mengubah data
 *   disembunyikan dan rutenya dijaga.
 *
 * Catatan: ini membatasi APA yang bisa dilakukan, bukan BARIS MANA yang terlihat.
 * Pembatasan per-chapter (data scoping) adalah langkah terpisah — dan batas
 * sesungguhnya tetap RLS di sisi database.
 */

export const ROLE_LABEL: Record<UserRole, string> = {
  admin: 'Administrator',
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
  user: ['export:data'],
}

export function can(role: UserRole | null | undefined, permission: Permission): boolean {
  if (!role) return false
  return ROLE_PERMISSIONS[role]?.includes(permission) ?? false
}

export function isAdmin(role: UserRole | null | undefined): boolean {
  return role === 'admin'
}
