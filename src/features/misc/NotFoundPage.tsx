import { Link } from 'react-router-dom'
import { Button } from '@/components/ui'

export function NotFoundPage() {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center text-center">
      <div className="text-7xl font-extrabold tracking-tight text-brand-500">404</div>
      <h1 className="mt-3 text-xl font-bold text-ink-900">Halaman tidak ditemukan</h1>
      <p className="mt-1.5 max-w-sm text-sm text-ink-500">
        Halaman yang Anda cari mungkin telah dipindahkan atau tidak tersedia.
      </p>
      <Link to="/dashboard" className="mt-6">
        <Button>Kembali ke Dashboard</Button>
      </Link>
    </div>
  )
}
