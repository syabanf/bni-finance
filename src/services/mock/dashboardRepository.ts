import type { DashboardRepository } from '@/services/types'
import type { DashboardSummary, Invoice, InvoiceStatus } from '@/types'
import { daysUntil, monthKey } from '@/lib/date'
import { delay, store } from './store'

const ID_MONTHS_SHORT = [
  'Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun',
  'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des',
]

function sum(invoices: Invoice[]): number {
  return invoices.reduce((acc, i) => acc + i.amount, 0)
}

/** % change, clamped/ rounded, safe against divide-by-zero. */
function pct(current: number, previous: number): number {
  if (previous === 0) return current > 0 ? 100 : 0
  return Math.round(((current - previous) / previous) * 100)
}

export const mockDashboardRepository: DashboardRepository = {
  async summary() {
    const now = new Date()
    const thisMonth = monthKey(now.toISOString())
    const prev = new Date(now.getFullYear(), now.getMonth() - 1, 1)
    const prevMonth = monthKey(prev.toISOString())

    const all = store.invoices
    const active = all.filter((i) => i.status !== 'cancelled')
    const issuedThis = active.filter((i) => monthKey(i.dueDate) === thisMonth)
    const issuedPrev = active.filter((i) => monthKey(i.dueDate) === prevMonth)

    const paidAll = all.filter((i) => i.status === 'paid')
    const paidThis = paidAll.filter((i) => i.paidAt && monthKey(i.paidAt) === thisMonth)
    const paidPrev = paidAll.filter((i) => i.paidAt && monthKey(i.paidAt) === prevMonth)

    // Outstanding = issued but not paid (sent/overdue). Snapshot, not month-bound.
    const outstanding = all.filter((i) => i.status === 'sent' || i.status === 'overdue')
    const overdue = all.filter((i) => i.status === 'overdue')
    const sentThis = active.filter((i) => i.status === 'sent' && monthKey(i.dueDate) === thisMonth)

    // Renewal due within 30 days (latest active period per member ending soon).
    const latestByMember = new Map<string, Invoice>()
    for (const inv of all) {
      if (inv.status === 'cancelled') continue
      const existing = latestByMember.get(inv.memberId)
      if (!existing || inv.periodEnd > existing.periodEnd) latestByMember.set(inv.memberId, inv)
    }
    const renewalDue = [...latestByMember.values()].filter((inv) => {
      const d = daysUntil(inv.periodEnd)
      if (d > 30) return false
      const hasFuture = all.some(
        (i) => i.memberId === inv.memberId && i.status !== 'cancelled' && i.periodStart > inv.periodStart,
      )
      return !hasFuture
    })

    // Status breakdown for the donut (cancelled isn't a payment status, so it's
    // excluded — this keeps the donut total aligned with the "Total Invoice" card).
    const statuses: InvoiceStatus[] = ['paid', 'sent', 'overdue', 'draft']
    const statusBreakdown = statuses
      .map((status) => ({ status, count: all.filter((i) => i.status === status).length }))
      .filter((s) => s.count > 0)

    // Last 6 months issued vs. paid amounts.
    const monthly: DashboardSummary['monthly'] = []
    for (let k = 5; k >= 0; k--) {
      const d = new Date(now.getFullYear(), now.getMonth() - k, 1)
      const key = monthKey(d.toISOString())
      const label = ID_MONTHS_SHORT[d.getMonth()]
      const issued = sum(all.filter((i) => monthKey(i.dueDate) === key && i.status !== 'cancelled'))
      const paid = sum(all.filter((i) => i.status === 'paid' && i.paidAt && monthKey(i.paidAt) === key))
      monthly.push({ month: label, issued, paid })
    }

    const summary: DashboardSummary = {
      total: {
        count: active.length,
        amount: sum(active),
        trend: pct(issuedThis.length, issuedPrev.length), // issued momentum MoM
      },
      paid: {
        count: paidAll.length,
        amount: sum(paidAll),
        trend: pct(paidThis.length, paidPrev.length), // payments MoM
      },
      outstanding: {
        count: outstanding.length,
        amount: sum(outstanding),
        trend: sentThis.length, // new outstanding this month (absolute)
      },
      overdue: {
        count: overdue.length,
        amount: sum(overdue),
        trend: overdue.length, // shown as absolute (e.g. +3)
      },
      renewalDue: {
        count: renewalDue.length,
        trend: 0,
      },
      statusBreakdown,
      monthly,
    }

    return delay(summary, 350)
  },
}
