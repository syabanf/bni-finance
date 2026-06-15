import { Link, useNavigate } from 'react-router-dom'
import {
  ArrowUpRight,
  CalendarClock,
  CheckCircle2,
  CreditCard,
  Plus,
  TriangleAlert,
  Wallet,
} from 'lucide-react'
import type { DashboardSummary, InvoiceStatus, InvoiceWithRelations } from '@/types'
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
import { formatCurrencyCompact } from '@/lib/format'
import { InvoiceTable } from '@/features/invoices/components/InvoiceTable'

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
