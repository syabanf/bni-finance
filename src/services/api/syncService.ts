import { api } from '@/lib/apiClient'

/**
 * Sinkronisasi BNI Visitor Management.
 *
 * Dulu ini berjalan di browser: halaman mengambil token BNI VM dari
 * app_settings, memanggil API lewat proxy Vite untuk menghindari CORS, lalu
 * menulis barisnya sendiri. Sekarang seluruhnya di server — token tidak pernah
 * sampai ke browser, dan tidak ada proxy yang perlu dijaga.
 */
export interface SyncResult {
  chapters: number
  members: number
  /** Member yang hilang dari BNI VM — dinonaktifkan, bukan dihapus. */
  deactivated: number
  syncedAt: string
}

/** Satu panggilan menyegarkan chapter DAN member; keduanya berasal dari sumber yang sama. */
export function runSync(): Promise<SyncResult> {
  return api.post<SyncResult>('/sync')
}
