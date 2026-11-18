import { useState } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { cn } from '@/lib/cn'
import { useAuth } from '@/features/auth/AuthContext'
import { NotificationsProvider } from '@/features/notifications/NotificationsContext'
import { Sidebar } from './Sidebar'
import { Topbar } from './Topbar'
import { BottomNav } from './BottomNav'

export function AppLayout() {
  const { user, loading } = useAuth()
  const location = useLocation()
  const [mobileOpen, setMobileOpen] = useState(false)

  if (loading) return null

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return (
    <NotificationsProvider>
      <div className="relative isolate min-h-screen">
      {/* Latar diam di tempat saat halaman bergulir. Elemen fixed tersendiri,
          bukan background-attachment: fixed, yang diabaikan Safari iOS. */}
      <div className="kanvas-kaca pointer-events-none fixed inset-0 -z-10" aria-hidden />
      {/* Desktop sidebar */}
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-[264px] lg:block">
        <Sidebar />
      </aside>

      {/* Menu untuk layar di bawah lg. Tiga bentuk menurut lebar layar:
          HP (di bawah md): lembar yang naik dari bawah, dekat dengan jempol
          dan tombol "Lainnya" yang membukanya. Tablet (md sampai lg): laci
          dari kiri, karena layarnya cukup lebar untuk menu di samping isi.
          Laptop (lg ke atas): sidebar tetap di atas, blok ini tidak dipakai. */}
      <div
        className={cn(
          'fixed inset-0 z-50 lg:hidden',
          mobileOpen ? 'pointer-events-auto' : 'pointer-events-none',
        )}
      >
        <div
          className={cn(
            'absolute inset-0 bg-ink-900/30 backdrop-blur-sm transition-opacity',
            mobileOpen ? 'opacity-100' : 'opacity-0',
          )}
          onClick={() => setMobileOpen(false)}
        />
        <aside
          className={cn(
            'absolute overflow-hidden shadow-2xl transition-transform duration-300 ease-out safe-bottom',
            'inset-x-0 bottom-0 h-[85dvh] rounded-t-[28px]',
            'md:inset-y-0 md:left-0 md:right-auto md:h-auto md:w-[320px] md:rounded-none',
            mobileOpen
              ? 'translate-y-0 md:translate-x-0'
              : 'translate-y-full md:translate-y-0 md:-translate-x-full',
          )}
        >
          {/* Pegangan lembar, hanya di HP. */}
          <span
            className="absolute left-1/2 top-2 z-10 h-1.5 w-10 -translate-x-1/2 rounded-full bg-ink-300/80 md:hidden"
            aria-hidden
          />
          <Sidebar onNavigate={() => setMobileOpen(false)} />
        </aside>
      </div>

      {/* Main column */}
      <div className="flex min-h-screen flex-col lg:pl-[264px]">
        <Topbar />
        <main className="flex-1 px-4 py-6 pb-28 lg:px-8 lg:py-8 lg:pb-8">
          <div className="mx-auto max-w-[1400px] animate-fade-in">
            <Outlet />
          </div>
        </main>
      </div>

      {/* Mobile bottom tab bar */}
      <BottomNav onMore={() => setMobileOpen(true)} />
      </div>
    </NotificationsProvider>
  )
}
