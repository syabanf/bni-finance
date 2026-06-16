// ===========================================================================
// xendit-create-payment
// Membuat pembayaran Xendit (Virtual Account / QRIS) untuk sebuah invoice.
// Secret key hanya hidup di server (Supabase secret) — tidak pernah di browser.
//
// Secrets:
//   XENDIT_SECRET_KEY           (wajib)
//   XENDIT_VA_EXPIRY_HOURS      (opsional, default 24)
// ===========================================================================
import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

const supabase = createClient(
  Deno.env.get('SUPABASE_URL')!,
  Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')!,
)

const XENDIT_KEY = Deno.env.get('XENDIT_SECRET_KEY') ?? ''
const VA_EXPIRY_HOURS = Number(Deno.env.get('XENDIT_VA_EXPIRY_HOURS') ?? 24)

const ALLOWED_BANKS = ['BCA', 'BNI', 'MANDIRI', 'BRI']

const CORS = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Headers': 'authorization, x-client-info, apikey, content-type',
  'Access-Control-Allow-Methods': 'POST, OPTIONS',
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { ...CORS, 'Content-Type': 'application/json' },
  })
}

// Basic auth Xendit: base64(secretKey + ':')
function xenditAuth(): string {
  return 'Basic ' + btoa(`${XENDIT_KEY}:`)
}

Deno.serve(async (req) => {
  if (req.method === 'OPTIONS') return new Response('ok', { headers: CORS })
  if (req.method !== 'POST') return json({ error: 'Method not allowed' }, 405)
  if (!XENDIT_KEY) return json({ error: 'XENDIT_SECRET_KEY belum diset' }, 500)

  let payload: { invoiceId?: string; method?: string; bank?: string }
  try {
    payload = await req.json()
  } catch {
    return json({ error: 'Body JSON tidak valid' }, 400)
  }

  const { invoiceId, method } = payload
  const bank = (payload.bank ?? '').toUpperCase()
  if (!invoiceId || !method) return json({ error: 'invoiceId & method wajib' }, 400)
  if (method !== 'va' && method !== 'qris') return json({ error: 'method harus va | qris' }, 400)
  if (method === 'va' && !ALLOWED_BANKS.includes(bank)) {
    return json({ error: `bank harus salah satu: ${ALLOWED_BANKS.join(', ')}` }, 400)
  }

  // Ambil invoice
  const { data: inv, error: invErr } = await supabase
    .from('invoices')
    .select('id, number, amount, status, members(name)')
    .eq('id', invoiceId)
    .single()
  if (invErr || !inv) return json({ error: 'Invoice tidak ditemukan' }, 404)
  if (inv.status === 'paid' || inv.status === 'cancelled') {
    return json({ error: `Invoice berstatus ${inv.status}, tidak bisa dibayar` }, 409)
  }

  // Referensi unik per pembuatan — Xendit menolak external_id/reference_id yang
  // sudah punya VA/QR aktif, jadi jangan pakai nomor invoice mentah berulang.
  const externalId = `${inv.number}-${method}-${crypto.randomUUID().slice(0, 8)}`
  const amount = inv.amount as number
  const customerName = ((inv.members as Record<string, unknown> | null)?.name as string) ?? 'BNI Member'

  try {
    if (method === 'va') {
      const expiry = new Date(Date.now() + VA_EXPIRY_HOURS * 3600_000).toISOString()
      const res = await fetch('https://api.xendit.co/callback_virtual_accounts', {
        method: 'POST',
        headers: { Authorization: xenditAuth(), 'Content-Type': 'application/json' },
        body: JSON.stringify({
          external_id: externalId,
          bank_code: bank,
          name: customerName,
          is_closed: true,
          is_single_use: true,
          expected_amount: amount,
          expiration_date: expiry,
        }),
      })
      const va = await res.json()
      if (!res.ok) return json({ error: 'Xendit VA gagal', detail: va }, 502)

      await supabase
        .from('invoices')
        .update({
          payment_provider: 'xendit',
          xendit_external_id: externalId,
          xendit_payment_id: va.id,
          xendit_payment_method: 'va',
          xendit_va_bank: va.bank_code,
          xendit_va_number: va.account_number,
          xendit_payment_status: 'PENDING',
          xendit_expires_at: expiry,
          xendit_qris_string: null,
        })
        .eq('id', invoiceId)

      return json({
        method: 'va',
        bank: va.bank_code,
        vaNumber: va.account_number,
        amount,
        expiresAt: expiry,
      })
    }

    // QRIS — Xendit membatasi nominal QRIS maksimal Rp 10.000.000 per transaksi
    if (amount > 10_000_000) {
      return json({ error: 'Nominal melebihi batas QRIS (maks Rp 10.000.000). Gunakan Virtual Account.' }, 422)
    }
    const res = await fetch('https://api.xendit.co/qr_codes', {
      method: 'POST',
      headers: {
        Authorization: xenditAuth(),
        'Content-Type': 'application/json',
        'api-version': '2022-07-31',
      },
      body: JSON.stringify({
        reference_id: externalId,
        type: 'DYNAMIC',
        currency: 'IDR',
        amount,
      }),
    })
    const qr = await res.json()
    if (!res.ok) return json({ error: 'Xendit QRIS gagal', detail: qr }, 502)

    await supabase
      .from('invoices')
      .update({
        payment_provider: 'xendit',
        xendit_external_id: externalId,
        xendit_payment_id: qr.id,
        xendit_payment_method: 'qris',
        xendit_qris_string: qr.qr_string,
        xendit_payment_status: 'PENDING',
        xendit_expires_at: qr.expires_at ?? null,
        xendit_va_bank: null,
        xendit_va_number: null,
      })
      .eq('id', invoiceId)

    return json({
      method: 'qris',
      qrString: qr.qr_string,
      amount,
      expiresAt: qr.expires_at ?? null,
    })
  } catch (e) {
    return json({ error: 'Gagal memanggil Xendit', detail: String(e) }, 500)
  }
})
