import { api } from '@/lib/apiClient'
import type { ImportOpsi, ImportRepository } from '@/services/types'
import type { ImportHasil } from '@/types'

/**
 * Impor lewat backend Go.
 *
 * `preview` dan `apply` memanggil endpoint YANG SAMA, hanya berbeda parameter
 * `terapkan`. Itu disengaja di sisi server: pratinjau yang dihitung kode
 * berbeda dari yang menulis adalah pratinjau yang bisa berbohong.
 */
function kueri(opsi: ImportOpsi | undefined, terapkan: boolean): string {
  const p = new URLSearchParams()
  if (terapkan) p.set('terapkan', 'true')
  if (opsi?.chapter) p.set('chapter', opsi.chapter)
  if (opsi?.statusBawaan) p.set('status_bawaan', opsi.statusBawaan)
  const q = p.toString()
  return q ? `?${q}` : ''
}

export const apiImportRepository: ImportRepository = {
  async preview(jenis, file, opsi) {
    return api.uploadFor<ImportHasil>(`/import/${jenis}${kueri(opsi, false)}`, file)
  },

  async apply(jenis, file, opsi) {
    return api.uploadFor<ImportHasil>(`/import/${jenis}${kueri(opsi, true)}`, file)
  },
}
