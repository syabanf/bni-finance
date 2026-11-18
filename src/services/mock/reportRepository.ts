import type { ReportRepository } from '@/services/types'
import type { ReportFilters } from '@/types'
import { hitungLaporan } from '@/features/reports/lib/hitungLaporan'
import { mockChapterRepository } from './chapterRepository'
import { mockInvoiceRepository } from './invoiceRepository'
import { mockPaymentRepository } from './paymentRepository'

/** Mode Data Contoh menghitung laporan di browser, dari store yang sama. */
async function hitung(f: ReportFilters | undefined) {
  const [invoices, payments, chapters] = await Promise.all([
    mockInvoiceRepository.list(),
    mockPaymentRepository.list(),
    mockChapterRepository.list(),
  ])
  return hitungLaporan(invoices, payments, chapters, f)
}

export const mockReportRepository: ReportRepository = {
  async summary(filters) {
    return (await hitung(filters)).summary
  },

  async detailPaged(filters) {
    const { detail } = await hitung(filters)
    const offset = filters?.offset ?? 0
    return { rows: detail.slice(offset, offset + (filters?.limit ?? 25)), total: detail.length }
  },

  async detail(filters) {
    return (await hitung(filters)).detail
  },
}
