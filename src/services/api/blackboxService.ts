import { api, query, type ListResponse } from '@/lib/apiClient'

/**
 * Rekaman "kotak hitam" lalu lintas integrasi.
 *
 * Hanya tersedia di mode Backend API — perekamnya hidup di server, tempat
 * panggilan ke Paper.id/Xendit/BNI VM benar-benar terjadi.
 */
export type BlackboxIntegration = 'paper_id' | 'xendit' | 'bni_vm'
export type BlackboxDirection = 'outbound' | 'inbound'

export interface BlackboxEntry {
  id: string
  time: string
  integration: BlackboxIntegration
  /** `outbound` = kita → pihak luar; `inbound` = callback masuk. */
  direction: BlackboxDirection
  method: string
  url: string
  request?: unknown
  response?: unknown
  status: number
  success: boolean
  durationMs: number
  error?: string
}

export interface BlackboxFilters {
  integration?: BlackboxIntegration | 'all'
  direction?: BlackboxDirection | 'all'
  /** 'failed' = hanya yang gagal. */
  status?: 'failed' | 'all'
  limit?: number
}

export async function listBlackbox(filters: BlackboxFilters = {}): Promise<BlackboxEntry[]> {
  const res = await api.get<ListResponse<BlackboxEntry>>(
    `/blackbox${query({
      integration: filters.integration && filters.integration !== 'all' ? filters.integration : undefined,
      direction: filters.direction && filters.direction !== 'all' ? filters.direction : undefined,
      status: filters.status === 'failed' ? 'failed' : undefined,
      limit: filters.limit ?? 200,
    })}`,
  )
  return res.data
}

export function clearBlackbox(): Promise<void> {
  return api.delete<void>('/blackbox')
}

export const INTEGRATION_LABEL: Record<BlackboxIntegration, string> = {
  paper_id: 'Paper.id',
  xendit: 'Xendit',
  bni_vm: 'BNI VM',
}
