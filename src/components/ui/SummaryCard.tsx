import { cn } from '@/lib/cn'

type Tone = 'default' | 'brand' | 'amber' | 'red' | 'green' | 'blue'

const toneText: Record<Tone, string> = {
  default: 'text-ink-900',
  brand: 'text-brand-600',
  amber: 'text-amber-600',
  red: 'text-red-600',
  green: 'text-emerald-600',
  blue: 'text-blue-600',
}

interface SummaryCardProps {
  label: string
  value: string | number
  sub?: string
  tone?: Tone
  /** Highlight as the active filter. */
  active?: boolean
  /** When set, the card becomes a clickable filter. */
  onClick?: () => void
}

/** Compact summary stat shown in a row above a list/table. */
export function SummaryCard({ label, value, sub, tone = 'default', active, onClick }: SummaryCardProps) {
  const className = cn(
    'surface flex flex-col gap-0.5 p-4 text-left transition-all',
    onClick && 'cursor-pointer hover:shadow-card-hover',
    active && 'ring-2 ring-brand-500/50',
  )
  const inner = (
    <>
      <span className="text-xs font-medium uppercase tracking-wide text-ink-400">{label}</span>
      <span className={cn('text-2xl font-bold leading-tight', toneText[tone])}>{value}</span>
      {sub && <span className="text-xs text-ink-400">{sub}</span>}
    </>
  )
  return onClick ? (
    <button type="button" onClick={onClick} className={className}>
      {inner}
    </button>
  ) : (
    <div className={className}>{inner}</div>
  )
}
