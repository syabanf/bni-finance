import { api } from '@/lib/apiClient'
import type { DashboardRepository } from '@/services/types'
import type { DashboardSummary } from '@/types'

// The backend computes the whole summary in SQL and returns exactly the shape
// DashboardSummary describes, so this is a straight pass-through.

export const apiDashboardRepository: DashboardRepository = {
  summary() {
    return api.get<DashboardSummary>('/dashboard/summary?months=6')
  },
}
