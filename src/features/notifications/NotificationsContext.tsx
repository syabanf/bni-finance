import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import type { InvoiceWithRelations, PaymentWithInvoice } from '@/types'
import { invoiceService, paymentService } from '@/services'
import { daysUntil } from '@/lib/date'
import { formatCurrency } from '@/lib/format'

export type NotificationType = 'overdue' | 'due-soon' | 'payment'

export interface AppNotification {
  id: string
  type: NotificationType
  title: string
  description: string
  /** Real event date (ISO) — used for relative display. */
  time: string
  link: string
}

interface NotificationsValue {
  items: AppNotification[]
  unreadCount: number
  loading: boolean
  isUnread: (id: string) => boolean
  markAllRead: () => void
  reload: () => void
}

const SEEN_KEY = 'bni:notif:seen'
const PAYMENT_LIMIT = 30

function loadSeen(): Set<string> {
  try {
    const raw = localStorage.getItem(SEEN_KEY)
    return new Set(raw ? (JSON.parse(raw) as string[]) : [])
  } catch {
    return new Set()
  }
}

const NotificationsContext = createContext<NotificationsValue | null>(null)

export function NotificationsProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<AppNotification[]>([])
  const [loading, setLoading] = useState(true)
  const [seen, setSeen] = useState<Set<string>>(() => loadSeen())

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [invoices, payments] = await Promise.all([
        invoiceService.list(),
        paymentService.list(),
      ])
      setItems(buildNotifications(invoices, payments))
    } catch {
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const isUnread = useCallback((id: string) => !seen.has(id), [seen])

  const unreadCount = useMemo(
    () => items.reduce((n, it) => (seen.has(it.id) ? n : n + 1), 0),
    [items, seen],
  )

  const markAllRead = useCallback(() => {
    setItems((current) => {
      const next = loadSeen()
      current.forEach((it) => next.add(it.id))
      localStorage.setItem(SEEN_KEY, JSON.stringify([...next]))
      setSeen(next)
      return current
    })
  }, [])

  const value = useMemo(
    () => ({ items, unreadCount, loading, isUnread, markAllRead, reload: load }),
    [items, unreadCount, loading, isUnread, markAllRead, load],
  )

  return <NotificationsContext.Provider value={value}>{children}</NotificationsContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useNotifications(): NotificationsValue {
  const ctx = useContext(NotificationsContext)
  if (!ctx) throw new Error('useNotifications must be used within NotificationsProvider')
  return ctx
}

// Order: upcoming dues (soonest first) → overdue (most recently due first) →
// payments (most recent first). Keeps the actionable items on top.
const RANK: Record<NotificationType, number> = { 'due-soon': 0, overdue: 1, payment: 2 }

function buildNotifications(
  invoices: InvoiceWithRelations[],
  payments: PaymentWithInvoice[],
): AppNotification[] {
  const out: AppNotification[] = []

  for (const inv of invoices) {
    const member = inv.member?.name ?? 'Member'
    if (inv.status === 'overdue') {
      out.push({
        id: `overdue:${inv.id}`,
        type: 'overdue',
        title: 'Tagihan terlambat',
        description: `${member} · ${inv.number} belum membayar ${formatCurrency(inv.amount)}.`,
        time: inv.dueDate,
        link: `/invoices/${inv.id}`,
      })
    } else if (inv.status === 'sent') {
      const d = daysUntil(inv.dueDate)
      if (d >= 0 && d <= 7) {
        out.push({
          id: `due-soon:${inv.id}`,
          type: 'due-soon',
          title: 'Jatuh tempo mendekat',
          description: `${member} · ${inv.number} jatuh tempo ${d === 0 ? 'hari ini' : `dalam ${d} hari`}.`,
          time: inv.dueDate,
          link: `/invoices/${inv.id}`,
        })
      }
    }
  }

  const recentPayments = [...payments]
    .sort((a, b) => (a.paidAt < b.paidAt ? 1 : a.paidAt > b.paidAt ? -1 : 0))
    .slice(0, PAYMENT_LIMIT)
  for (const p of recentPayments) {
    const member = p.member?.name ?? 'Member'
    const num = p.invoice?.number
    out.push({
      id: `payment:${p.id}`,
      type: 'payment',
      title: 'Pembayaran diterima',
      description: `${member} membayar ${formatCurrency(p.amount)}${num ? ` · ${num}` : ''}.`,
      time: p.paidAt,
      link: p.invoiceId ? `/invoices/${p.invoiceId}` : '/payments',
    })
  }

  return out.sort((a, b) => {
    if (RANK[a.type] !== RANK[b.type]) return RANK[a.type] - RANK[b.type]
    // due-soon: soonest first; overdue/payment: most recent first.
    if (a.type === 'due-soon') return a.time < b.time ? -1 : a.time > b.time ? 1 : 0
    return a.time < b.time ? 1 : a.time > b.time ? -1 : 0
  })
}
