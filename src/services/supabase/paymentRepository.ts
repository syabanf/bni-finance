import { supabase } from '@/lib/supabase'
import type { PaymentRepository } from '@/services/types'
import type { Invoice, InvoiceStatus, InvoiceType, Member, MemberStatus, Payment, PaymentWithInvoice } from '@/types'

function rowToPayment(r: Record<string, unknown>): Payment {
  return {
    id: r.id as string,
    invoiceId: r.invoice_id as string,
    amount: r.amount as number,
    paidAt: r.paid_at as string,
    paymentMethod: r.payment_method as string | undefined,
    paperIdPaymentId: r.paper_id_payment_id as string | undefined,
    paperIdStatus: r.paper_id_status as string | undefined,
    proofUrl: r.proof_url as string | undefined,
    note: r.note as string | undefined,
    createdAt: r.created_at as string,
  }
}

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

export const supabasePaymentRepository: PaymentRepository = {
  async list() {
    const { data, error } = await supabase
      .from('payments')
      .select('*, invoices(*, members(*))')
      .order('paid_at', { ascending: false })
    if (error) throw new Error(error.message)
    return (data ?? []).map((r: Record<string, unknown>) => {
      const inv = r.invoices as Record<string, unknown> | null
      const mem = inv ? (inv.members as Record<string, unknown> | null) : null
      return {
        ...rowToPayment(r),
        invoice: inv ? rowToInvoice(inv) : null,
        member: mem ? rowToMember(mem) : null,
      } satisfies PaymentWithInvoice
    })
  },

  async listByInvoice(invoiceId) {
    const { data, error } = await supabase
      .from('payments')
      .select('*, invoices(*, members(*))')
      .eq('invoice_id', invoiceId)
      .order('paid_at', { ascending: false })
    if (error) throw new Error(error.message)
    return (data ?? []).map((r: Record<string, unknown>) => {
      const inv = r.invoices as Record<string, unknown> | null
      const mem = inv ? (inv.members as Record<string, unknown> | null) : null
      return {
        ...rowToPayment(r),
        invoice: inv ? rowToInvoice(inv) : null,
        member: mem ? rowToMember(mem) : null,
      } satisfies PaymentWithInvoice
    })
  },

  async uploadProof(file) {
    const ext = file.name.split('.').pop()?.toLowerCase() || 'bin'
    const path = `proofs/${Date.now()}-${Math.random().toString(36).slice(2, 8)}.${ext}`
    const { error } = await supabase.storage
      .from('payment-proofs')
      .upload(path, file, { cacheControl: '3600', upsert: false })
    if (error) throw new Error('Gagal mengunggah bukti pembayaran: ' + error.message)
    const { data } = supabase.storage.from('payment-proofs').getPublicUrl(path)
    return data.publicUrl
  },
}
