import { api, query, type ListResponse } from '@/lib/apiClient'
import type { RenewalRepository } from '@/services/types'
import type { RenewalRequest } from '@/types'

/**
 * Konfirmasi renewal lewat backend Go.
 *
 * Daftarnya sudah DIBATASI CHAPTER di server: ST dan MC hanya menerima
 * permintaan chapternya sendiri, dan permintaan chapter lain tidak muncul sama
 * sekali — bukan muncul lalu ditolak. Halaman ini tidak perlu menyaring apa pun.
 */
export const apiRenewalRepository: RenewalRepository = {
  async list(params) {
    const res = await api.get<ListResponse<RenewalRequest>>(
      `/renewal-requests${query({ answer: params?.answer, period: params?.period, limit: 200 })}`,
    )
    return res.data
  },

  async request(memberIds, period) {
    return api.post<{ dibuat: number; dilewati: number; total: number }>('/renewal-requests', {
      memberIds,
      period,
    })
  },

  async answer(id, answer, note) {
    return api.patch<RenewalRequest>(`/renewal-requests/${encodeURIComponent(id)}`, {
      answer,
      note: note?.trim() || null,
    })
  },
}
