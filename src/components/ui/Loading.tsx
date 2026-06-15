import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/cn'

export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={cn('h-5 w-5 animate-spin text-brand-500', className)} />
}

/** Shimmering placeholder block. */
export function Skeleton({ className }: { className?: string }) {
  return (
    <div className={cn('relative overflow-hidden rounded-md bg-ink-100', className)}>
      <div className="absolute inset-0 -translate-x-full animate-shimmer bg-gradient-to-r from-transparent via-white/60 to-transparent" />
    </div>
  )
}

/** Full-area centered spinner for page/section loads. */
export function LoadingState({ label = 'Memuat…' }: { label?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16 text-ink-400">
      <Spinner className="h-6 w-6" />
      <span className="text-sm">{label}</span>
    </div>
  )
}

/** Reusable skeleton rows for tables. */
export function TableSkeleton({ rows = 6, cols = 5 }: { rows?: number; cols?: number }) {
  return (
    <div className="divide-y divide-ink-100">
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex items-center gap-4 px-5 py-4">
          {Array.from({ length: cols }).map((_, c) => (
            <Skeleton
              key={c}
              className={cn('h-4', c === 0 ? 'w-40' : 'flex-1', c === cols - 1 && 'w-20 flex-none')}
            />
          ))}
        </div>
      ))}
    </div>
  )
}
