import { supabase } from '@/lib/supabase'
import type { DashboardRepository } from '@/services/types'
import type { ChapterStat, DashboardSummary, InvoiceStatus } from '@/types'

export const supabaseDashboardRepository: DashboardRepository = {
  async summary(): Promise<DashboardSummary> {
    const today = new Date().toISOString().slice(0, 10)

    // Auto-mark overdue
    await supabase
      .from('invoices')
      .update({ status: 'overdue' })
      .eq('status', 'sent')
      .lt('due_date', today)

    const { data: invs, error } = await supabase
      .from('invoices')
      .select('id, status, amount, chapter_id, created_at, period_end, chapters(id, display_name)')
      .neq('status', 'cancelled')

    if (error) throw new Error(error.message)

    const rows = invs ?? []

    // Counts
    let totalCount = 0, totalAmount = 0
    let paidCount = 0, paidAmount = 0
    let outstandingCount = 0, outstandingAmount = 0
    let overdueCount = 0, overdueAmount = 0

    const statusBreakdownMap: Record<string, number> = {}
    const chapterMap: Record<string, ChapterStat> = {}

    const now = new Date()
    const thirtyDaysAgo = new Date(now); thirtyDaysAgo.setDate(now.getDate() - 30)

    // For trend: previous period (30–60 days ago)
    let prevTotal = 0, prevPaid = 0, prevOutstanding = 0, prevOverdue = 0

    for (const r of rows) {
      const row = r as Record<string, unknown>
      const status = row.status as InvoiceStatus
      const amount = row.amount as number
      const createdAt = new Date(row.created_at as string)
      const chId = row.chapter_id as string
      const ch = row.chapters as Record<string, unknown> | null

      totalCount++
      totalAmount += amount

      if (status === 'paid') { paidCount++; paidAmount += amount }
      else if (status === 'overdue') { overdueCount++; overdueAmount += amount }
      else if (status === 'sent') { outstandingCount++; outstandingAmount += amount }

      statusBreakdownMap[status] = (statusBreakdownMap[status] ?? 0) + 1

      // Chapter stats
      if (!chapterMap[chId]) {
        chapterMap[chId] = {
          chapterId: chId,
          chapterName: ch ? (ch.display_name as string) : chId,
          total: 0, paid: 0, outstanding: 0, overdue: 0, totalAmount: 0,
        }
      }
      chapterMap[chId].total++
      chapterMap[chId].totalAmount += amount
      if (status === 'paid') chapterMap[chId].paid++
      else if (status === 'overdue') chapterMap[chId].overdue++
      else if (status === 'sent') chapterMap[chId].outstanding++

      // Trend: 30–60 days ago
      const sixtyDaysAgo = new Date(now); sixtyDaysAgo.setDate(now.getDate() - 60)
      if (createdAt >= sixtyDaysAgo && createdAt < thirtyDaysAgo) {
        prevTotal++
        if (status === 'paid') prevPaid++
        else if (status === 'overdue') prevOverdue++
        else if (status === 'sent') prevOutstanding++
      }
    }

    // Renewal due in 30 days
    const renewalCutoff = new Date(now); renewalCutoff.setDate(now.getDate() + 30)
    const renewalCutoffStr = renewalCutoff.toISOString().slice(0, 10)
    const { count: renewalCount } = await supabase
      .from('invoices')
      .select('id', { count: 'exact', head: true })
      .eq('status', 'paid')
      .gte('period_end', today)
      .lte('period_end', renewalCutoffStr)

    // Monthly: last 6 months
    const monthly: DashboardSummary['monthly'] = []
    for (let i = 5; i >= 0; i--) {
      const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
      const monthStr = d.toISOString().slice(0, 7) // YYYY-MM
      const label = d.toLocaleDateString('id-ID', { month: 'short', year: '2-digit' })

      const monthRows = rows.filter((r: Record<string, unknown>) => {
        const row = r as Record<string, unknown>
        return (row.created_at as string).startsWith(monthStr)
      })
      const issued = monthRows.reduce((s: number, r: Record<string, unknown>) => s + (r.amount as number), 0)
      const paid = monthRows
        .filter((r: Record<string, unknown>) => r.status === 'paid')
        .reduce((s: number, r: Record<string, unknown>) => s + (r.amount as number), 0)
      monthly.push({ month: label, issued, paid })
    }

    const trend = (curr: number, prev: number) =>
      prev === 0 ? 0 : Math.round(((curr - prev) / prev) * 100)

    const chapterStats = Object.values(chapterMap)
      .sort((a, b) => b.overdue - a.overdue || b.outstanding - a.outstanding)

    return {
      total: { count: totalCount, amount: totalAmount, trend: trend(totalCount, prevTotal) },
      paid: { count: paidCount, amount: paidAmount, trend: trend(paidCount, prevPaid) },
      outstanding: { count: outstandingCount, amount: outstandingAmount, trend: trend(outstandingCount, prevOutstanding) },
      overdue: { count: overdueCount, amount: overdueAmount, trend: trend(overdueCount, prevOverdue) },
      renewalDue: { count: renewalCount ?? 0, trend: 0 },
      statusBreakdown: Object.entries(statusBreakdownMap).map(([status, count]) => ({
        status: status as InvoiceStatus,
        count,
      })),
      monthly,
      chapterStats,
    }
  },
}
