import type { InvoiceWithRelations } from '@/types'
import { formatCurrency, formatDate } from './format'

/**
 * Normalize a phone number to wa.me international format (digits only, no '+').
 * Assumes Indonesia (62) when no country code is present.
 */
export function normalizePhone(phone?: string | null): string | null {
  if (!phone) return null
  let d = phone.replace(/\D/g, '')
  if (!d) return null
  if (d.startsWith('0')) d = '62' + d.slice(1)
  else if (!d.startsWith('62')) d = '62' + d
  return d
}

/** Compose the WhatsApp message body for an invoice (tailored by status). */
export function buildInvoiceWhatsAppMessage(inv: InvoiceWithRelations): string {
  const name = inv.member?.name ?? 'Bapak/Ibu'
  const typeLabel = inv.type === 'registration' ? 'pendaftaran' : 'perpanjangan (renewal)'

  if (inv.status === 'paid') {
    return [
      `Halo ${name},`,
      ``,
      `Terima kasih. Pembayaran invoice *${inv.number}* (${typeLabel} keanggotaan BNI) sebesar *${formatCurrency(inv.amount)}* telah kami terima. ✅`,
      ``,
      `Keanggotaan Anda berlaku hingga ${formatDate(inv.periodEnd)}.`,
    ].join('\n')
  }

  const lines = [
    `Halo ${name},`,
    ``,
    `Berikut invoice ${typeLabel} keanggotaan BNI Anda:`,
    `• No. Invoice: ${inv.number}`,
    `• Nominal: ${formatCurrency(inv.amount)}`,
    `• Jatuh tempo: ${formatDate(inv.dueDate)}`,
  ]
  if (inv.paperIdPaymentUrl) {
    lines.push(``, `Silakan lakukan pembayaran melalui link berikut:`, inv.paperIdPaymentUrl)
  }
  lines.push(``, `Terima kasih.`)
  return lines.join('\n')
}

/** Full wa.me click-to-chat URL, or null if the member has no usable phone. */
export function buildInvoiceWhatsAppUrl(inv: InvoiceWithRelations): string | null {
  const phone = normalizePhone(inv.member?.phone)
  if (!phone) return null
  return `https://wa.me/${phone}?text=${encodeURIComponent(buildInvoiceWhatsAppMessage(inv))}`
}
