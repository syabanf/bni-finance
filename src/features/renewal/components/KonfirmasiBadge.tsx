import { Badge } from '@/components/ui'
import { LABEL_KONFIRMASI, alasanBelumKonfirmasi, type StatusRenewal } from '@/lib/status'

/**
 * Jawaban MC untuk satu member di daftar renewal. Baris yang belum dikonfirmasi
 * tidak bisa dicentang, dan lencana ini menjelaskan sebabnya; tanpa itu kotak
 * centang yang mati terbaca sebagai tombol rusak.
 */
export function KonfirmasiBadge({ jawaban }: { jawaban: StatusRenewal | undefined }) {
  const alasan = alasanBelumKonfirmasi(jawaban ?? null)
  return (
    <span title={alasan ?? undefined}>
      <Badge tone={jawaban === 'will_renew' ? 'green' : jawaban === 'will_not' ? 'red' : jawaban === 'sudah_ditagih' ? 'blue' : 'gray'}>
        {LABEL_KONFIRMASI[jawaban ?? 'belum_diminta']}
      </Badge>
    </span>
  )
}
