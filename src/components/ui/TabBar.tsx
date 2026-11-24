import { cn } from '@/lib/cn'

/** Deret tab di atas tabel, dengan gaya yang sama dengan tab status invoice. */
export function TabBar<T extends string>({
  tabs,
  value,
  onChange,
}: {
  tabs: { value: T; label: string; count?: number }[]
  value: T
  onChange: (value: T) => void
}) {
  return (
    <div className="flex gap-1 overflow-x-auto border-b border-ink-100 px-4 pt-3 pb-0">
      {tabs.map((tab) => (
        <button
          key={tab.value}
          type="button"
          onClick={() => onChange(tab.value)}
          className={cn(
            'flex shrink-0 items-center gap-1.5 rounded-t-lg border-b-2 px-3 pb-2.5 pt-2 text-[13px] font-medium transition-colors',
            value === tab.value ? 'border-brand-500 text-brand-600' : 'border-transparent text-ink-500 hover:text-ink-800',
          )}
        >
          {tab.label}
          {!!tab.count && (
            <span
              className={cn(
                'rounded-full px-1.5 py-0.5 text-[11px] font-semibold leading-none',
                value === tab.value ? 'bg-brand-100 text-brand-600' : 'bg-ink-100 text-ink-500',
              )}
            >
              {tab.count}
            </span>
          )}
        </button>
      ))}
    </div>
  )
}
