import type { InvoiceType } from '@/types'
import { cn } from '@/lib/cn'

export type InvoiceTypeFilterValue = InvoiceType | 'all'

const PILIHAN: { value: InvoiceTypeFilterValue; label: string }[] = [
  { value: 'all', label: 'Semua Tipe' },
  { value: 'renewal', label: 'Renewal' },
  { value: 'registration', label: 'Pendaftaran' },
]

/**
 * Saringan tipe invoice untuk dashboard dan laporan.
 *
 * Harga renewal dan pendaftaran jauh berbeda, jadi total gabungannya tidak
 * menjawab pertanyaan yang sering diajukan: berapa yang masuk dari
 * pendaftaran, berapa renewal yang masih menunggak. Bentuknya sama dengan
 * tombol periode di laporan supaya dua saringan itu terbaca sebagai satu baris.
 */
export function InvoiceTypeFilter({
  value,
  onChange,
}: {
  value: InvoiceTypeFilterValue
  onChange: (v: InvoiceTypeFilterValue) => void
}) {
  return (
    <div className="flex flex-wrap gap-1.5" role="group" aria-label="Tipe invoice">
      {PILIHAN.map((p) => (
        <button
          key={p.value}
          type="button"
          onClick={() => onChange(p.value)}
          className={cn(
            'rounded-lg px-3 py-1.5 text-[13px] font-medium transition-colors',
            value === p.value ? 'bg-brand-500 text-white' : 'bg-ink-100 text-ink-600 hover:bg-ink-200',
          )}
        >
          {p.label}
        </button>
      ))}
    </div>
  )
}
