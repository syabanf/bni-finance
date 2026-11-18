import type { LucideIcon } from 'lucide-react'
import { ArrowUpRight } from 'lucide-react'
import { cn } from '@/lib/cn'

type IconTone = 'brand' | 'amber' | 'blue' | 'green' | 'red'

const iconTones: Record<IconTone, string> = {
  brand: 'bg-brand-500/10 text-brand-500',
  amber: 'bg-amber-500/15 text-amber-600',
  blue: 'bg-blue-500/10 text-blue-500',
  green: 'bg-emerald-500/10 text-emerald-600',
  red: 'bg-red-500/10 text-red-500',
}

interface StatCardProps {
  icon: LucideIcon
  iconTone?: IconTone
  value: string | number
  label: string
  hint?: string
  onClick?: () => void
}

export function StatCard({ icon: Icon, iconTone = 'brand', value, label, hint, onClick }: StatCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!onClick}
      className={cn(
        'surface group flex min-w-0 flex-col p-4 text-left transition-shadow sm:p-5',
        onClick ? 'hover:shadow-card-hover cursor-pointer' : 'cursor-default',
      )}
    >
      <div className="flex items-start justify-between">
        <div className={cn('flex h-11 w-11 items-center justify-center rounded-xl ring-1 ring-inset ring-white/70', iconTones[iconTone])}>
          <Icon className="h-[22px] w-[22px]" strokeWidth={2} />
        </div>
      </div>
      <div className="mt-4 text-[22px] font-bold leading-none tracking-tight text-ink-900 sm:text-[28px]">{value}</div>
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
