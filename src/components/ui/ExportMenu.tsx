import { useEffect, useRef, useState } from 'react'
import { ChevronDown, Download, FileSpreadsheet, FileText } from 'lucide-react'
import { cn } from '@/lib/cn'

interface ExportMenuProps {
  onCsv: () => void
  onPdf: () => void
  disabled?: boolean
  label?: string
}

/** "Export ▾" button with CSV / PDF options. */
export function ExportMenu({ onCsv, onPdf, disabled, label = 'Export' }: ExportMenuProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  const pick = (fn: () => void) => {
    setOpen(false)
    fn()
  }

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        disabled={disabled}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-2 rounded-xl border border-ink-200 bg-white px-3.5 py-2 text-sm font-medium text-ink-700 transition-colors hover:bg-ink-50 disabled:cursor-not-allowed disabled:opacity-50"
      >
        <Download className="h-4 w-4" />
        {label}
        <ChevronDown className={cn('h-4 w-4 text-ink-400 transition-transform', open && 'rotate-180')} />
      </button>
      {open && (
        <div className="absolute right-0 top-full z-20 mt-2 w-44 overflow-hidden rounded-xl border border-ink-100 bg-white py-1.5 shadow-card-hover animate-fade-in">
          <button
            type="button"
            onClick={() => pick(onCsv)}
            className="flex w-full items-center gap-2.5 px-4 py-2.5 text-sm text-ink-600 hover:bg-ink-50"
          >
            <FileSpreadsheet className="h-4 w-4 text-emerald-600" />
            Export CSV
          </button>
          <button
            type="button"
            onClick={() => pick(onPdf)}
            className="flex w-full items-center gap-2.5 px-4 py-2.5 text-sm text-ink-600 hover:bg-ink-50"
          >
            <FileText className="h-4 w-4 text-red-600" />
            Export PDF
          </button>
        </div>
      )}
    </div>
  )
}
