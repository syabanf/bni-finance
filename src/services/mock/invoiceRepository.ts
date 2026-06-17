import type { CreateInvoiceInput, InvoiceRepository } from '@/services/types'
import type {
  AuditLogEntry,
  Invoice,
  InvoiceWithRelations,
  RenewalDueMember,
} from '@/types'
import { daysUntil } from '@/lib/date'
import { delay, nextId, nowISO, store } from './store'

/** Mark sent invoices past their due date as overdue. Called lazily before reads. */
function syncOverdueStatus() {
  const today = new Date().toISOString().slice(0, 10)
  for (const inv of store.invoices) {
    if (inv.status === 'sent' && inv.dueDate < today) {
      inv.status = 'overdue'
      inv.updatedAt = nowISO()
      pushAudit({
        invoiceId: inv.id,
        action: 'overdue',
        oldStatus: 'sent',
        newStatus: 'overdue',
        actorId: 'system',
        actorName: 'Sistem',
        notes: 'Jatuh tempo terlewati — status otomatis diubah ke overdue',
      })
    }
  }
}

function relations(invoice: Invoice): InvoiceWithRelations {
  return {
    ...invoice,
    member: store.members.find((m) => m.id === invoice.memberId) ?? null,
    chapter: store.chapters.find((c) => c.id === invoice.chapterId) ?? null,
  }
}

function pushAudit(entry: Omit<AuditLogEntry, 'id' | 'createdAt'>) {
  store.auditLog.push({
    ...entry,
    id: nextId('aud'),
    createdAt: nowISO(),
  })
}

function nextInvoiceNumber(): string {
  const year = new Date().getFullYear()
  const seq = store.invoices.length + 1
  return `INV-${year}-${String(seq).padStart(3, '0')}`
}

