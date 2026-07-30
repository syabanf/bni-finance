import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { CheckCircle2, AlertCircle, Info, X } from 'lucide-react'
import { cn } from '@/lib/cn'

type ToastTone = 'success' | 'error' | 'info'
interface Toast {
  id: number
  tone: ToastTone
  message: string
}

interface ToastContextValue {
  toast: (message: string, tone?: ToastTone) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

const config: Record<ToastTone, { icon: typeof Info; bar: string; iconColor: string }> = {
  success: { icon: CheckCircle2, bar: 'bg-emerald-500', iconColor: 'text-emerald-500' },
  error: { icon: AlertCircle, bar: 'bg-red-500', iconColor: 'text-red-500' },
  info: { icon: Info, bar: 'bg-blue-500', iconColor: 'text-blue-500' },
}

let counter = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const remove = useCallback((id: number) => {
    setToasts((t) => t.filter((x) => x.id !== id))
  }, [])

  const toast = useCallback(
    (message: string, tone: ToastTone = 'success') => {
      const id = ++counter
      setToasts((t) => [...t, { id, tone, message }])
      setTimeout(() => remove(id), 3800)
    },
    [remove],
  )

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      {createPortal(
        // `w-full max-w-sm` sendirian meleset di ponsel: max-w-sm itu 384px,
        // lebih lebar dari layar 375px, jadi toast menonjol keluar tepi kanan.
        // Di layar sempit toast dikurung oleh inset kiri+kanan; mulai sm ia
        // kembali menempel di kanan dengan lebar maksimum semula.
        <div className="pointer-events-none fixed bottom-4 left-3 right-3 z-[60] flex flex-col gap-2.5 sm:bottom-5 sm:left-auto sm:right-5 sm:w-full sm:max-w-sm">
          {toasts.map((t) => {
            const c = config[t.tone]
            const Icon = c.icon
            return (
              <div
                key={t.id}
                className="pointer-events-auto flex items-stretch overflow-hidden rounded-xl border border-ink-100 bg-white shadow-card-hover animate-fade-in"
              >
                <div className={cn('w-1', c.bar)} />
                <div className="flex flex-1 items-center gap-3 px-4 py-3">
                  <Icon className={cn('h-5 w-5 flex-shrink-0', c.iconColor)} />
                  <p className="flex-1 text-sm text-ink-700">{t.message}</p>
                  <button
                    onClick={() => remove(t.id)}
                    className="rounded-md p-0.5 text-ink-400 hover:bg-ink-100 hover:text-ink-700"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>
            )
          })}
        </div>,
        document.body,
      )}
    </ToastContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}
