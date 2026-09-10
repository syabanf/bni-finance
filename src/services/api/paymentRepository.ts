import { ApiError, api, fileUrl, query, type ListResponse } from '@/lib/apiClient'
import type { PaymentRepository } from '@/services/types'
import type { Invoice, MemberWithChapter, Payment, PaymentWithInvoice } from '@/types'

/** Attaches the invoice and its member, which the payments table doesn't carry. */
async function withRelations(payments: Payment[]): Promise<PaymentWithInvoice[]> {
  if (payments.length === 0) return []

  const [invoiceRes, memberRes] = await Promise.all([
    api.get<ListResponse<Invoice>>(`/invoices${query({ limit: 200 })}`),
    api.get<ListResponse<MemberWithChapter>>(`/members${query({ limit: 200 })}`),
  ])
  const invoices = new Map(invoiceRes.data.map((i) => [i.id, i]))
  const members = new Map(memberRes.data.map((m) => [m.id, m]))

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

export const apiPaymentRepository: PaymentRepository = {
  async list() {
    const res = await api.get<ListResponse<Payment>>(`/payments${query({ limit: 200 })}`)
    return withRelations(res.data)
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
    const res = await api.get<ListResponse<Payment>>(
      `/payments${query({ invoiceId, limit: 200 })}`,
    )
    return withRelations(res.data)
  },

  async uploadProof(file) {
    const { url } = await api.upload('/uploads', file)
    return url
  },
}