export const mockInvoiceRepository: InvoiceRepository = {
  async list(filters) {
    syncOverdueStatus()
    let result = store.invoices.map(relations)

    if (filters?.status && filters.status !== 'all') {
      result = result.filter((i) => i.status === filters.status)
    }
    if (filters?.type && filters.type !== 'all') {
      result = result.filter((i) => i.type === filters.type)
    }
    if (filters?.chapterId && filters.chapterId !== 'all') {
      result = result.filter((i) => i.chapterId === filters.chapterId)
    }
    if (filters?.search) {
      const q = filters.search.toLowerCase()
      result = result.filter(
        (i) =>
          i.number.toLowerCase().includes(q) ||
          (i.member?.name ?? '').toLowerCase().includes(q),
      )
    }

    return delay(
      result.sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
    )
  },

  async getById(id) {
    const invoice = store.invoices.find((i) => i.id === id)
    return delay(invoice ? relations(invoice) : null)
  },

  async listByMember(memberId) {
    return delay(
      store.invoices
        .filter((i) => i.memberId === memberId)
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
    )
  },

  async create(input: CreateInvoiceInput) {
    const member = store.members.find((m) => m.id === input.memberId)
    if (!member) throw new Error('Member tidak ditemukan.')

    const invoice: Invoice = {
      id: nextId('inv'),
      number: nextInvoiceNumber(),
      memberId: input.memberId,
      chapterId: member.chapterId,
      type: input.type,
      amount: input.amount,
      currency: 'IDR',
      dueDate: input.dueDate,
      periodStart: input.periodStart,
      periodEnd: input.periodEnd,
      status: 'draft',
      notes: input.notes,
      createdBy: 'admin-national',
      createdAt: nowISO(),
      updatedAt: nowISO(),
    }
    store.invoices.unshift(invoice)
    pushAudit({
      invoiceId: invoice.id,
      action: 'created',
      newStatus: 'draft',
      actorId: 'admin-national',
      actorName: 'Admin Nasional',
      notes: `Invoice ${input.type} dibuat`,
    })
    return delay(invoice, 400)
  },

  async send(id) {
    const invoice = store.invoices.find((i) => i.id === id)
    if (!invoice) throw new Error('Invoice tidak ditemukan.')
    if (invoice.status !== 'draft') throw new Error('Hanya invoice draft yang bisa dikirim.')

    const old = invoice.status
    invoice.status = 'sent'
    // Simulate Paper.id response
    const ref = nextId('PPR').toUpperCase()
    invoice.paperIdInvoiceId = ref
    invoice.paperIdInvoiceUrl = `https://app.paper.id/invoice/${ref}`
    invoice.paperIdPaymentUrl = `https://pay.paper.id/${ref}`
    invoice.paperIdSentAt = nowISO()
    invoice.updatedAt = nowISO()

    pushAudit({
      invoiceId: id,
      action: 'sent',
      oldStatus: old,
      newStatus: 'sent',
      actorId: 'admin-national',
      actorName: 'Admin Nasional',
      notes: 'Dikirim ke Paper.id — link pembayaran dibuat',
    })
    return delay(invoice, 800)
  },

  async resend(id) {
    const invoice = store.invoices.find((i) => i.id === id)
    if (!invoice) throw new Error('Invoice tidak ditemukan.')
    if (invoice.status !== 'sent' && invoice.status !== 'overdue') {
      throw new Error('Hanya invoice outstanding/overdue yang bisa di-resend.')
    }
    // Refresh Paper.id payment link
    const ref = invoice.paperIdInvoiceId ?? nextId('PPR').toUpperCase()
    invoice.paperIdInvoiceId = ref
    invoice.paperIdInvoiceUrl = `https://app.paper.id/invoice/${ref}`
    invoice.paperIdPaymentUrl = `https://pay.paper.id/${ref}`
    invoice.paperIdSentAt = nowISO()
    invoice.updatedAt = nowISO()

    pushAudit({
      invoiceId: id,
      action: 'sent',
      oldStatus: invoice.status,
      newStatus: invoice.status,
      actorId: 'admin-national',
      actorName: 'Admin Nasional',
      notes: 'Dikirim ulang ke Paper.id — link pembayaran diperbarui',
    })
    return delay(invoice, 600)
  },

  async cancel(id, reason) {
    const invoice = store.invoices.find((i) => i.id === id)
    if (!invoice) throw new Error('Invoice tidak ditemukan.')
    if (invoice.status === 'paid') throw new Error('Invoice yang sudah dibayar tidak bisa dibatalkan.')

    const old = invoice.status
    invoice.status = 'cancelled'
    invoice.cancelledBy = 'admin-national'
    invoice.cancelledAt = nowISO()
    invoice.cancelReason = reason
    invoice.updatedAt = nowISO()

    pushAudit({
      invoiceId: id,
      action: 'cancelled',
      oldStatus: old,
      newStatus: 'cancelled',
      actorId: 'admin-national',
      actorName: 'Admin Nasional',
      notes: reason,
    })
    return delay(invoice, 400)
  },

  async markPaid(id) {
    // Simulates the Paper.id "payment.success" webhook arriving.
    const invoice = store.invoices.find((i) => i.id === id)
    if (!invoice) throw new Error('Invoice tidak ditemukan.')
    if (invoice.status !== 'sent' && invoice.status !== 'overdue') {
      throw new Error('Hanya invoice terkirim/overdue yang bisa ditandai lunas.')
    }

    const old = invoice.status
    const paidAt = nowISO()
    invoice.status = 'paid'
    invoice.paidAt = paidAt
    invoice.paidAmount = invoice.amount
    invoice.updatedAt = paidAt

    store.payments.unshift({
      id: nextId('pay'),
      invoiceId: id,
      amount: invoice.amount,
      paidAt,
      paymentMethod: 'virtual_account',
      paperIdPaymentId: nextId('PAY').toUpperCase(),
      paperIdStatus: 'success',
      createdAt: paidAt,
    })

    pushAudit({
      invoiceId: id,
      action: 'paid',
      oldStatus: old,
      newStatus: 'paid',
      actorId: 'system',
      actorName: 'Webhook Paper.id',
      notes: 'Pembayaran diterima via Paper.id',
    })
    return delay(invoice, 600)
  },

  async recordManualPayment(id, input) {
    const invoice = store.invoices.find((i) => i.id === id)
    if (!invoice) throw new Error('Invoice tidak ditemukan.')
    if (invoice.status !== 'sent' && invoice.status !== 'overdue') {
      throw new Error('Hanya invoice terkirim/overdue yang bisa dicatat pembayarannya.')
    }

    const old = invoice.status
    const paidAt = new Date(input.paidAt).toISOString()
    invoice.status = 'paid'
    invoice.paidAt = paidAt
    invoice.paidAmount = input.amount
    invoice.updatedAt = nowISO()

    store.payments.unshift({
      id: nextId('pay'),
      invoiceId: id,
      amount: input.amount,
      paidAt,
      paymentMethod: input.method,
      proofUrl: input.proofUrl,
      note: input.note,
      paperIdStatus: 'manual',
      createdAt: nowISO(),
    })

    pushAudit({
      invoiceId: id,
      action: 'paid',
      oldStatus: old,
      newStatus: 'paid',
      actorId: 'admin-national',
      actorName: 'Admin Nasional',
      notes: `Pembayaran manual dicatat (${input.method})${input.note ? ' — ' + input.note : ''}`,
    })
    return delay(invoice, 500)
  },

  async getAuditLog(invoiceId) {
    return delay(
      store.auditLog
        .filter((a) => a.invoiceId === invoiceId)
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
    )
  },

  async renewalDue(withinDays = 30) {
    // Latest invoice per member that defines the current membership period.
    const latestByMember = new Map<string, Invoice>()
    for (const inv of store.invoices) {
      if (inv.status === 'cancelled') continue
      const existing = latestByMember.get(inv.memberId)
      if (!existing || inv.periodEnd > existing.periodEnd) {
        latestByMember.set(inv.memberId, inv)
      }
    }

    const result: RenewalDueMember[] = []
    for (const [memberId, lastInvoice] of latestByMember) {
      const days = daysUntil(lastInvoice.periodEnd)
      if (days > withinDays) continue // not due yet

      // Skip if a newer invoice already covers the next period.
      const hasFuture = store.invoices.some(
        (i) =>
          i.memberId === memberId &&
          i.status !== 'cancelled' &&
          i.periodStart > lastInvoice.periodStart,
      )
      if (hasFuture) continue

      const member = store.members.find((m) => m.id === memberId)
      if (!member) continue

      result.push({
        ...member,
        chapter: store.chapters.find((c) => c.id === member.chapterId) ?? null,
        lastInvoice,
        daysUntilDue: days,
      })
    }

    return delay(result.sort((a, b) => a.daysUntilDue - b.daysUntilDue))
  },
}
