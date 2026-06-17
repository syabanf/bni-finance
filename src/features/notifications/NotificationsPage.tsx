import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle, Bell, CalendarClock, CheckCircle2 } from 'lucide-react'
import { Card, EmptyState, PageHeader, TableSkeleton } from '@/components/ui'
import { useNotifications, type AppNotification, type NotificationType } from './NotificationsContext'
import { cn } from '@/lib/cn'
import { daysUntil } from '@/lib/date'
import { formatDateTime } from '@/lib/format'

const STYLE: Record<NotificationType, { icon: typeof Bell; wrap: string; fg: string }> = {
  overdue: { icon: AlertTriangle, wrap: 'bg-red-50', fg: 'text-red-600' },
  'due-soon': { icon: CalendarClock, wrap: 'bg-amber-50', fg: 'text-amber-600' },
  payment: { icon: CheckCircle2, wrap: 'bg-emerald-50', fg: 'text-emerald-600' },
}

function metaLabel(n: AppNotification): string {
  if (n.type === 'due-soon') {
    const d = daysUntil(n.time)
    return d <= 0 ? 'Hari ini' : `${d} hari lagi`
  }
  if (n.type === 'overdue') {
    const d = daysUntil(n.time)
    return d === 0 ? 'Jatuh tempo hari ini' : `Telat ${Math.abs(d)} hari`
  }
  return formatDateTime(n.time)
}

export function NotificationsPage() {
  const { items, loading, isUnread, markAllRead } = useNotifications()
  // Freeze which items were unread on entry, then mark everything read so the
  // bell clears — the highlight stays for this visit only.
  const [highlight, setHighlight] = useState<Set<string> | null>(null)

  useEffect(() => {
    if (loading || highlight) return
    setHighlight(new Set(items.filter((it) => isUnread(it.id)).map((it) => it.id)))
    markAllRead()
  }, [loading, items, isUnread, markAllRead, highlight])

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader title="Notifikasi" description="Tagihan, jatuh tempo, dan pembayaran terbaru." />

      {loading ? (
        <Card>
          <TableSkeleton rows={6} cols={1} />
        </Card>
      ) : items.length === 0 ? (
        <Card>
          <EmptyState
            icon={Bell}
            title="Belum ada notifikasi"
            description="Tagihan jatuh tempo dan pembayaran terbaru akan muncul di sini."
          />
        </Card>
      ) : (
        <Card className="overflow-hidden">
          {items.map((n) => {
            const s = STYLE[n.type]
            const Icon = s.icon
            const unread = highlight?.has(n.id)
            return (
              <Link
                key={n.id}
                to={n.link}
                className={cn(
                  'flex items-start gap-3 border-b border-ink-100 px-5 py-4 transition-colors last:border-0 hover:bg-ink-50',
                  unread && 'bg-brand-50/40',
                )}
              >
                <span className={cn('mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full', s.wrap)}>
                  <Icon className={cn('h-4 w-4', s.fg)} />
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold text-ink-900">{n.title}</span>
                    {unread && <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-brand-500" />}
                  </div>
                  <p className="mt-0.5 text-sm text-ink-500">{n.description}</p>
                </div>
                <span className="shrink-0 whitespace-nowrap pt-0.5 text-xs text-ink-400">{metaLabel(n)}</span>
              </Link>
            )
          })}
        </Card>
      )}
    </div>
  )
}
