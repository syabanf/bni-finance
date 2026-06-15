import { Link, useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  ArrowUpRight,
  CalendarClock,
  CheckCircle2,
  CreditCard,
  Plus,
  TriangleAlert,
  Wallet,
} from 'lucide-react'
import type { ChapterStat, DashboardSummary, InvoiceStatus, InvoiceWithRelations } from '@/types'
import {
  Button,
  Card,
  CardHeader,
  DonutChart,
  type DonutSegment,
  LoadingState,
  PageHeader,
  StatCard,
  TableSkeleton,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { dashboardService, invoiceService } from '@/services'
import { formatCurrency, formatCurrencyCompact } from '@/lib/format'
import { InvoiceTable } from '@/features/invoices/components/InvoiceTable'
import { cn } from '@/lib/cn'

function ChapterStatsCard({ stats }: { stats: ChapterStat[] }) {
  return (
    <Card>
      <CardHeader
        title="Statistik per Chapter"
        subtitle="Ringkasan invoice aktif, outstanding, dan overdue setiap chapter."
      />
      {/* Mobile cards */}
      <div className="divide-y divide-ink-100 lg:hidden">
        {stats.map((s) => (
          <div key={s.chapterId} className="px-4 py-3.5">
            <div className="flex items-center justify-between">
              <div className="font-medium text-ink-900 text-sm">{s.chapterName}</div>
              <div className="text-sm font-semibold text-ink-900">{formatCurrencyCompact(s.totalAmount)}</div>
            </div>
            <div className="mt-2 flex items-center gap-3 text-xs">
              <span className="text-ink-500">{s.total} invoice</span>
              {s.overdue > 0 && (
                <span className="font-semibold text-red-600">{s.overdue} overdue</span>
              )}
              {s.outstanding > 0 && (
                <span className="font-semibold text-amber-600">{s.outstanding} outstanding</span>
              )}
              <span className="text-emerald-600">{s.paid} lunas</span>
            </div>
          </div>
        ))}
      </div>
      {/* Desktop table */}
      <div className="hidden lg:block overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-ink-100 text-left text-xs font-semibold uppercase tracking-wide text-ink-400">
              <th className="px-5 py-3">Chapter</th>
              <th className="px-3 py-3 text-center">Total</th>
              <th className="px-3 py-3 text-center">Overdue</th>
              <th className="px-3 py-3 text-center">Outstanding</th>
              <th className="px-3 py-3 text-center">Lunas</th>
              <th className="px-5 py-3 text-right">Total Nilai</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-ink-50">
            {stats.map((s) => (
              <tr key={s.chapterId} className="hover:bg-ink-50/50">
                <td className="px-5 py-3 font-medium text-ink-900">{s.chapterName}</td>
                <td className="px-3 py-3 text-center text-ink-600">{s.total}</td>
                <td className="px-3 py-3 text-center">
                  <span className={cn('font-semibold', s.overdue > 0 ? 'text-red-600' : 'text-ink-300')}>
                    {s.overdue}
                  </span>
                </td>
                <td className="px-3 py-3 text-center">
                  <span className={cn('font-semibold', s.outstanding > 0 ? 'text-amber-600' : 'text-ink-300')}>
                    {s.outstanding}
                  </span>
                </td>
                <td className="px-3 py-3 text-center text-emerald-600 font-semibold">{s.paid}</td>
                <td className="px-5 py-3 text-right font-medium text-ink-900">{formatCurrency(s.totalAmount)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}

const STATUS_META: Record<InvoiceStatus, { label: string; color: string }> = {
  paid: { label: 'Lunas', color: '#10b981' },
  sent: { label: 'Menunggu', color: '#f59e0b' },
  overdue: { label: 'Overdue', color: '#ef4444' },
  draft: { label: 'Draft', color: '#94a3b8' },
  cancelled: { label: 'Dibatalkan', color: '#cbd5e1' },
}

export function DashboardPage() {
  const navigate = useNavigate()
  const { data: summary, loading } = useAsync<DashboardSummary>(() => dashboardService.summary())
  const { data: recent, loading: recentLoading } = useAsync<InvoiceWithRelations[]>(() =>
    invoiceService.list(),
  )

  const donutData: DonutSegment[] =
    summary?.statusBreakdown.map((s) => ({
      label: STATUS_META[s.status].label,
      value: s.count,
      color: STATUS_META[s.status].color,
    })) ?? []

  const totalInvoices = summary?.statusBreakdown.reduce((acc, s) => acc + s.count, 0) ?? 0

  return (
    <div>
      <PageHeader
        title="Dashboard"
        description="Ringkasan invoice, pembayaran, dan keanggotaan."
        action={
          <Button onClick={() => navigate('/invoices/new')}>
            <Plus className="h-4 w-4" />
            Buat Invoice
          </Button>
        }
      />

      {/* Urgent banner */}
      {summary && (summary.overdue.count > 0 || summary.renewalDue.count > 0) && (
        <button
          onClick={() => navigate('/urgent')}
          className="mb-4 flex w-full items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-left transition-colors hover:bg-red-100"
        >
          <AlertTriangle className="h-5 w-5 shrink-0 text-red-500" />
          <div className="flex-1 text-sm">
            <span className="font-semibold text-red-700">Perlu tindakan segera: </span>
            <span className="text-red-600">
              {[
                summary.overdue.count > 0 && `${summary.overdue.count} invoice overdue`,
                summary.renewalDue.count > 0 && `${summary.renewalDue.count} member akan renewal`,
              ]
                .filter(Boolean)
                .join(' · ')}
            </span>
          </div>
          <ArrowUpRight className="h-4 w-4 shrink-0 text-red-400" />
        </button>
      )}

      {/* KPI cards */}
      {loading || !summary ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="surface h-[148px] animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            icon={CreditCard}
            iconTone="brand"
            value={summary.total.count}
            label="Total Invoice"
            hint={formatCurrencyCompact(summary.total.amount)}
            trend={{ value: summary.total.trend, intent: summary.total.trend >= 0 ? 'good' : 'bad' }}
            onClick={() => navigate('/invoices')}
          />
          <StatCard
            icon={CheckCircle2}
            iconTone="green"
            value={summary.paid.count}
            label="Sudah Dibayar"
            hint={formatCurrencyCompact(summary.paid.amount)}
            trend={{ value: summary.paid.trend, intent: summary.paid.trend >= 0 ? 'good' : 'bad' }}
            onClick={() => navigate('/invoices?status=paid')}
          />
          <StatCard
            icon={Wallet}
            iconTone="amber"
            value={summary.outstanding.count}
            label="Outstanding"
            hint={formatCurrencyCompact(summary.outstanding.amount)}
            trend={{ value: summary.outstanding.trend, intent: 'bad' }}
            onClick={() => navigate('/invoices?status=sent')}
          />
          <StatCard
            icon={TriangleAlert}
            iconTone="red"
            value={summary.overdue.count}
            label="Overdue"
            hint={formatCurrencyCompact(summary.overdue.amount)}
            trend={{ value: summary.overdue.trend, format: 'absolute', intent: 'bad' }}
            onClick={() => navigate('/invoices?status=overdue')}
          />
        </div>
      )}

      {/* Chapter stats */}
      {summary && summary.chapterStats.length > 0 && (
        <div className="mt-5">
          <ChapterStatsCard stats={summary.chapterStats} />
        </div>
      )}

      {/* Main grid */}
      <div className="mt-5 grid grid-cols-1 gap-5 lg:grid-cols-3">
        {/* Recent invoices */}
        <Card className="lg:col-span-2">
          <CardHeader
            title="Invoice Terbaru"
            subtitle="Aktivitas invoice terkini di seluruh chapter"
            action={
              <Link
                to="/invoices"
                className="inline-flex items-center gap-1 text-sm font-medium text-brand-500 hover:text-brand-600"
              >
                Lihat semua
                <ArrowUpRight className="h-4 w-4" />
              </Link>
            }
          />
          {recentLoading || !recent ? (
            <TableSkeleton rows={6} cols={6} />
          ) : (
            <>
              <InvoiceTable invoices={recent.slice(0, 6)} compact />
              <div className="px-5 py-3 text-xs text-ink-400">
                Menampilkan {Math.min(6, recent.length)} dari {recent.length} invoice
              </div>
            </>
          )}
        </Card>

        {/* Right rail */}
        <div className="space-y-5">
          <Card>
            <CardHeader title="Status Pembayaran" />
            <div className="px-5 pb-6 pt-2">
              {loading || !summary ? (
                <LoadingState />
              ) : (
                <DonutChart
                  data={donutData}
                  centerValue={totalInvoices}
                  centerLabel="Total Invoice"
                />
              )}
            </div>
          </Card>

          {/* Renewal due callout */}
          {summary && (
            <button
              onClick={() => navigate('/invoices/renewal-due')}
              className="surface group flex w-full items-center gap-4 p-5 text-left transition-shadow hover:shadow-card-hover"
            >
              <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-amber-50 text-amber-500">
                <CalendarClock className="h-[22px] w-[22px]" />
              </div>
              <div className="flex-1">
                <div className="text-sm font-semibold text-ink-900">
                  {summary.renewalDue.count} member akan jatuh tempo
                </div>
                <div className="text-xs text-ink-500">Renewal dalam 30 hari ke depan</div>
              </div>
              <ArrowUpRight className="h-4 w-4 text-ink-300 transition-colors group-hover:text-brand-500" />
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
