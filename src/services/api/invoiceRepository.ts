import { api, query, type ListResponse } from '@/lib/apiClient'
import { todayISO } from '@/lib/date'
import type { CreateInvoiceInput, InvoiceRepository, ManualPaymentInput } from '@/services/types'
import type {
  AuditLogEntry,
  Chapter,
  Invoice,
  InvoiceWithRelations,
  MemberWithChapter,
  RenewalDueMember,
} from '@/types'
import { getAppSetting } from './settingsRepository'
import { isNotFound } from './chapterRepository'

/**
 * The API returns bare invoices; the UI wants member and chapter attached.
 * Rather than a join per invoice, the reference tables are fetched once per
 * call and indexed — they are small (hundreds of rows) and change rarely.
 */
async function loadDirectory(): Promise<{
  members: Map<string, MemberWithChapter>
  chapters: Map<string, Chapter>
}> {
  const [memberRes, chapterRes] = await Promise.all([
    api.get<ListResponse<MemberWithChapter>>(`/members${query({ limit: 200 })}`),
    api.get<ListResponse<Chapter>>(`/chapters${query({ limit: 500 })}`),
  ])
  return {
    members: new Map(memberRes.data.map((m) => [m.id, m])),
    chapters: new Map(chapterRes.data.map((c) => [c.id, c])),
  }
}

function attach(
  invoice: Invoice,
  members: Map<string, MemberWithChapter>,
  chapters: Map<string, Chapter>,
): InvoiceWithRelations {
  return {
    ...invoice,
    member: members.get(invoice.memberId) ?? null,
    chapter: chapters.get(invoice.chapterId) ?? null,
  }
}

/** Marks sent invoices past their due date as overdue, matching the UI's view. */
async function syncOverdue(invoices: Invoice[]): Promise<void> {
  const today = todayISO()
  const stale = invoices.filter((i) => i.status === 'sent' && i.dueDate < today)
  await Promise.all(
    stale.map((i) =>
      api.patch<Invoice>(`/invoices/${i.id}`, { status: 'overdue' }).catch(() => {
        // A failed sweep must not break the list the user asked for; the next
        // load tries again.
      }),
    ),
  )
  for (const invoice of stale) invoice.status = 'overdue'
}

