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
  registration: ['visitor', 'pending', 'new_member'],
  renewal: ['active', 'inactive', 'new_member'],
}

/**
 * Alasan invoice renewal belum boleh terbit, dari jawaban konfirmasi terakhir;
 * null berarti boleh. Cermin periksaKonfirmasiRenewal di server.
 */
export function alasanBelumKonfirmasi(jawaban: StatusRenewal | null): string | null {
  switch (jawaban) {
    case 'will_renew':
      return null
    case 'sudah_ditagih':
      return 'Member ini sudah punya invoice renewal untuk konfirmasi terakhirnya.'
    case null:
    case 'belum_diminta':
      return 'Member ini belum dimintai konfirmasi renewal; minta konfirmasi di halaman Konfirmasi Renewal dulu.'
    case 'pending':
      return 'Konfirmasi renewal member ini belum dijawab MC.'
    case 'will_not':
      return 'MC menjawab member ini tidak memperpanjang; invoice renewal tidak diterbitkan.'
    default:
      return 'Jawaban MC atas konfirmasi renewal member ini masih belum pasti.'
  }
}

/**
 * Keadaan renewal seorang member: jawaban MC pada permintaan konfirmasi
 * TERAKHIR-nya, atau "sudah_ditagih" bila jawabannya Diterima dan invoice
 * renewal-nya sudah terbit sejak permintaan itu. Member tanpa permintaan sama
 * sekali tidak ada di peta; anggap "belum_diminta".
 */
export type StatusRenewal = RenewalAnswer | 'belum_diminta' | 'sudah_ditagih'

/** Lencana pendek di baris tabel; alasan panjangnya di alasanBelumKonfirmasi. */
export const LABEL_KONFIRMASI: Record<StatusRenewal, string> = {
  belum_diminta: 'Belum diminta konfirmasi',
  pending: 'Menunggu jawaban MC',
  will_renew: 'Dikonfirmasi MC',
  will_not: 'MC: tidak perpanjang',
  unsure: 'MC: belum pasti',
  sudah_ditagih: 'Sudah ditagih',
}

/**
 * Hanya permintaan terakhir yang dihitung, sama seperti periksaKonfirmasiRenewal
 * di server: jawaban "Diterima" tahun lalu tidak membuka renewal tahun ini.
 *
 * "sudah_ditagih" menahan satu jawaban Diterima dipakai dua kali. Server belum
 * menolak invoice renewal kedua untuk konfirmasi yang sama, jadi tanpa ini
 * member yang ditagih dari Buat Invoice muncul lagi di Perlu Tindakan.
 */
export function statusRenewal(
  permintaan: { memberId: string; requestedAt: string; answer: RenewalAnswer }[],
  invoiceRenewal: { memberId: string; status: InvoiceStatus; createdAt: string }[],
): Map<string, StatusRenewal> {
  const terakhir = new Map<string, { requestedAt: string; answer: RenewalAnswer }>()
  for (const r of permintaan) {
    const ada = terakhir.get(r.memberId)
    if (!ada || r.requestedAt > ada.requestedAt) terakhir.set(r.memberId, r)
  }
  const hasil = new Map<string, StatusRenewal>()
  for (const [id, r] of terakhir) {
    const ditagih =
      r.answer === 'will_renew' &&
      invoiceRenewal.some(
        (i) => i.memberId === id && i.status !== 'cancelled' && i.status !== 'terminated' && Date.parse(i.createdAt) >= Date.parse(r.requestedAt),
      )
    hasil.set(id, ditagih ? 'sudah_ditagih' : r.answer)
  }
  return hasil
}

export function bolehDitagih(status: MemberStatus, tipe: InvoiceType): boolean {
  return STATUS_UNTUK_TIPE[tipe].includes(status)
}
