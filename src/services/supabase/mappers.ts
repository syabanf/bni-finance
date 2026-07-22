// Single source of truth for Supabase row → domain-model mapping. Previously
// each repository kept its own copies of these, which drifted (invoice/payment
// repos dropped members.company/business_field and the xendit_* columns).
import type {
  Chapter,
  Invoice,
  InvoiceStatus,
  InvoiceType,
  Member,
  MemberStatus,
  Payment,
} from '@/types'

type Row = Record<string, unknown>

export function rowToChapter(r: Row): Chapter {
  return {
    id: r.id as string,
    name: r.name as string,
    displayName: r.display_name as string,
    areaName: r.area_name as string | undefined,
    cityName: r.city_name as string | undefined,
    syncedAt: r.synced_at as string,
  }
}

export function rowToMember(r: Row): Member {
  return {
    id: r.id as string,
    chapterId: r.chapter_id as string,
    name: r.name as string,
    email: r.email as string | undefined,
    phone: r.phone as string | undefined,
    company: r.company as string | undefined,
    businessField: r.business_field as string | undefined,
    status: r.status as MemberStatus,
    joinedDate: r.joined_date as string | null,
    renewalDate: r.renewal_date as string | null,
    syncedAt: r.synced_at as string,
  }
}

export function rowToInvoice(r: Row): Invoice {
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

export function rowToPayment(r: Row): Payment {
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
