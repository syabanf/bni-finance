// ===========================================================================
// get-public-invoice
// Public, READ-ONLY endpoint for the /pay/:id page.
//
// Returns ONLY the minimal fields a payer needs for ONE invoice, using the
// service-role key. This lets RLS on invoices / members / app_settings be
// locked to authenticated-only (run supabase/rls.sql) WITHOUT breaking the
// public payment page — and it never exposes member email / phone / company.
//
// Rollout:
//   1. supabase functions deploy get-public-invoice
//   2. switch PublicPaymentPage to fetch via this function (instead of the
//      anon Supabase client)
//   3. run supabase/rls.sql to lock the tables
// ===========================================================================
import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

const supabase = createClient(
  Deno.env.get('SUPABASE_URL')!,
  Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')!,
)

const CORS = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Headers': 'authorization, x-client-info, apikey, content-type',
  'Access-Control-Allow-Methods': 'POST, OPTIONS',
}
const json = (b: unknown, s = 200) =>
  new Response(JSON.stringify(b), { status: s, headers: { ...CORS, 'Content-Type': 'application/json' } })

Deno.serve(async (req) => {
  if (req.method === 'OPTIONS') return new Response('ok', { headers: CORS })
  if (req.method !== 'POST') return json({ error: 'Method not allowed' }, 405)

  let invoiceId = ''
  try {
    invoiceId = (await req.json())?.invoiceId ?? ''
  } catch {
    return json({ error: 'Body JSON tidak valid' }, 400)
  }
  if (!invoiceId) return json({ error: 'invoiceId wajib' }, 400)

  // Minimal projection — NOTE: members(name) only, never email/phone/company.
  const { data: inv, error } = await supabase
    .from('invoices')
    .select(
      `id, number, type, amount, currency, status,
       due_date, period_start, period_end, created_at, notes,
       payment_provider, paper_id_payment_url,
       xendit_payment_method, xendit_va_bank, xendit_va_number,
       xendit_qris_string, xendit_payment_status, xendit_expires_at,
       members(name), chapters(display_name)`,
    )
    .eq('id', invoiceId)
    .maybeSingle()

  if (error || !inv) return json({ error: 'Invoice tidak ditemukan' }, 404)

  const { data: spm } = await supabase
    .from('app_settings')
    .select('value')
    .eq('key', 'self_payment_mode')
    .maybeSingle()

  return json({ invoice: inv, selfPaymentMode: spm?.value === 'true' })
})
