import { api } from '@/lib/apiClient'
import type { ImportRepository } from '@/services/types'
import type { ImportHasil } from '@/types'

/**
 * Impor lewat backend Go.
 *
 * `preview` dan `apply` memanggil endpoint YANG SAMA, hanya berbeda parameter
 * `terapkan`. Itu disengaja di sisi server: pratinjau yang dihitung kode
 * berbeda dari yang menulis adalah pratinjau yang bisa berbohong.
 */
export const apiImportRepository: ImportRepository = {
  async preview(jenis, file) {
    return api.uploadFor<ImportHasil>(`/import/${jenis}`, file)
  },

  async apply(jenis, file) {
    return api.uploadFor<ImportHasil>(`/import/${jenis}?terapkan=true`, file)
  },
}
