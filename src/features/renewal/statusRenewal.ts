import { invoiceService, renewalService } from '@/services'
import { statusRenewal, type StatusRenewal } from '@/lib/status'

/**
 * Memuat keadaan renewal semua member yang pernah dimintai konfirmasi.
 *
 * Invoice yang dibaca hanya renewal yang terbit sejak permintaan terakhir
 * paling lama di antara mereka: yang lebih tua tidak mungkin menghitung, dan
 * daftar invoice dibatasi satu halaman oleh server.
 */
export async function muatStatusRenewal(): Promise<Map<string, StatusRenewal>> {
  const permintaan = await renewalService.list()
  const terakhir = new Map<string, string>()
  for (const r of permintaan) {
    const ada = terakhir.get(r.memberId)
    if (!ada || r.requestedAt > ada) terakhir.set(r.memberId, r.requestedAt)
  }
  const sejak = [...terakhir.values()].sort()[0]
  const invoices = sejak ? await invoiceService.list({ type: 'renewal', issuedFrom: sejak.slice(0, 10) }) : []
  return statusRenewal(permintaan, invoices)
}
