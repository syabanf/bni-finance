import { supabase } from '@/lib/supabase'
import { getAppSetting } from './settingsRepository'
import type { CreateInvoiceInput, InvoiceRepository } from '@/services/types'
import type {
  AuditAction,
  AuditLogEntry,
  Chapter,
  Invoice,
  InvoiceFilters,
  InvoiceStatus,
  InvoiceType,
  InvoiceWithRelations,
  Member,
  MemberStatus,
  MemberWithChapter,
  RenewalDueMember,
} from '@/types'

// ---------------------------------------------------------------------------
// Row mappers
// ---------------------------------------------------------------------------

function rowToInvoice(r: Record<string, unknown>): Invoice {
  return {
    id: r.id as string,
    number: r.number as string,
    memberId: r.member_id as string,
    chapterId: r.chapter_id as string,
    type: r.type as InvoiceType,
    amount: r.amount as number,
    currency: r.currency as string,
    dueDate: r.due_date as string,
    periodStart: r.period_start as string,
    periodEnd: r.period_end as string,
    status: r.status as InvoiceStatus,
    paperIdInvoiceId: r.paper_id_invoice_id as string | undefined,
    paperIdInvoiceUrl: r.paper_id_invoice_url as string | undefined,
    paperIdPaymentUrl: r.paper_id_payment_url as string | undefined,
    paperIdSentAt: r.paper_id_sent_at as string | undefined,
    paymentProvider: r.payment_provider as string | undefined,
    xenditPaymentId: r.xendit_payment_id as string | undefined,
    xenditPaymentMethod: r.xendit_payment_method as 'va' | 'qris' | undefined,
    xenditVaBank: r.xendit_va_bank as string | undefined,
    xenditVaNumber: r.xendit_va_number as string | undefined,
    xenditQrisString: r.xendit_qris_string as string | undefined,
    xenditPaymentStatus: r.xendit_payment_status as string | undefined,
    xenditExpiresAt: r.xendit_expires_at as string | undefined,
    paidAt: r.paid_at as string | undefined,
    paidAmount: r.paid_amount as number | undefined,
    notes: r.notes as string | undefined,
    createdBy: r.created_by as string | undefined,
    cancelledBy: r.cancelled_by as string | undefined,
    cancelledAt: r.cancelled_at as string | undefined,
    cancelReason: r.cancel_reason as string | undefined,
    createdAt: r.created_at as string,
    updatedAt: r.updated_at as string,
  }
}

function rowToMember(r: Record<string, unknown>): Member {
  return {
    id: r.id as string,
    chapterId: r.chapter_id as string,
    name: r.name as string,
    email: r.email as string | undefined,
    phone: r.phone as string | undefined,
    status: r.status as MemberStatus,
    joinedDate: r.joined_date as string | null,
    renewalDate: r.renewal_date as string | null,
    syncedAt: r.synced_at as string,
  }
}

function rowToChapter(r: Record<string, unknown>): Chapter {
  return {
    id: r.id as string,
    name: r.name as string,
    displayName: r.display_name as string,
    areaName: r.area_name as string | undefined,
    cityName: r.city_name as string | undefined,
    syncedAt: r.synced_at as string,
  }
}

function rowToAudit(r: Record<string, unknown>): AuditLogEntry {
  return {
    id: r.id as string,
    invoiceId: r.invoice_id as string,
    action: r.action as AuditAction,
    oldStatus: r.old_status as InvoiceStatus | undefined,
    newStatus: r.new_status as InvoiceStatus | undefined,
    actorId: r.actor_id as string | undefined,
    actorName: r.actor_name as string | undefined,
    notes: r.notes as string | undefined,
    createdAt: r.created_at as string,
  }
}

function withRelations(r: Record<string, unknown>): InvoiceWithRelations {
  const mem = r.members as Record<string, unknown> | null
  const ch = r.chapters as Record<string, unknown> | null
  return {
    ...rowToInvoice(r),
    member: mem ? rowToMember(mem) : null,
    chapter: ch ? rowToChapter(ch) : null,
  }
}

// ---------------------------------------------------------------------------
// Auto-mark overdue (server-side: update rows where status=sent and due_date<today)
// ---------------------------------------------------------------------------

async function syncOverdue() {
  const today = new Date().toISOString().slice(0, 10)
  await supabase
    .from('invoices')
    .update({ status: 'overdue' })
    .eq('status', 'sent')
    .lt('due_date', today)
}

// ---------------------------------------------------------------------------
// Next invoice number (client-generated, good enough for now)
// ---------------------------------------------------------------------------

async function nextInvoiceNumber(): Promise<string> {
  const year = new Date().getFullYear()
  const { count } = await supabase
    .from('invoices')
    .select('id', { count: 'exact', head: true })
  const seq = (count ?? 0) + 1
  return `INV-${year}-${String(seq).padStart(3, '0')}`
}

