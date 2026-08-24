import type { InvoiceStatus } from '@/types'

/**
 * Single source of truth for how invoice statuses are presented across the app
 * (badges, tabs, donut, summary cards) — so the same status never shows up
 * under different names/colors.
 */
export const INVOICE_STATUS_LABEL: Record<InvoiceStatus, string> = {
  draft: 'Draft',
  sent: 'Menunggu', // issued, awaiting payment
  paid: 'Lunas',
  overdue: 'Overdue',
  cancelled: 'Dibatalkan',
  // Dibedakan dari "Dibatalkan" dengan sengaja: yang satu tagihan yang ditarik
  // kembali, yang lain keanggotaan yang diputus. Menyamakan namanya di layar
  // menghapus perbedaan yang justru jadi alasan statusnya ditambahkan.
  terminated: 'Diputus',
}

export const INVOICE_STATUS_COLOR: Record<InvoiceStatus, string> = {
  paid: '#10b981',
  sent: '#f59e0b',
  overdue: '#ef4444',
  draft: '#94a3b8',
  cancelled: '#cbd5e1',
  terminated: '#7c3aed',
}

/**
 * "Outstanding" = invoice sudah diterbitkan tapi belum dibayar.
 * Always means sent + overdue — used everywhere outstanding is shown/filtered.
 */
export const OUTSTANDING_STATUSES: InvoiceStatus[] = ['sent', 'overdue']

export function isOutstanding(status: InvoiceStatus): boolean {
  return status === 'sent' || status === 'overdue'
}
