import type { PaymentRepository } from '@/services/types'
import type { PaymentFilters, PaymentPage, PaymentSummary, PaymentWithInvoice } from '@/types'
import { delay, store } from './store'

function withRelations(p: (typeof store.payments)[number]): PaymentWithInvoice {
  const invoice = store.invoices.find((i) => i.id === p.invoiceId) ?? null
  const member = invoice ? store.members.find((m) => m.id === invoice.memberId) ?? null : null
  return { ...p, invoice, member }
}

/**
 * Saringan yang SAMA dengan klausaPayment di server. `denganMetode=false`
 * meniru daftar metode di ringkasan, yang tidak ikut menyempit saat metodenya
 * disaring.
 */
function saring(f: PaymentFilters | undefined, denganMetode: boolean): PaymentWithInvoice[] {
  const q = f?.search?.trim().toLowerCase()
  return store.payments
    .map(withRelations)
    .filter((p) => {
      if (denganMetode && f?.method && f.method !== 'all' && (p.paymentMethod ?? '') !== f.method) return false
      if (f?.chapterId && f.chapterId !== 'all' && p.invoice?.chapterId !== f.chapterId) return false
      if (f?.type && f.type !== 'all' && p.invoice?.type !== f.type) return false
      const hari = p.paidAt ? p.paidAt.slice(0, 10) : ''
      if (f?.paidFrom && (!hari || hari < f.paidFrom)) return false
      if (f?.paidTo && (!hari || hari > f.paidTo)) return false
      if (
        q &&
        !(p.member?.name ?? '').toLowerCase().includes(q) &&
        !(p.invoice?.number ?? '').toLowerCase().includes(q)
      )
        return false
      return true
    })
    .sort((a, b) => b.paidAt.localeCompare(a.paidAt))
}

export const mockPaymentRepository: PaymentRepository = {
  async list(filters) {
    return delay(saring(filters, true))
  },

  async listPaged(filters) {
    const semua = saring(filters, true)
    const offset = filters?.offset ?? 0
    const halaman: PaymentPage = {
      rows: semua.slice(offset, offset + (filters?.limit ?? 25)),
      total: semua.length,
    }
    if (filters?.summary) {
      const byType: PaymentSummary['byType'] = {}
      for (const p of semua) {
        if (p.invoice) byType[p.invoice.type] = (byType[p.invoice.type] ?? 0) + p.amount
      }
      const methods = new Set<string>()
      for (const p of saring(filters, false)) if (p.paymentMethod) methods.add(p.paymentMethod)
      halaman.summary = {
        count: semua.length,
        amount: semua.reduce((a, p) => a + p.amount, 0),
        byType,
        methods: [...methods].sort(),
      }
    }
    return delay(halaman)
  },

  async getById(id) {
    const p = store.payments.find((x) => x.id === id)
    return delay(p ? withRelations(p) : null)
  },

  async listByInvoice(invoiceId) {
    return delay(
      store.payments
        .filter((p) => p.invoiceId === invoiceId)
        .map(withRelations)
        .sort((a, b) => b.paidAt.localeCompare(a.paidAt)),
    )
  },

  async update(id, input) {
    const payment = store.payments.find((p) => p.id === id)
    if (!payment) throw new Error('Pembayaran tidak ditemukan.')
    Object.assign(payment, input)
    return delay(payment, 250)
  },

  async uploadProof(file) {
    // Mock: no real storage — return an in-session object URL so the uploaded
    // proof can still be previewed during the demo.
    return delay(URL.createObjectURL(file), 300)
  },
}
