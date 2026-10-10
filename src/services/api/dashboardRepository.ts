import { api, query } from '@/lib/apiClient'
import type { DashboardRepository } from '@/services/types'
import type { DashboardSummary } from '@/types'

// The backend computes the whole summary in SQL and returns exactly the shape
// DashboardSummary describes, so this is a straight pass-through.

export const apiDashboardRepository: DashboardRepository = {
  summary(params) {
    return api.get<DashboardSummary>(`/dashboard/summary${query({ months: 6, type: params?.type })}`)
  },
}
