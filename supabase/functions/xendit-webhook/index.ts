// ===========================================================================
// xendit-webhook
// Menerima callback Xendit (VA paid / QRIS paid) → tandai invoice Lunas otomatis.
// Daftarkan URL fungsi ini di Xendit Dashboard → Settings → Callbacks.
//
// Secrets:
//   XENDIT_CALLBACK_TOKEN  (wajib — verifikasi header x-callback-token)
// ===========================================================================
import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

const supabase = createClient(
  Deno.env.get('SUPABASE_URL')!,
  Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')!,
)

const CALLBACK_TOKEN = Deno.env.get('XENDIT_CALLBACK_TOKEN') ?? ''

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

Deno.serve(async (req) => {
  if (req.method !== 'POST') return json({ error: 'Method not allowed' }, 405)

  // Verifikasi token callback Xendit
  const token = req.headers.get('x-callback-token') ?? ''
  if (!CALLBACK_TOKEN || token !== CALLBACK_TOKEN) {
    return json({ error: 'Unauthorized' }, 401)
  }

  let body: Record<string, unknown>
  try {
    body = await req.json()
  } catch {
    return json({ error: 'Body JSON tidak valid' }, 400)
  }

  // Normalisasi: VA callback memakai external_id di root,
  // QRIS callback memakai reference_id (kadang di body.data).
  const data = (body.data as Record<string, unknown>) ?? body
  const externalId =
    (body.external_id as string) ??
    (data.reference_id as string) ??
    (data.external_id as string) ??
    null
  const xenditPaymentId =
    (body.payment_id as string) ??
    (data.payment_id as string) ??
    (body.id as string) ??
    (data.id as string) ??
    null
  const paidAmount = Number((body.amount as number) ?? (data.amount as number) ?? 0)

  if (!externalId && !xenditPaymentId) {
    return json({ received: true, skipped: 'tidak ada external_id/payment_id' })
  }

  // Cari invoice — utamakan external_id (= nomor invoice), fallback ke payment id
  let query = supabase.from('invoices').select('*')
  query = externalId ? query.eq('xendit_external_id', externalId) : query.eq('xendit_payment_id', xenditPaymentId)
  const { data: inv } = await query.maybeSingle()

  if (!inv) return json({ received: true, skipped: 'invoice tidak ditemukan' })
  if (inv.status === 'paid') return json({ received: true, alreadyPaid: true })

  const now = new Date().toISOString()
  const amount = paidAmount || (inv.amount as number)

  await supabase
    .from('invoices')
    .update({
      status: 'paid',
      paid_at: now,
      paid_amount: amount,
      xendit_payment_status: 'PAID',
    })
    .eq('id', inv.id)

  await supabase.from('payments').insert({
    invoice_id: inv.id,
    amount,
    paid_at: now,
    payment_method: inv.xendit_payment_method === 'qris' ? 'qris' : 'virtual_account',
    xendit_payment_id: xenditPaymentId,
    xendit_status: 'PAID',
  })

  await supabase.from('invoice_audit_log').insert({
    invoice_id: inv.id,
    action: 'paid',
    old_status: inv.status,
    new_status: 'paid',
    actor_name: 'Xendit',
    notes: `Pembayaran ${inv.xendit_payment_method?.toUpperCase() ?? ''} diterima via Xendit`,
  })

  // Renewal otomatis: majukan renewal_date member ke akhir periode invoice ini (+1 tahun).
  let newRenewalDate: string | null = null
  if (inv.member_id) {
    if (inv.period_end) {
      newRenewalDate = inv.period_end as string
    } else {
      const { data: mem } = await supabase
        .from('members')
        .select('renewal_date')
        .eq('id', inv.member_id)
        .maybeSingle()
      const base = (mem?.renewal_date as string) ?? now.slice(0, 10)
      const d = new Date(base)
      d.setFullYear(d.getFullYear() + 1)
      newRenewalDate = d.toISOString().slice(0, 10)
    }
    await supabase.from('members').update({ renewal_date: newRenewalDate }).eq('id', inv.member_id)
  }

  return json({ received: true, invoiceId: inv.id, status: 'paid', renewalDate: newRenewalDate })
})
