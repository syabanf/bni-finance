import { Link, NavLink, useLocation } from 'react-router-dom'
import { FileText, LayoutGrid, MoreHorizontal, Plus, Users } from 'lucide-react'
import { cn } from '@/lib/cn'

const tab = (active: boolean) =>
  cn(
    'flex flex-1 flex-col items-center justify-center gap-1 py-2.5 text-[11px] font-medium transition-colors',
    active ? 'text-brand-500' : 'text-ink-400 hover:text-ink-600',
  )

/**
 * Native-style bottom tab bar shown on mobile / installed PWA. Surfaces the
 * most-used destinations plus a center "Buat Invoice" action; everything else
 * lives behind "Lainnya" (the full nav drawer).
 */
export function BottomNav({ onMore }: { onMore: () => void }) {
  const { pathname } = useLocation()
  const moreActive =
    pathname !== '/dashboard' &&
    !pathname.startsWith('/invoices') &&
    !pathname.startsWith('/members')

  return (
    <nav className="fixed inset-x-0 bottom-0 z-40 border-t border-ink-100 bg-white/95 backdrop-blur-md pb-safe lg:hidden">
      <div className="flex items-stretch px-1">
        <NavLink to="/dashboard" className={({ isActive }) => tab(isActive)}>
          {({ isActive }) => (
            <>
              <LayoutGrid className="h-5 w-5" strokeWidth={isActive ? 2.4 : 2} />
              Dashboard
            </>
          )}
        </NavLink>

        <NavLink to="/invoices" className={({ isActive }) => tab(isActive)}>
          {({ isActive }) => (
            <>
              <FileText className="h-5 w-5" strokeWidth={isActive ? 2.4 : 2} />
              Invoice
            </>
          )}
        </NavLink>

        {/* Center action */}
        <div className="flex flex-1 items-start justify-center">
          <Link
            to="/invoices/new"
            aria-label="Buat Invoice"
            className="-mt-5 flex h-14 w-14 items-center justify-center rounded-full bg-brand-500 text-white shadow-lg shadow-brand-500/40 ring-4 ring-white transition-transform active:scale-95"
          >
            <Plus className="h-6 w-6" strokeWidth={2.5} />
          </Link>
        </div>

        <NavLink to="/members" className={({ isActive }) => tab(isActive)}>
          {({ isActive }) => (
            <>
              <Users className="h-5 w-5" strokeWidth={isActive ? 2.4 : 2} />
              Member
            </>
          )}
        </NavLink>

        <button type="button" onClick={onMore} className={tab(moreActive)}>
          <MoreHorizontal className="h-5 w-5" strokeWidth={moreActive ? 2.4 : 2} />
          Lainnya
        </button>
      </div>
    </nav>
  )
}
