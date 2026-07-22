import { AlertTriangle } from 'lucide-react'
import { Button } from './Button'

interface ErrorStateProps {
  message?: string | null
  onRetry?: () => void
  className?: string
}

/** Shown when a data fetch fails — distinct from the empty state so a backend
 * error never reads as "no data". */
export function ErrorState({ message, onRetry, className }: ErrorStateProps) {
  return (
    <div className={`flex flex-col items-center justify-center px-6 py-16 text-center ${className ?? ''}`}>
      <span className="flex h-12 w-12 items-center justify-center rounded-full bg-red-50 text-red-500">
        <AlertTriangle className="h-6 w-6" />
      </span>
      <h3 className="mt-4 text-[15px] font-semibold text-ink-900">Gagal memuat data</h3>
      <p className="mt-1 max-w-sm text-sm text-ink-500">
        {message || 'Terjadi kesalahan saat memuat data. Coba lagi.'}
      </p>
      {onRetry && (
        <Button variant="outline" className="mt-4" onClick={onRetry}>
          Coba lagi
        </Button>
      )}
    </div>
  )
}
