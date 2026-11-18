import { ApiError, api, fileUrl, query, type ListResponse } from '@/lib/apiClient'
import type { PaymentRepository } from '@/services/types'
import type {
  Invoice,
  MemberWithChapter,
  Payment,
  PaymentFilters,
  PaymentSummary,
  PaymentWithInvoice,
} from '@/types'
import { MAKS_PER_HALAMAN, ambilPerId, tarikSemua } from './kumpulan'

/**
 * Menempelkan invoice dan membernya, yang tidak dibawa tabel payments.
 *
 * Hanya invoice dan member yang disebut baris-baris ini yang diminta.
 * Sebelumnya 200 invoice dan 200 member pertama ditarik pada setiap panggilan,
 * termasuk saat membuka satu pembayaran, dan pembayaran untuk invoice ke-201
 * tampil tanpa nama member.
 */
async function withRelations(payments: Payment[]): Promise<PaymentWithInvoice[]> {
  if (payments.length === 0) return []

  const invoices = await ambilPerId<Invoice>(
    '/invoices',
    payments.map((p) => p.invoiceId),
  )
  const members = await ambilPerId<MemberWithChapter>(
    '/members',
    [...invoices.values()].map((i) => i.memberId),
  )

  return payments.map((p) => {
    const invoice = invoices.get(p.invoiceId) ?? null
    return {
      ...p,
      // Proof URLs are stored as paths; make them loadable by the browser.
      proofUrl: fileUrl(p.proofUrl),
      invoice,
      member: invoice ? (members.get(invoice.memberId) ?? null) : null,
    }
  })
}

/** Menyusun query dari filter UI; "all" berarti tidak menyaring. */
function paramFilter(f: PaymentFilters | undefined) {
  const nilai = (v?: string) => (v && v !== 'all' ? v : undefined)
  return {
    q: f?.search?.trim() || undefined,
    method: nilai(f?.method),
    chapterId: nilai(f?.chapterId),
    type: nilai(f?.type),
    paidFrom: f?.paidFrom || undefined,
    paidTo: f?.paidTo || undefined,
  }
}

export const apiPaymentRepository: PaymentRepository = {
  async list(filters) {
    return withRelations(await tarikSemua<Payment>('/payments', paramFilter(filters)))
  },

  async listPaged(filters) {
    const res = await api.get<ListResponse<Payment> & { meta: { summary?: PaymentSummary } }>(
      `/payments${query({
        ...paramFilter(filters),
        summary: filters?.summary ? 'true' : undefined,
        limit: Math.min(filters?.limit ?? 25, MAKS_PER_HALAMAN),
        offset: filters?.offset ?? 0,
      })}`,
    )
    return {
      rows: await withRelations(res.data),
      total: res.meta.total,
      summary: res.meta.summary,
    }
  },

  async getById(id) {
    let payment: Payment
    try {
      payment = await api.get<Payment>(`/payments/${encodeURIComponent(id)}`)
    } catch (err) {
      // Kontraknya mengembalikan null untuk "tidak ditemukan", bukan melempar —
      // halaman detail perlu membedakan "tidak ada" dari "gagal memuat".
      if (err instanceof ApiError && err.status === 404) return null
      throw err
    }
    const [lengkap] = await withRelations([payment])
    return lengkap ?? null
  },

  async listByInvoice(invoiceId) {
    return withRelations(await tarikSemua<Payment>('/payments', { invoiceId }))
  },

  async update(id, input) {
    return api.patch<Payment>(`/payments/${encodeURIComponent(id)}`, input)
  },

  async uploadProof(file) {
    const { url } = await api.upload('/uploads', file)
    return url
  },
}
