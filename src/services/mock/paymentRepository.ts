import type { PaymentRepository } from '@/services/types'
import type { PaymentWithInvoice } from '@/types'
import { delay, store } from './store'

function withRelations(p: (typeof store.payments)[number]): PaymentWithInvoice {
  const invoice = store.invoices.find((i) => i.id === p.invoiceId) ?? null
  const member = invoice ? store.members.find((m) => m.id === invoice.memberId) ?? null : null
  return { ...p, invoice, member }
}

export const mockPaymentRepository: PaymentRepository = {
  async list() {
    return delay(
      store.payments.map(withRelations).sort((a, b) => b.paidAt.localeCompare(a.paidAt)),
    )
  },

  async listByInvoice(invoiceId) {
    return delay(
      store.payments
        .filter((p) => p.invoiceId === invoiceId)
        .map(withRelations)
        .sort((a, b) => b.paidAt.localeCompare(a.paidAt)),
    )
  },

  async uploadProof(file) {
    // Mock: no real storage — return an in-session object URL so the uploaded
    // proof can still be previewed during the demo.
    return delay(URL.createObjectURL(file), 300)
  },
}
