import { useState } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { cn } from '@/lib/cn'
import { useAuth } from '@/features/auth/AuthContext'
import { Sidebar } from './Sidebar'
import { Topbar } from './Topbar'

export function AppLayout() {
  const { user } = useAuth()
  const location = useLocation()
  const [mobileOpen, setMobileOpen] = useState(false)

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return (
    <div className="min-h-screen bg-ink-50">
      {/* Desktop sidebar */}
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-[264px] border-r border-ink-100 lg:block">
        <Sidebar />
      </aside>

      {/* Mobile sidebar drawer */}
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
            'absolute inset-y-0 left-0 w-[280px] border-r border-ink-100 shadow-xl transition-transform duration-200',
            mobileOpen ? 'translate-x-0' : '-translate-x-full',
          )}
        >
          <Sidebar onNavigate={() => setMobileOpen(false)} />
        </aside>
      </div>

      {/* Main column */}
      <div className="flex min-h-screen flex-col lg:pl-[264px]">
        <Topbar onMenuClick={() => setMobileOpen(true)} />
        <main className="flex-1 px-4 py-6 lg:px-8 lg:py-8">
          <div className="mx-auto max-w-[1400px] animate-fade-in">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
