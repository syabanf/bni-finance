import type { InvoiceStatus, InvoiceType, MemberStatus, RenewalAnswer } from '@/types'

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

/**
 * Status member yang boleh menerima tiap tipe invoice. Cermin dari
 * MemberStatus.BolehDitagih di server: pendaftaran untuk yang belum anggota,
 * renewal untuk yang sudah. Formulir memakainya untuk menyaring pilihan, mock
 * memakainya untuk menolak, supaya keduanya tidak berbeda dari server.
 */
export const STATUS_UNTUK_TIPE: Record<InvoiceType, MemberStatus[]> = {
  registration: ['visitor', 'pending'],
  renewal: ['active', 'inactive'],
}

/**
 * Alasan invoice renewal belum boleh terbit, dari jawaban konfirmasi terakhir;
 * null berarti boleh. Cermin periksaKonfirmasiRenewal di server.
 */
export function alasanBelumKonfirmasi(jawaban: RenewalAnswer | null): string | null {
  switch (jawaban) {
    case 'will_renew':
      return null
    case null:
      return 'Member ini belum dimintai konfirmasi renewal; minta konfirmasi di halaman Konfirmasi Renewal dulu.'
    case 'pending':
      return 'Konfirmasi renewal member ini belum dijawab MC.'
    case 'will_not':
      return 'MC menjawab member ini tidak memperpanjang; invoice renewal tidak diterbitkan.'
    default:
      return 'Jawaban MC atas konfirmasi renewal member ini masih belum pasti.'
  }
}

export function bolehDitagih(status: MemberStatus, tipe: InvoiceType): boolean {
  return STATUS_UNTUK_TIPE[tipe].includes(status)
}
