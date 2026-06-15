import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

const supabase = createClient(
  Deno.env.get('SUPABASE_URL')!,
  Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')!,
)

Deno.serve(async () => {
  // Baca konfigurasi timing dari app_settings
  const { data: settingRow } = await supabase
    .from('app_settings')
    .select('key, value')
    .in('key', ['invoice_draft_days_before'])
  const settingsMap: Record<string, string> = {}
  for (const r of settingRow ?? []) settingsMap[r.key] = r.value
  const draftDaysBefore = Number(settingsMap['invoice_draft_days_before'] ?? 30)

  const today = new Date()
  const todayStr = today.toISOString().slice(0, 10)
  const target = new Date(today)
  target.setDate(target.getDate() + draftDaysBefore)
  const targetDate = target.toISOString().slice(0, 10)

  // Ambil semua member yang renewal_date-nya antara hari ini s/d H+N
  // (gunakan <= agar tidak terlewat jika cron sempat tidak jalan)
  const { data: members, error: mErr } = await supabase
    .from('members')
    .select('id, name, chapter_id, renewal_date')
    .gte('renewal_date', todayStr)
    .lte('renewal_date', targetDate)
    .eq('status', 'active')

  if (mErr) return new Response(JSON.stringify({ error: mErr.message }), { status: 500 })
  if (!members?.length) return new Response(JSON.stringify({ created: 0, message: 'Tidak ada member dengan renewal H-30 hari ini' }), { status: 200 })

  // Ambil fee renewal
  const { data: fees } = await supabase
    .from('fee_settings')
    .select('renewal_fee')
    .eq('id', 'default')
    .single()
  const renewalFee = fees?.renewal_fee ?? 1500000

  // Cek member yang sudah punya invoice renewal draft/sent/overdue (hindari duplikat)
  const memberIds = members.map((m: Record<string, unknown>) => m.id)
  const { data: existing } = await supabase
    .from('invoices')
    .select('member_id')
    .eq('type', 'renewal')
    .in('status', ['draft', 'sent', 'overdue'])
    .in('member_id', memberIds)

  const alreadyHas = new Set((existing ?? []).map((r: Record<string, unknown>) => r.member_id))

  // Buat invoice draft untuk member yang belum punya
  const toCreate = members.filter((m: Record<string, unknown>) => !alreadyHas.has(m.id))

  // Generate nomor invoice berurutan
  const year = new Date().getFullYear()
  const { count: invoiceCount } = await supabase
    .from('invoices')
    .select('id', { count: 'exact', head: true })
  const baseSeq = (invoiceCount ?? 0) + 1

  const invoiceRows = toCreate.map((m: Record<string, unknown>, i: number) => {
    const renewalDate = m.renewal_date as string
    // period = renewal_date s/d renewal_date + 1 tahun
    const periodStart = renewalDate
    const periodEnd = new Date(renewalDate)
    periodEnd.setFullYear(periodEnd.getFullYear() + 1)
    return {
      number: `INV-${year}-${String(baseSeq + i).padStart(3, '0')}`,
      member_id: m.id,
      chapter_id: m.chapter_id,
      type: 'renewal',
      status: 'draft',
      amount: renewalFee,
      currency: 'IDR',
      period_start: periodStart,
      period_end: periodEnd.toISOString().slice(0, 10),
      due_date: targetDate, // placeholder, dioverride saat send
      notes: `Renewal otomatis — renewal date ${renewalDate}`,
    }
  })

  if (invoiceRows.length === 0) {
    return new Response(JSON.stringify({ created: 0, message: 'Semua member sudah punya invoice renewal aktif' }), { status: 200 })
  }

  const { error: iErr } = await supabase.from('invoices').insert(invoiceRows)
  if (iErr) return new Response(JSON.stringify({ error: iErr.message }), { status: 500 })

  return new Response(JSON.stringify({ created: invoiceRows.length, members: toCreate.map((m: Record<string, unknown>) => m.name) }), { status: 200 })
})
