import type { PaymentRepository } from '@/services/types'
import type { PaymentWithInvoice } from '@/types'
import { delay, store } from './store'

export const mockPaymentRepository: PaymentRepository = {
  async list() {
    const result: PaymentWithInvoice[] = store.payments.map((p) => {
      const invoice = store.invoices.find((i) => i.id === p.invoiceId) ?? null
      const member = invoice
        ? store.members.find((m) => m.id === invoice.memberId) ?? null
        : null
      return { ...p, invoice, member }
    })
    return delay(result.sort((a, b) => b.paidAt.localeCompare(a.paidAt)))
  },
}
