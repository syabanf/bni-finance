import { useId } from 'react'

export interface DonutSegment {
  label: string
  value: number
  color: string
}

interface DonutChartProps {
  data: DonutSegment[]
  size?: number
  thickness?: number
  centerLabel?: string
  centerValue?: string | number
  /** When set, segments + legend become clickable (drill-down). */
  onSelect?: (index: number) => void
}

export function DonutChart({
  data,
  size = 200,
  thickness = 26,
  centerLabel,
  centerValue,
  onSelect,
}: DonutChartProps) {
  const id = useId()
  const total = data.reduce((acc, d) => acc + d.value, 0)
  const radius = (size - thickness) / 2
  const cx = size / 2
  const circumference = 2 * Math.PI * radius
  const gap = total > 1 ? 2 : 0 // px gap between segments

  let offset = 0
  const segments = data.map((d) => {
    const fraction = total === 0 ? 0 : d.value / total
    const length = Math.max(fraction * circumference - gap, 0)
    const seg = {
      ...d,
      dasharray: `${length} ${circumference - length}`,
      dashoffset: -offset,
      pct: Math.round(fraction * 100),
    }
    offset += fraction * circumference
    return seg
  })

  return (
    <div className="flex flex-col items-center gap-5">
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size} className="-rotate-90">
          {/* track */}
          <circle
            cx={cx}
            cy={cx}
            r={radius}
            fill="none"
            stroke="currentColor"
            strokeWidth={thickness}
            className="text-ink-100"
          />
          {total > 0 &&
            segments.map((s, i) => (
              <circle
                key={`${id}-${i}`}
                cx={cx}
                cy={cx}
                r={radius}
                fill="none"
                stroke={s.color}
                strokeWidth={thickness}
                strokeDasharray={s.dasharray}
                strokeDashoffset={s.dashoffset}
                strokeLinecap="round"
                onClick={onSelect ? () => onSelect(i) : undefined}
                style={onSelect ? { cursor: 'pointer' } : undefined}
                className={onSelect ? 'transition-opacity hover:opacity-80' : undefined}
              />
            ))}
        </svg>
        {(centerValue !== undefined || centerLabel) && (
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            {centerValue !== undefined && (
              <span className="text-2xl font-bold text-ink-900">{centerValue}</span>
            )}
            {centerLabel && <span className="text-xs text-ink-400">{centerLabel}</span>}
          </div>
        )}
      </div>

      <div className="grid w-full grid-cols-2 gap-x-4 gap-y-1">
        {data.map((d, i) => {
          const inner = (
            <>
              <span className="h-2.5 w-2.5 flex-shrink-0 rounded-full" style={{ backgroundColor: d.color }} />
              <span className="text-ink-600">{d.label}</span>
              <span className="ml-auto font-semibold text-ink-900">{d.value}</span>
            </>
          )
          return onSelect ? (
            <button
              key={d.label}
              type="button"
              onClick={() => onSelect(i)}
              className="-mx-1.5 flex items-center gap-2 rounded-lg px-1.5 py-1 text-sm transition-colors hover:bg-ink-50"
            >
              {inner}
            </button>
          ) : (
            <div key={d.label} className="flex items-center gap-2 py-1 text-sm">
              {inner}
            </div>
          )
        })}
      </div>
    </div>
  )
}