async function pushAudit(
  invoiceId: string,
  action: AuditAction,
  oldStatus?: InvoiceStatus,
  newStatus?: InvoiceStatus,
  notes?: string,
) {
  await supabase.from('invoice_audit_log').insert({
    invoice_id: invoiceId,
    action,
    old_status: oldStatus,
    new_status: newStatus,
    actor_name: 'Admin',
    notes,
  })
}

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

export const supabaseInvoiceRepository: InvoiceRepository = {
  async list(filters?: InvoiceFilters) {
    await syncOverdue()
    let q = supabase
      .from('invoices')
      .select('*, members(*), chapters(*)')
      .order('created_at', { ascending: false })

    if (filters?.status && filters.status !== 'all') {
      q = q.eq('status', filters.status)
    }
    if (filters?.type && filters.type !== 'all') {
      q = q.eq('type', filters.type)
    }
    if (filters?.chapterId && filters.chapterId !== 'all') {
      q = q.eq('chapter_id', filters.chapterId)
    }
    if (filters?.search) {
      // search by invoice number (ilike) — member name search requires RPC or post-filter
      q = q.ilike('number', `%${filters.search}%`)
    }

    const { data, error } = await q
    if (error) throw new Error(error.message)
    return (data ?? []).map(withRelations)
  },

  async getById(id) {
    const { data, error } = await supabase
      .from('invoices')
      .select('*, members(*), chapters(*)')
      .eq('id', id)
      .maybeSingle()
    if (error) throw new Error(error.message)
    return data ? withRelations(data) : null
  },

  async listByMember(memberId) {
    const { data, error } = await supabase
      .from('invoices')
      .select('*')
      .eq('member_id', memberId)
      .order('created_at', { ascending: false })
    if (error) throw new Error(error.message)
    return (data ?? []).map(rowToInvoice)
  },

  async create(input: CreateInvoiceInput) {
    const { data: member, error: memErr } = await supabase
      .from('members')
      .select('chapter_id')
      .eq('id', input.memberId)
      .single()
    if (memErr) throw new Error(memErr.message)

    const number = await nextInvoiceNumber()

    const { data, error } = await supabase
      .from('invoices')
      .insert({
        number,
        member_id: input.memberId,
        chapter_id: (member as Record<string, unknown>).chapter_id,
        type: input.type,
        amount: input.amount,
        due_date: input.dueDate,
        period_start: input.periodStart,
        period_end: input.periodEnd,
        notes: input.notes,
        status: 'draft',
      })
      .select('*')
      .single()
    if (error) throw new Error(error.message)

    const invoice = rowToInvoice(data as Record<string, unknown>)
    await pushAudit(invoice.id, 'created', undefined, 'draft', 'Invoice dibuat')
    return invoice
  },

  async send(id) {
    const { data: existing, error: fetchErr } = await supabase
      .from('invoices')
      .select('*')
      .eq('id', id)
      .single()
    if (fetchErr) throw new Error(fetchErr.message)
    const inv = rowToInvoice(existing as Record<string, unknown>)
    if (inv.status !== 'draft') throw new Error('Hanya invoice draft yang bisa dikirim')

    // due_date = hari pengiriman + N hari (dari app_settings, default 30)
    const dueDaysAfterSetting = await getAppSetting('invoice_due_days_after')
    const dueDaysAfter = Number(dueDaysAfterSetting ?? 30)
    const sentAt = new Date()
    const dueDate = new Date(sentAt)
    dueDate.setDate(dueDate.getDate() + dueDaysAfter)
    const dueDateStr = dueDate.toISOString().slice(0, 10)

    // Self Payment Mode ON  → bayar mandiri via Xendit (tanpa Paper.id)
    // Self Payment Mode OFF → integrasi Paper.id (simulasi link pembayaran)
    const selfPayment = (await getAppSetting('self_payment_mode')) === 'true'

    const updatePayload: Record<string, unknown> = {
      status: 'sent',
      due_date: dueDateStr,
    }
    let auditNote = `Invoice diterbitkan — jatuh tempo ${dueDateStr}`

    if (!selfPayment) {
      const fakeId = `paperid-${id.slice(0, 8)}`
      updatePayload.paper_id_invoice_id = fakeId
      updatePayload.paper_id_invoice_url = `https://paper.id/invoice/${fakeId}`
      updatePayload.paper_id_payment_url = `https://paper.id/pay/${fakeId}`
      updatePayload.paper_id_sent_at = sentAt.toISOString()
      auditNote = 'Dikirim ke Paper.id'
    }

    const { data, error } = await supabase
      .from('invoices')
      .update(updatePayload)
      .eq('id', id)
      .select('*')
      .single()
    if (error) throw new Error(error.message)

    const updated = rowToInvoice(data as Record<string, unknown>)
    await pushAudit(id, 'sent', 'draft', 'sent', auditNote)
    return updated
  },

  async resend(id) {
    const { data: existing, error: fetchErr } = await supabase
      .from('invoices')
      .select('*')
      .eq('id', id)
      .single()
    if (fetchErr) throw new Error(fetchErr.message)
    const inv = rowToInvoice(existing as Record<string, unknown>)
    if (inv.status !== 'sent' && inv.status !== 'overdue') {
      throw new Error('Hanya invoice sent/overdue yang bisa di-resend')
    }

    const fakeId = inv.paperIdInvoiceId ?? `paperid-${id.slice(0, 8)}`
    const { data, error } = await supabase
      .from('invoices')
      .update({
        paper_id_invoice_id: fakeId,
        paper_id_invoice_url: `https://paper.id/invoice/${fakeId}`,
        paper_id_payment_url: `https://paper.id/pay/${fakeId}?resend=${Date.now()}`,
        paper_id_sent_at: new Date().toISOString(),
      })
      .eq('id', id)
      .select('*')
      .single()
    if (error) throw new Error(error.message)

    const updated = rowToInvoice(data as Record<string, unknown>)
    await pushAudit(id, 'sent', inv.status, inv.status, 'Resend ke Paper.id — link diperbarui')
    return updated
  },

  async cancel(id, reason) {
    const now = new Date().toISOString()
    const { data, error } = await supabase
      .from('invoices')
      .update({ status: 'cancelled', cancelled_at: now, cancel_reason: reason })
      .eq('id', id)
      .select('*, members(*), chapters(*)')
      .single()
    if (error) throw new Error(error.message)

    const inv = rowToInvoice(data as Record<string, unknown>)
    await pushAudit(id, 'cancelled', undefined, 'cancelled', reason)
    return inv
  },

  async markPaid(id) {
    const { data: existing, error: fetchErr } = await supabase
      .from('invoices')
      .select('*')
      .eq('id', id)
      .single()
    if (fetchErr) throw new Error(fetchErr.message)
    const inv = rowToInvoice(existing as Record<string, unknown>)

    const now = new Date().toISOString()
    const { data, error } = await supabase
      .from('invoices')
      .update({ status: 'paid', paid_at: now, paid_amount: inv.amount })
      .eq('id', id)
      .select('*')
      .single()
    if (error) throw new Error(error.message)

    // Insert payment record
    await supabase.from('payments').insert({
      invoice_id: id,
      amount: inv.amount,
      paid_at: now,
      payment_method: 'paper_id',
    })

    const updated = rowToInvoice(data as Record<string, unknown>)
    await pushAudit(id, 'paid', inv.status, 'paid', 'Pembayaran dikonfirmasi')
    return updated
  },

  async getAuditLog(invoiceId) {
    const { data, error } = await supabase
      .from('invoice_audit_log')
      .select('*')
      .eq('invoice_id', invoiceId)
      .order('created_at', { ascending: true })
    if (error) throw new Error(error.message)
    return (data ?? []).map(rowToAudit)
  },

  async renewalDue(withinDays = 30) {
    await syncOverdue()
    const today = new Date()
    const cutoff = new Date(today)
    cutoff.setDate(cutoff.getDate() + withinDays)
    const cutoffStr = cutoff.toISOString().slice(0, 10)
    const todayStr = today.toISOString().slice(0, 10)

    // Get the latest paid invoice per member where period_end is within range
    const { data, error } = await supabase
      .from('invoices')
      .select('*, members(*, chapters(*))')
      .eq('status', 'paid')
      .gte('period_end', todayStr)
      .lte('period_end', cutoffStr)
      .order('period_end', { ascending: true })
    if (error) throw new Error(error.message)

    // Deduplicate: keep latest invoice per member
    const seen = new Set<string>()
    const result: RenewalDueMember[] = []
    for (const r of (data ?? [])) {
      const row = r as Record<string, unknown>
      if (seen.has(row.member_id as string)) continue
      seen.add(row.member_id as string)

      const mem = row.members as Record<string, unknown> | null
      if (!mem) continue
      const ch = mem.chapters as Record<string, unknown> | null

      const member: MemberWithChapter = {
        ...rowToMember(mem),
        chapter: ch ? rowToChapter(ch) : null,
      }
      const invoice = rowToInvoice(row)
      const periodEnd = new Date(invoice.periodEnd)
      const daysUntilDue = Math.ceil((periodEnd.getTime() - today.getTime()) / 86400000)

      result.push({ ...member, lastInvoice: invoice, daysUntilDue })
    }
    return result
  },
}
