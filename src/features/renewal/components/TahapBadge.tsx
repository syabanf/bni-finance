import { Badge } from '@/components/ui'
import type { RenewalStage } from '@/types'

type Nada = 'gray' | 'green' | 'red' | 'amber' | 'blue' | 'purple'

/** Urutan alur renewal klien, dipakai juga untuk pilihan saring tahap. */
export const TAHAP: { value: RenewalStage; label: string; tone: Nada }[] = [
  { value: 'pending', label: 'Menunggu jawaban MC', tone: 'gray' },
  { value: 'unsure', label: 'Belum pasti', tone: 'amber' },
  { value: 'will_not', label: 'Tidak diperpanjang', tone: 'red' },
  { value: 'will_renew', label: 'Siap diminta invoice', tone: 'blue' },
  { value: 'invoice_requested', label: 'Invoice diminta', tone: 'amber' },
  { value: 'invoice_draft', label: 'Invoice disiapkan', tone: 'amber' },
  { value: 'invoiced', label: 'Belum dibayar', tone: 'amber' },
  { value: 'proof_pending', label: 'Bukti menunggu finance', tone: 'purple' },
  { value: 'paid', label: 'Lunas', tone: 'green' },
  { value: 'submitted', label: 'Diajukan ke BNI Global', tone: 'green' },
]

const PER_TAHAP = new Map(TAHAP.map((t) => [t.value, t]))

export function TahapBadge({ tahap }: { tahap: RenewalStage }) {
  const t = PER_TAHAP.get(tahap)
  return <Badge tone={t?.tone ?? 'gray'}>{t?.label ?? tahap}</Badge>
}
