import { LayoutGrid, Table2 } from 'lucide-react'
import { cn } from '@/lib/cn'
import type { Tampilan } from '@/hooks/useTampilan'

const PILIHAN = [
  { nilai: 'kartu', label: 'Kartu', icon: LayoutGrid },
  { nilai: 'tabel', label: 'Tabel', icon: Table2 },
] as const

/** Pemilih bentuk daftar. Pasangannya hook useTampilan. */
export function TampilanToggle({
  value,
  onChange,
  className,
}: {
  value: Tampilan
  onChange: (v: Tampilan) => void
  className?: string
}) {
  return (
    <div
      role="group"
      aria-label="Tampilan daftar"
      className={cn('inline-flex shrink-0 rounded-xl border border-ink-200 bg-white/70 p-0.5', className)}
    >
      {PILIHAN.map(({ nilai, label, icon: Icon }) => (
        <button
          key={nilai}
          type="button"
          aria-pressed={value === nilai}
          onClick={() => onChange(nilai)}
          className={cn(
            'inline-flex h-8 items-center gap-1.5 rounded-[10px] px-2.5 text-xs font-medium transition-colors',
            value === nilai ? 'bg-brand-500 text-white shadow-sm' : 'text-ink-500 hover:text-ink-800',
          )}
        >
          <Icon className="h-3.5 w-3.5" />
          {label}
        </button>
      ))}
    </div>
  )
}

/**
 * Kelas untuk dua bentuk daftar yang sama-sama dirender. Tabel diberi lebar
 * minimum supaya di HP ia menggulir mendatar, bukan memeras kolomnya.
 */
export function kelasTampilan(tampilan: Tampilan) {
  return {
    kartu: tampilan === 'kartu' ? '' : 'hidden',
    tabel: tampilan === 'tabel' ? 'overflow-x-auto [&_table]:min-w-[720px]' : 'hidden',
  }
}
