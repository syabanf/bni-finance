import type { LucideIcon } from 'lucide-react'
import { ArrowUpRight, TrendingDown, TrendingUp, Minus } from 'lucide-react'
import { cn } from '@/lib/cn'

type IconTone = 'brand' | 'amber' | 'blue' | 'green' | 'red'

const iconTones: Record<IconTone, string> = {
  brand: 'bg-brand-50 text-brand-500',
  amber: 'bg-amber-50 text-amber-500',
  blue: 'bg-blue-50 text-blue-500',
  green: 'bg-emerald-50 text-emerald-500',
  red: 'bg-red-50 text-red-500',
}

interface Trend {
  value: number
  /** How to read the trend: 'percent' shows %, 'absolute' shows +N. */
  format?: 'percent' | 'absolute'
  /** Visual intent — green for good, red for bad, gray for neutral. */
  intent?: 'good' | 'bad' | 'neutral'
}

interface StatCardProps {
  icon: LucideIcon
  iconTone?: IconTone
  value: string | number
  label: string
  hint?: string
  trend?: Trend
  onClick?: () => void
}

function TrendPill({ value, format = 'percent', intent = 'neutral' }: Trend) {
  const Icon = value > 0 ? TrendingUp : value < 0 ? TrendingDown : Minus
  const text =
    format === 'percent'
      ? `${value > 0 ? '+' : ''}${value}%`
      : `${value > 0 ? '+' : ''}${value}`
  const tone =
    intent === 'good'
      ? 'bg-emerald-50 text-emerald-600'
      : intent === 'bad'
        ? 'bg-red-50 text-red-500'
        : 'bg-ink-100 text-ink-500'
  return (
    <span className={cn('inline-flex items-center gap-1 rounded-full px-2 py-1 text-xs font-semibold', tone)}>
      <Icon className="h-3.5 w-3.5" />
      {text}
    </span>
  )
}

export function StatCard({ icon: Icon, iconTone = 'brand', value, label, hint, trend, onClick }: StatCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!onClick}
      className={cn(
        'surface group flex flex-col p-5 text-left transition-shadow',
        onClick ? 'hover:shadow-card-hover cursor-pointer' : 'cursor-default',
      )}
    >
      <div className="flex items-start justify-between">
        <div className={cn('flex h-11 w-11 items-center justify-center rounded-xl', iconTones[iconTone])}>
          <Icon className="h-[22px] w-[22px]" strokeWidth={2} />
        </div>
        {trend && <TrendPill {...trend} />}
      </div>
      <div className="mt-4 text-[28px] font-bold leading-none tracking-tight text-ink-900">{value}</div>
      <div className="mt-2 flex items-center gap-1 text-sm font-medium text-ink-500">
        {label}
        {onClick && (
          <ArrowUpRight className="h-3.5 w-3.5 text-ink-300 transition-colors group-hover:text-brand-500" />
        )}
      </div>
      {hint && <div className="mt-0.5 text-xs text-ink-400">{hint}</div>}
    </button>
  )
}