export const apiInvoiceRepository: InvoiceRepository = {
  async list(filters) {
    const res = await api.get<ListResponse<Invoice>>(
      `/invoices${query({
        status: filters?.status && filters.status !== 'all' ? filters.status : undefined,
        type: filters?.type && filters.type !== 'all' ? filters.type : undefined,
        chapterId: filters?.chapterId && filters.chapterId !== 'all' ? filters.chapterId : undefined,
        limit: 200,
      })}`,
    )
    await syncOverdue(res.data)

    const { members, chapters } = await loadDirectory()
    let rows = res.data.map((i) => attach(i, members, chapters))

    // Search spans the invoice number AND the member name, which the API can't
    // do in one query — it only indexes the number.
    const term = filters?.search?.trim().toLowerCase()
    if (term) {
      rows = rows.filter(
        (r) =>
          r.number.toLowerCase().includes(term) ||
          (r.member?.name ?? '').toLowerCase().includes(term),
      )
    }
    return rows
  },

  async getById(id) {
    let invoice: Invoice
    try {
      invoice = await api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
    } catch (err) {
      if (isNotFound(err)) return null
      throw err
    }
    const { members, chapters } = await loadDirectory()
    return attach(invoice, members, chapters)
  },

  async listByMember(memberId) {
    const res = await api.get<ListResponse<Invoice>>(
      `/invoices${query({ memberId, limit: 200 })}`,
    )
    return res.data
  },

  async create(input: CreateInvoiceInput) {
    // chapterId isn't in the input — it comes from the member's chapter.
    const member = await api.get<MemberWithChapter>(`/members/${encodeURIComponent(input.memberId)}`)
    return api.post<Invoice>('/invoices', {
      memberId: input.memberId,
      chapterId: member.chapterId,
      type: input.type,
      amount: input.amount,
      dueDate: input.dueDate,
      periodStart: input.periodStart,
      periodEnd: input.periodEnd,
      notes: input.notes,
    })
  },

  async send(id) {
    const invoice = await api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
    if (invoice.status !== 'draft') throw new Error('Hanya invoice draft yang bisa dikirim')

    // Self Payment Mode ON  → the payer settles via Xendit on the public page,
    //                         so there's no Paper.id push — just issue it.
    // Self Payment Mode OFF → the server pushes a real invoice to Paper.id and
    //                         flips the status in one call.
    const selfPayment = (await getAppSetting('self_payment_mode')) === 'true'

    if (selfPayment) {
      const dueDaysAfter = Number((await getAppSetting('invoice_due_days_after')) ?? 30)
      const due = new Date()
      due.setDate(due.getDate() + dueDaysAfter)
      return api.patch<Invoice>(`/invoices/${encodeURIComponent(id)}`, {
        status: 'sent',
        dueDate: due.toISOString().slice(0, 10),
      })
    }

    // The Paper.id credentials live on the server; this endpoint does the push,
    // stores the returned payment link + PDF, and records the audit entry.
    //
    // The body is deliberately empty: which channels deliver the invoice is an
    // operational policy read from app_settings by the server, not a per-click
    // choice. Passing flags from here would mean this path, the bulk actions in
    // the invoice list, and "create + send" each had to remember them — and the
    // one that forgot would quietly stop reaching members while still
    // reporting success. Set the channels in Pengaturan.
    return api.post<Invoice>(`/invoices/${encodeURIComponent(id)}/send`, {})
  },

  /**
   * Pengingat: minta Paper.id mengantar tagihan ini lagi ke member.
   *
   * Dulu fungsi ini tidak melakukan apa pun — ia mengambil invoice, memeriksa
   * statusnya, lalu mengembalikannya utuh dengan alasan "tautannya masih
   * berlaku, tidak ada yang perlu dibuat ulang". Benar soal tautannya, tetapi
   * itu bukan yang diminta: menekan Kirim Ulang tidak pernah membuat satu pesan
   * pun sampai ke member, sambil melaporkan sukses.
   *
   * Sekarang benar-benar mengirim. Backend menerbitkan dokumen baru di Paper.id
   * dengan nomor turunan (-R1, -R2) karena Paper.id membakar nomor secara
   * permanen dan tidak punya endpoint pengingat. Status invoice tidak berubah.
   */
  async resend(id) {
    return api.post<Invoice>(`/invoices/${encodeURIComponent(id)}/remind`, {})
  },

  async cancel(id, reason) {
    return api.patch<Invoice>(`/invoices/${encodeURIComponent(id)}`, {
      status: 'cancelled',
      cancelReason: reason,
    })
  },

  async markPaid(id) {
    const invoice = await api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
    // Recording the payment settles the invoice in the same transaction, so
    // there is no separate status update to make.
    await api.post('/payments', {
      invoiceId: id,
      amount: invoice.amount,
      paymentMethod: 'paper_id',
    })
    return api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
  },

  async recordManualPayment(id, input: ManualPaymentInput) {
    await api.post('/payments', {
      invoiceId: id,
      amount: input.amount,
      paidAt: new Date(input.paidAt).toISOString(),
      paymentMethod: input.method,
      proofUrl: input.proofUrl,
      note: input.note,
    })
    return api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
  },

  async getAuditLog(invoiceId) {
    const res = await api.get<ListResponse<AuditLogEntry>>(
      `/invoices/${encodeURIComponent(invoiceId)}/audit${query({ limit: 200 })}`,
    )
    return res.data
  },

  async renewalDue(withinDays = 30) {
    const [due, invoices] = await Promise.all([
      api.get<ListResponse<MemberWithChapter & { daysUntilDue: number }>>(
        `/members/renewal-due${query({ days: withinDays, limit: 200 })}`,
      ),
      api.get<ListResponse<Invoice>>(`/invoices${query({ limit: 200 })}`),
    ])

    // Newest invoice per member — the one whose period is ending.
    const latest = new Map<string, Invoice>()
    for (const invoice of invoices.data) {
      const existing = latest.get(invoice.memberId)
      if (!existing || invoice.periodEnd > existing.periodEnd) latest.set(invoice.memberId, invoice)
    }

    return due.data
      .map((member): RenewalDueMember | null => {
        const lastInvoice = latest.get(member.id)
        // The type requires an invoice; a member who has never been billed has
        // nothing to renew yet.
        if (!lastInvoice) return null
        return { ...member, lastInvoice, daysUntilDue: member.daysUntilDue }
      })
      .filter((m): m is RenewalDueMember => m !== null)
  },
}
