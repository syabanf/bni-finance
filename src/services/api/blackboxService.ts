import { api, query, type ListResponse } from '@/lib/apiClient'
import { isMockMode } from '@/services/dataSource'
import { clearMockCalls, listMockCalls } from '@/services/mock/blackbox'

/**
 * Rekaman "kotak hitam" lalu lintas integrasi.
 *
 * Di mode Backend API perekamnya hidup di server, tempat panggilan ke
 * Paper.id/Xendit/BNI VM benar-benar terjadi. Di mode Data Contoh panggilan itu
 * tidak pergi ke mana-mana, tetapi bentuk rekamannya dibuat identik oleh
 * `mock/blackbox` — sebelumnya halaman ini mati total pada mode yang justru
 * dipakai untuk demo.
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
  if (isMockMode()) {
    return listMockCalls({
      integration: filters.integration,
      direction: filters.direction,
      status: filters.status,
    }).slice(0, filters.limit ?? 200)
  }
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
  if (isMockMode()) {
    clearMockCalls()
    return Promise.resolve()
  }
  return api.delete<void>('/blackbox')
}

export const INTEGRATION_LABEL: Record<BlackboxIntegration, string> = {
  paper_id: 'Paper.id',
  xendit: 'Xendit',
  bni_vm: 'BNI VM',
}
