import { api } from '@/lib/apiClient'
import type { PeriodRepository } from '@/services/types'
import type { RenewalPeriod } from '@/types'

export const apiPeriodRepository: PeriodRepository = {
  async list() {
    const res = await api.get<{ data: RenewalPeriod[] }>('/renewal-periods')
    return res.data
  },

  async create(input) {
    return api.post<RenewalPeriod>('/renewal-periods', input)
  },

  async update(id, input) {
    return api.patch<RenewalPeriod>(`/renewal-periods/${encodeURIComponent(id)}`, input)
  },

  async remove(id) {
    await api.delete(`/renewal-periods/${encodeURIComponent(id)}`)
  },
}
