import { useRouteError } from 'react-router-dom'
import { AlertTriangle, RefreshCw, RotateCcw } from 'lucide-react'
import { Button, Card, CardBody } from '@/components/ui'

const isDev = import.meta.env.DEV

/**
 * Route-level error boundary. Without this, any render error shows React
 * Router's raw "Unexpected Application Error!" screen with a stack trace.
 *
 * The "muat ulang bersih" action also clears the PWA service worker + caches,
 * which is the recovery path when a stale precached bundle is the cause.
 */
export function RouteErrorPage() {
  const error = useRouteError()
  const message =
    error instanceof Error ? error.message : typeof error === 'string' ? error : 'Terjadi kesalahan.'

  const reload = () => window.location.reload()

  const hardReload = async () => {
    try {
      const regs = (await navigator.serviceWorker?.getRegistrations?.()) ?? []
      await Promise.all(regs.map((r) => r.unregister()))
      if ('caches' in window) {
        const keys = await caches.keys()
        await Promise.all(keys.map((k) => caches.delete(k)))
      }
    } catch {
      // best effort — reload anyway
    }
    window.location.reload()
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-ink-50 p-4">
      <Card className="w-full max-w-md">
        <CardBody className="flex flex-col items-center py-10 text-center">
          <span className="flex h-14 w-14 items-center justify-center rounded-full bg-red-50 text-red-500">
            <AlertTriangle className="h-7 w-7" />
          </span>
          <h1 className="mt-4 text-xl font-bold text-ink-900">Terjadi kesalahan</h1>
          <p className="mt-1.5 max-w-sm text-sm text-ink-500">
            Halaman gagal ditampilkan. Coba muat ulang — jika masih bermasalah, gunakan
            &ldquo;Muat ulang bersih&rdquo; untuk membersihkan cache aplikasi.
          </p>

          {isDev && (
            <pre className="mt-4 max-h-40 w-full overflow-auto rounded-lg bg-ink-50 p-3 text-left text-xs text-ink-600">
              {message}
            </pre>
          )}

          <div className="mt-6 flex flex-wrap items-center justify-center gap-2">
            <Button onClick={reload}>
              <RefreshCw className="h-4 w-4" />
              Muat ulang
            </Button>
            <Button variant="outline" onClick={hardReload}>
              <RotateCcw className="h-4 w-4" />
              Muat ulang bersih
            </Button>
          </div>
        </CardBody>
      </Card>
    </div>
  )
}
