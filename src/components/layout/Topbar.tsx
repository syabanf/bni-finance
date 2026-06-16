import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Bell, ChevronDown, LogOut, Search, UserCircle2 } from 'lucide-react'
import { Avatar, BniLogo } from '@/components/ui'
import { useAuth } from '@/features/auth/AuthContext'
import { cn } from '@/lib/cn'

export function Topbar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <header className="sticky top-0 z-30 border-b border-ink-100 bg-white/90 backdrop-blur-md safe-top">
      <div className="flex h-16 items-center gap-3 px-4 lg:px-6">
        {/* Mobile brand (sidebar logo is hidden on mobile) */}
        <Link to="/dashboard" className="flex items-center gap-2.5 lg:hidden">
          <BniLogo className="h-7 w-auto" />
          <span className="border-l border-ink-100 pl-2.5 text-[15px] font-bold text-ink-900">Finance Hub</span>
        </Link>

        {/* Search */}
        <div className="relative ml-auto hidden w-full max-w-xs sm:block">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
          <input
            type="search"
            placeholder="Cari invoice, member…"
            className="h-9 w-full rounded-xl border border-ink-200 bg-ink-50 pl-9 pr-3 text-sm text-ink-700 placeholder:text-ink-400 transition-colors focus-ring focus:border-brand-400 focus:bg-white"
          />
        </div>

        {/* Notifications */}
        <button
          className="relative ml-auto rounded-xl p-2 text-ink-500 transition-colors hover:bg-ink-100 sm:ml-0"
          aria-label="Notifikasi"
        >
          <Bell className="h-5 w-5" />
          <span className="absolute right-2 top-2 h-2 w-2 rounded-full bg-brand-500 ring-2 ring-white" />
        </button>

        {/* User menu */}
        <div className="relative" ref={menuRef}>
          <button
            onClick={() => setMenuOpen((o) => !o)}
            className="flex items-center gap-2.5 rounded-xl py-1.5 pl-2 pr-2.5 transition-colors hover:bg-ink-100"
          >
            <div className="hidden text-right leading-tight sm:block">
              <div className="text-sm font-semibold text-ink-900">{user?.name ?? 'Admin'}</div>
              <div className="text-xs text-ink-400">National Admin</div>
            </div>
            <Avatar name={user?.name ?? 'Admin'} size="sm" />
            <ChevronDown className={cn('h-4 w-4 text-ink-400 transition-transform', menuOpen && 'rotate-180')} />
          </button>

          {menuOpen && (
            <div className="absolute right-0 top-full mt-2 w-56 overflow-hidden rounded-xl border border-ink-100 bg-white py-1.5 shadow-card-hover animate-fade-in">
              <div className="border-b border-ink-100 px-4 py-3">
                <div className="text-sm font-semibold text-ink-900">{user?.name}</div>
                <div className="truncate text-xs text-ink-400">{user?.email}</div>
              </div>
              <button className="flex w-full items-center gap-2.5 px-4 py-2.5 text-sm text-ink-600 hover:bg-ink-50">
                <UserCircle2 className="h-4 w-4" />
                Profil Saya
              </button>
              <button
                onClick={handleLogout}
                className="flex w-full items-center gap-2.5 px-4 py-2.5 text-sm text-red-600 hover:bg-red-50"
              >
                <LogOut className="h-4 w-4" />
                Keluar
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
