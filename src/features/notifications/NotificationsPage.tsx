import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle, Bell, CalendarClock, CheckCircle2, Search } from 'lucide-react'
import {
  Card,
  EmptyState,
  ExportMenu,
  Input,
  PageHeader,
  TableSkeleton,
  useToast,
} from '@/components/ui'
import { useNotifications, type AppNotification, type NotificationType } from './NotificationsContext'
import { cn } from '@/lib/cn'
import { daysUntil } from '@/lib/date'
import { formatDate, formatDateTime } from '@/lib/format'
import { makeExportHandlers } from '@/lib/exporters'

const STYLE: Record<NotificationType, { icon: typeof Bell; wrap: string; fg: string }> = {
  overdue: { icon: AlertTriangle, wrap: 'bg-red-50', fg: 'text-red-600' },
  'due-soon': { icon: CalendarClock, wrap: 'bg-amber-50', fg: 'text-amber-600' },
  payment: { icon: CheckCircle2, wrap: 'bg-emerald-50', fg: 'text-emerald-600' },
}

const TYPE_LABEL: Record<NotificationType, string> = {
  overdue: 'Terlambat',
  'due-soon': 'Jatuh Tempo',
  payment: 'Pembayaran',
}

type TypeFilter = NotificationType | 'all'

const TYPE_TABS: { value: TypeFilter; label: string; dot?: string }[] = [
  { value: 'all', label: 'Semua' },
  { value: 'due-soon', label: 'Jatuh Tempo', dot: 'bg-amber-400' },
  { value: 'overdue', label: 'Terlambat', dot: 'bg-red-500' },
  { value: 'payment', label: 'Pembayaran', dot: 'bg-emerald-500' },
]

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
  const { toast } = useToast()
  // Freeze which items were unread on entry, then mark everything read so the
  // bell clears — the highlight stays for this visit only.
  const [highlight, setHighlight] = useState<Set<string> | null>(null)
  const [type, setType] = useState<TypeFilter>('all')
  const [search, setSearch] = useState('')

  useEffect(() => {
    if (loading || highlight) return
    setHighlight(new Set(items.filter((it) => isUnread(it.id)).map((it) => it.id)))
    markAllRead()
  }, [loading, items, isUnread, markAllRead, highlight])

  const countByType = useMemo(
    () =>
      items.reduce<Record<string, number>>((acc, n) => {
        acc[n.type] = (acc[n.type] ?? 0) + 1
        return acc
      }, {}),
    [items],
  )

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return items.filter((n) => {
      if (type !== 'all' && n.type !== type) return false
      if (q && !n.title.toLowerCase().includes(q) && !n.description.toLowerCase().includes(q))
        return false
      return true
    })
  }, [items, type, search])

  // Exports reflect the active filters.
  const exportHandlers = makeExportHandlers({
    filename: 'notifikasi',
    title: 'Daftar Notifikasi',
    subtitle: `Jenis: ${TYPE_TABS.find((t) => t.value === type)?.label ?? 'Semua'}`,
    meta: [`${filtered.length} notifikasi`, `Dibuat ${formatDateTime(new Date())}`],
    columns: [
      { label: 'Jenis' },
      { label: 'Judul' },
      { label: 'Keterangan' },
      { label: 'Tanggal' },
      { label: 'Status' },
    ],
    rows: filtered.map((n) => [
      TYPE_LABEL[n.type],
      n.title,
      n.description,
      formatDate(n.time),
      metaLabel(n),
    ]),
    onPopupBlocked: () => toast('Izinkan popup di browser untuk mengekspor PDF.', 'error'),
  })

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="Notifikasi"
        description="Tagihan, jatuh tempo, dan pembayaran terbaru."
        action={<ExportMenu {...exportHandlers} disabled={filtered.length === 0} />}
      />

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
          {/* Filter: jenis + pencarian */}
          <div className="flex gap-1 overflow-x-auto border-b border-ink-100 px-4 pt-3 pb-0">
            {TYPE_TABS.map((tab) => {
              const count = tab.value === 'all' ? items.length : (countByType[tab.value] ?? 0)
              return (
                <button
                  key={tab.value}
                  onClick={() => setType(tab.value)}
                  className={cn(
                    'flex shrink-0 items-center gap-1.5 rounded-t-lg border-b-2 px-3 pb-2.5 pt-2 text-[13px] font-medium transition-colors',
                    type === tab.value
                      ? 'border-brand-500 text-brand-600'
                      : 'border-transparent text-ink-500 hover:text-ink-800',
                  )}
                >
                  {tab.dot && <span className={cn('h-2 w-2 rounded-full', tab.dot)} />}
                  {tab.label}
                  {count > 0 && (
                    <span
                      className={cn(
                        'rounded-full px-1.5 py-0.5 text-[11px] font-semibold leading-none',
                        type === tab.value ? 'bg-brand-100 text-brand-600' : 'bg-ink-100 text-ink-500',
                      )}
                    >
                      {count}
                    </span>
                  )}
                </button>
              )
            })}
          </div>
          <div className="border-b border-ink-100 p-4">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Cari notifikasi…"
                className="pl-10"
              />
            </div>
          </div>

          {filtered.length === 0 ? (
            <EmptyState
              icon={Bell}
              title="Tidak ada hasil"
              description="Tidak ada notifikasi yang cocok dengan filter saat ini."
            />
          ) : (
            filtered.map((n) => {
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
            })
          )}
        </Card>
      )}
    </div>
  )
}
