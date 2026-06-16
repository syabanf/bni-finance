import { supabase } from '@/lib/supabase'
import { getAppSetting } from './settingsRepository'

export const VA_BANKS = ['BCA', 'BNI', 'MANDIRI', 'BRI'] as const
export type VaBank = (typeof VA_BANKS)[number]

export interface XenditPaymentResult {
  method: 'va' | 'qris'
  bank?: string
  vaNumber?: string
  qrString?: string
  amount: number
  expiresAt: string | null
}

/** Apakah Self Payment Mode (Xendit) aktif. */
export async function isSelfPaymentMode(): Promise<boolean> {
  const v = await getAppSetting('self_payment_mode')
  return v === 'true'
}

/**
 * Minta server (Edge Function) membuat pembayaran Xendit untuk invoice.
 * Secret key tidak pernah ada di browser — semua dilakukan di Edge Function.
 */
export async function createXenditPayment(
  invoiceId: string,
  method: 'va' | 'qris',
  bank?: VaBank,
): Promise<XenditPaymentResult> {
  const { data, error } = await supabase.functions.invoke('xendit-create-payment', {
    body: { invoiceId, method, bank },
  })
  if (error) {
    // supabase.functions.invoke menyembunyikan body saat non-2xx — baca dari context
    let msg = error.message
    try {
      const ctx = (error as { context?: Response }).context
      const body = ctx ? await ctx.json() : null
      if (body?.error) msg = body.error as string
    } catch {
      // abaikan — pakai pesan default
    }
    throw new Error(msg)
  }
  if (data?.error) throw new Error(data.error as string)
  return data as XenditPaymentResult
}
