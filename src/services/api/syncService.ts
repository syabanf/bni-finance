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
  /** Baris tersimpan yang pada sinkronisasi ini ditautkan ke BNI VM lewat nama. */
  linkedChapters: number
  linkedMembers: number
  /** Baris BNI VM yang namanya ambigu. Mendapat baris baru, jadi bisa saja duplikat. */
  needsReview: string[]
  syncedAt: string
}

/** Satu panggilan menyegarkan chapter DAN member; keduanya berasal dari sumber yang sama. */
export function runSync(): Promise<SyncResult> {
  return api.post<SyncResult>('/sync')
}
