import { supabase } from '@/lib/supabase'
import type { PaymentRepository } from '@/services/types'
import type { PaymentWithInvoice } from '@/types'
import { rowToInvoice, rowToMember, rowToPayment } from './mappers'

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
