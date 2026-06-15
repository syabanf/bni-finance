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
}
