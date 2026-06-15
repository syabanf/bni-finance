import { useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/cn'
import { Avatar } from '@/components/ui'
import { useAuth } from '@/features/auth/AuthContext'
import { useUrgentCount } from '@/hooks/useUrgentCount'
import { NAV, type NavLeaf } from './nav'

const itemBase =
  'flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors'

function leafIsActive(pathname: string, leaf: NavLeaf) {
  return leaf.end ? pathname === leaf.to : pathname.startsWith(leaf.to)
}

function NavGroup({ label, icon: Icon, children }: Extract<(typeof NAV)[number], { kind: 'group' }>) {
  const { pathname } = useLocation()
  const groupActive = children.some((c) => leafIsActive(pathname, c))
  const [open, setOpen] = useState(groupActive)

  return (
    <div>
      <button
        onClick={() => setOpen((o) => !o)}
        className={cn(itemBase, 'w-full justify-between text-ink-600 hover:bg-ink-100 hover:text-ink-900')}
      >
        <span className="flex items-center gap-3">
          <Icon className="h-[18px] w-[18px]" />
          {label}
        </span>
        <ChevronDown className={cn('h-4 w-4 text-ink-400 transition-transform', open && 'rotate-180')} />
      </button>
      {open && (
        <div className="mt-1 space-y-0.5 pl-[34px]">
          {children.map((leaf) => (
            <NavLink
              key={leaf.to}
              to={leaf.to}
              end={leaf.end}
              className={({ isActive }) =>
                cn(
                  'block rounded-lg px-3 py-2 text-[13px] font-medium transition-colors',
                  isActive
                    ? 'bg-brand-50 text-brand-600'
                    : 'text-ink-500 hover:bg-ink-100 hover:text-ink-800',
                )
              }
            >
              {leaf.label}
            </NavLink>
          ))}
        </div>
      )}
    </div>
  )
}

export function Sidebar({ onNavigate }: { onNavigate?: () => void }) {
  const { user } = useAuth()
  const urgentCount = useUrgentCount()
  return (
    <div className="flex h-full flex-col bg-white">
      {/* Brand */}
      <div className="flex items-center gap-3 px-5 py-5">
        <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-brand-500 to-brand-600 text-lg font-extrabold text-white shadow-sm">
          B
        </div>
        <div className="leading-tight">
          <div className="text-[10px] font-bold uppercase tracking-wider text-brand-500">
            BNI Indonesia
          </div>
          <div className="text-[15px] font-bold text-ink-900">Finance Hub</div>
          <div className="text-[11px] text-ink-400">Payment Platform</div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 space-y-1 overflow-y-auto px-3 pb-4" onClick={onNavigate}>
        {NAV.map((node, i) => {
          if (node.kind === 'section') {
            return (
              <div
                key={`s-${i}`}
                className="px-3 pb-1.5 pt-5 text-[11px] font-semibold uppercase tracking-wider text-ink-400"
              >
                {node.label}
              </div>
            )
          }
          if (node.kind === 'group') {
            return <NavGroup key={node.label} {...node} />
          }
          const Icon = node.icon
          if (node.urgent) {
            return (
              <NavLink
                key={node.to}
                to={node.to}
                end={node.end}
                className={({ isActive }) =>
                  cn(
                    itemBase,
                    isActive
                      ? 'bg-red-500 text-white shadow-sm shadow-red-500/30'
                      : 'bg-red-50 text-red-600 hover:bg-red-100',
                  )
                }
              >
                <Icon className="h-[18px] w-[18px]" />
                <span className="flex-1">{node.label}</span>
                {urgentCount.total > 0 && (
                  <span className="flex h-5 min-w-[20px] items-center justify-center rounded-full bg-red-500 px-1.5 text-[11px] font-bold text-white">
                    {urgentCount.total}
                  </span>
                )}
              </NavLink>
            )
          }
          return (
            <NavLink
              key={node.to}
              to={node.to}
              end={node.end}
              className={({ isActive }) =>
                cn(
                  itemBase,
                  isActive
                    ? 'bg-brand-500 text-white shadow-sm shadow-brand-500/30'
                    : 'text-ink-600 hover:bg-ink-100 hover:text-ink-900',
                )
              }
            >
              <Icon className="h-[18px] w-[18px]" />
              {node.label}
            </NavLink>
          )
        })}
      </nav>

      {/* User footer */}
      <div className="border-t border-ink-100 p-3">
        <div className="flex items-center gap-3 rounded-xl px-2 py-2">
          <Avatar name={user?.name ?? 'Admin'} size="sm" />
          <div className="min-w-0 leading-tight">
            <div className="truncate text-sm font-semibold text-ink-900">
              {user?.name ?? 'Admin Nasional'}
            </div>
            <div className="truncate text-xs text-ink-400">National Admin</div>
          </div>
        </div>
      </div>
    </div>
  )
}
