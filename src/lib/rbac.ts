import type { UserRole } from '@/types'

/**
 * RBAC berbasis kemampuan (capability), bukan cakupan data.
 *
 * - **Admin** — kontrol penuh, nasional.
 * - **ST**: Secretary/Treasurer, di chapternya sendiri. Mengikuti alur
 *   renewal klien: meminta invoice ke finance dan mengirim bukti bayar. Finance
 *   (admin) yang menerbitkan invoice dan menandai lunas.
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
  /** ST meminta finance menerbitkan invoice renewal. */
  | 'invoice:request'
  /** ST mengirim bukti bayar ke finance untuk diverifikasi. */
  | 'payment:submit'

const ROLE_PERMISSIONS: Record<UserRole, readonly Permission[]> = {
  admin: [
    'invoice:create',
    'invoice:manage',
    'payment:record',
    'settings:manage',
    'sync:run',
    'export:data',
  ],
  // Alur renewal klien (24 November 2026): ST meminta invoice ke finance
  // nasional dan mengirim bukti bayar kepadanya. Menerbitkan invoice dan
  // menandai lunas adalah pekerjaan finance, dan server menolaknya dari ST.
  st: ['invoice:request', 'payment:submit', 'export:data'],
  mc: ['export:data'],
  user: ['export:data'],
}

export function can(role: UserRole | null | undefined, permission: Permission): boolean {
  if (!role) return false
  return ROLE_PERMISSIONS[role]?.includes(permission) ?? false
}

