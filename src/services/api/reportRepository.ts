import { api, query, type ListResponse } from '@/lib/apiClient'
import type { ReportRepository } from '@/services/types'
import type { ReportDetailRow, ReportFilters, ReportSummary } from '@/types'
import { MAKS_PER_HALAMAN, tarikSemua } from './kumpulan'

/** Menyusun query dari filter UI; "all" berarti tidak menyaring. */
function paramFilter(f: ReportFilters | undefined) {
  return {
    from: f?.from || undefined,
    to: f?.to || undefined,
    type: f?.type && f.type !== 'all' ? f.type : undefined,
  }
}

/**
 * Laporan dihitung server dari view v_laporan_invoice dan v_laporan_pembayaran.
 * Sebelumnya browser menarik seluruh invoice dan pembayaran untuk menjumlahkan
 * sendiri, satu permintaan per 200 baris.
 */
export const apiReportRepository: ReportRepository = {
  async summary(filters) {
    return api.get<ReportSummary>(`/reports/summary${query(paramFilter(filters))}`)
  },

  async detailPaged(filters) {
    const res = await api.get<ListResponse<ReportDetailRow>>(
      `/reports/detail${query({
        ...paramFilter(filters),
        limit: Math.min(filters?.limit ?? 25, MAKS_PER_HALAMAN),
        offset: filters?.offset ?? 0,
      })}`,
    )
    return { rows: res.data, total: res.meta.total }
  },

  async detail(filters) {
    return tarikSemua<ReportDetailRow>('/reports/detail', paramFilter(filters))
  },
}
