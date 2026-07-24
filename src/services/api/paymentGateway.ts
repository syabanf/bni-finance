import { api } from '@/lib/apiClient'
import { getAppSetting } from '../appSettings'

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

/**
 * Whether Self Payment Mode (Xendit) is switched on.
 *
 * Reads through the composition point, so mock mode answers from localStorage
 * and API mode from the server — no branch needed here.
 */
export async function isSelfPaymentMode(): Promise<boolean> {
  return (await getAppSetting('self_payment_mode')) === 'true'
}

/**
 * Asks the server to open a Xendit charge for an invoice.
 *
 * The endpoint is unauthenticated because the payer isn't a user of the app —
 * the server re-checks Self Payment Mode and the invoice status itself, so a
 * link alone can't create a charge against a paid or cancelled invoice.
 */
export async function createXenditPayment(
  invoiceId: string,
  method: 'va' | 'qris',
  bank?: VaBank,
): Promise<XenditPaymentResult> {
  return api.publicPost<XenditPaymentResult>(
    `/public/invoices/${encodeURIComponent(invoiceId)}/payment`,
    { method, bank },
  )
}

/** The projection the public payment page renders. */
export interface PublicInvoiceView {
  invoice: {
    id: string
    number: string
    type: string
    amount: number
    currency: string
    status: string
    dueDate: string
    periodStart: string
    periodEnd: string
    memberName: string
    chapterName?: string
    notes?: string
    paymentProvider?: string
    paperIdPaymentUrl?: string
    xenditPaymentMethod?: 'va' | 'qris'
    xenditVaBank?: string
    xenditVaNumber?: string
    xenditQrisString?: string
    xenditPaymentStatus?: string
    xenditExpiresAt?: string
    createdAt: string
  }
  selfPaymentMode: boolean
}

export async function fetchPublicInvoice(invoiceId: string): Promise<PublicInvoiceView> {
  return api.publicGet<PublicInvoiceView>(`/public/invoices/${encodeURIComponent(invoiceId)}`)
}
