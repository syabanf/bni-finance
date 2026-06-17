import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CheckCircle2, CreditCard, Download, Percent, Wallet } from 'lucide-react'
import type { Chapter, InvoiceWithRelations, PaymentWithInvoice } from '@/types'
import {
  Card,
  CardHeader,
  DonutChart,
  type DonutSegment,
  EmptyState,
  Input,
  PageHeader,
  StatCard,
  TableSkeleton,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, invoiceService, paymentService } from '@/services'
import { formatCurrency, formatCurrencyCompact } from '@/lib/format'
import { todayISO } from '@/lib/date'
import { downloadCsv } from '@/lib/csv'
import { cn } from '@/lib/cn'

type Preset = 'this-month' | 'last-month' | 'this-year' | 'all' | 'custom'

const PRESETS: { value: Preset; label: string }[] = [
  { value: 'this-month', label: 'Bulan Ini' },
  { value: 'last-month', label: 'Bulan Lalu' },
  { value: 'this-year', label: 'Tahun Ini' },
  { value: 'all', label: 'Semua' },
  { value: 'custom', label: 'Kustom' },
]

const pad = (n: number) => String(n).padStart(2, '0')

function presetRange(preset: Preset): { from: string; to: string } {
  const today = todayISO()
  const d = new Date(today)
  const y = d.getFullYear()
  const m = d.getMonth()
  const firstOf = (yy: number, mm: number) => `${yy}-${pad(mm + 1)}-01`
  const lastOf = (yy: number, mm: number) => `${yy}-${pad(mm + 1)}-${pad(new Date(yy, mm + 1, 0).getDate())}`
  switch (preset) {
    case 'this-month':
      return { from: firstOf(y, m), to: today }
    case 'last-month': {
      const lm = m === 0 ? 11 : m - 1
      const ly = m === 0 ? y - 1 : y
      return { from: firstOf(ly, lm), to: lastOf(ly, lm) }
    }
    case 'this-year':
      return { from: `${y}-01-01`, to: today }
    default:
      return { from: '', to: '' }
  }
}

function monthLabel(key: string): string {
  const [y, m] = key.split('-')
  const name = new Date(Number(y), Number(m) - 1, 1).toLocaleDateString('id-ID', { month: 'short' })
  return `${name} ${y.slice(2)}`
}

const sum = (ns: number[]) => ns.reduce((a, n) => a + n, 0)

interface ChapterRow {
  id: string
  name: string
  count: number
  ditagih: number
  diterima: number
  outstanding: number
  rate: number
}

export function ReportPage() {
  const navigate = useNavigate()
  const { data: invoices, loading: invLoading } = useAsync<InvoiceWithRelations[]>(() => invoiceService.list())
  const { data: payments, loading: payLoading } = useAsync<PaymentWithInvoice[]>(() => paymentService.list())
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())

  const [preset, setPreset] = useState<Preset>('this-year')
  const [customFrom, setCustomFrom] = useState('')
  const [customTo, setCustomTo] = useState('')

  const range = preset === 'custom' ? { from: customFrom, to: customTo } : presetRange(preset)

  const inRange = useMemo(() => {
    const { from, to } = range
    return (iso?: string) => {
      if (!iso) return false
      const day = iso.slice(0, 10)
      if (from && day < from) return false
      if (to && day > to) return false
      return true
    }
  }, [range.from, range.to])

  const report = useMemo(() => {
    const invs = (invoices ?? []).filter((i) => i.status !== 'cancelled' && inRange(i.createdAt))
    const paid = invs.filter((i) => i.status === 'paid')
    const outstandingInvs = invs.filter((i) => i.status === 'sent' || i.status === 'overdue')

    const ditagih = sum(invs.map((i) => i.amount))
    const diterima = sum(paid.map((i) => i.paidAmount ?? i.amount))
    const outstanding = sum(outstandingInvs.map((i) => i.amount))
    const rate = ditagih > 0 ? Math.round((diterima / ditagih) * 100) : 0

    // Per chapter
    const chapMap = new Map<string, ChapterRow>()
    for (const i of invs) {
      const id = i.chapterId
      const name = i.chapter?.displayName ?? chapters?.find((c) => c.id === id)?.displayName ?? '—'
      const row = chapMap.get(id) ?? { id, name, count: 0, ditagih: 0, diterima: 0, outstanding: 0, rate: 0 }
      row.count += 1
      row.ditagih += i.amount
      if (i.status === 'paid') row.diterima += i.paidAmount ?? i.amount
      if (i.status === 'sent' || i.status === 'overdue') row.outstanding += i.amount
      chapMap.set(id, row)
    }
    const chapterRows = [...chapMap.values()]
      .map((r) => ({ ...r, rate: r.ditagih > 0 ? Math.round((r.diterima / r.ditagih) * 100) : 0 }))
      .sort((a, b) => b.ditagih - a.ditagih)

    // Per type
    const reg = invs.filter((i) => i.type === 'registration')
    const ren = invs.filter((i) => i.type === 'renewal')
    const typeData: DonutSegment[] = [
      { label: 'Pendaftaran', value: reg.length, color: '#ef4444' },
      { label: 'Renewal', value: ren.length, color: '#3b82f6' },
    ]

    // Monthly issued vs collected
    const monthMap = new Map<string, { issued: number; paid: number }>()
    const bump = (key: string, field: 'issued' | 'paid', amt: number) => {
      const m = monthMap.get(key) ?? { issued: 0, paid: 0 }
      m[field] += amt
      monthMap.set(key, m)
    }
    for (const i of invs) bump(i.createdAt.slice(0, 7), 'issued', i.amount)
    for (const i of paid) bump((i.paidAt ?? i.createdAt).slice(0, 7), 'paid', i.paidAmount ?? i.amount)
    const monthly = [...monthMap.entries()]
      .sort((a, b) => a[0].localeCompare(b[0]))
      .slice(-12)
      .map(([month, v]) => ({ month, ...v }))

    // Payment methods (cash actually received in the period)
    const pays = (payments ?? []).filter((p) => inRange(p.paidAt))
    const methodMap = new Map<string, { count: number; amount: number }>()
    for (const p of pays) {
      const key = (p.paymentMethod || 'Lainnya').replace(/_/g, ' ')
      const m = methodMap.get(key) ?? { count: 0, amount: 0 }
      m.count += 1
      m.amount += p.amount
      methodMap.set(key, m)
    }
    const methods = [...methodMap.entries()]
      .map(([method, v]) => ({ method, ...v }))
      .sort((a, b) => b.amount - a.amount)

    return { ditagih, diterima, outstanding, rate, count: invs.length, chapterRows, typeData, monthly, methods }
  }, [invoices, payments, chapters, inRange])

  const loading = invLoading || payLoading

  const exportCsv = () => {
    downloadCsv(
      `laporan-${range.from || 'awal'}_${range.to || todayISO()}.csv`,
      ['Chapter', 'Jumlah Invoice', 'Ditagih', 'Diterima', 'Outstanding', 'Collection Rate'],
      report.chapterRows.map((r) => [r.name, r.count, r.ditagih, r.diterima, r.outstanding, `${r.rate}%`]),
    )
  }

  return (
    <div>
      <PageHeader
        title="Laporan Keuangan"
        description="Ringkasan penagihan dan penerimaan per periode."
        action={
          <button
            onClick={exportCsv}
            disabled={report.chapterRows.length === 0}
            className="inline-flex items-center gap-2 rounded-xl border border-ink-200 bg-white px-3.5 py-2 text-sm font-medium text-ink-700 transition-colors hover:bg-ink-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Download className="h-4 w-4" />
            Export CSV
          </button>
        }
      />

      {/* Period filter */}
      <Card className="mb-5">
        <div className="flex flex-wrap items-center gap-3 p-4">
          <div className="flex flex-wrap gap-1.5">
            {PRESETS.map((p) => (
              <button
                key={p.value}
                onClick={() => setPreset(p.value)}
                className={cn(
                  'rounded-lg px-3 py-1.5 text-[13px] font-medium transition-colors',
                  preset === p.value ? 'bg-brand-500 text-white' : 'bg-ink-100 text-ink-600 hover:bg-ink-200',
                )}
              >
                {p.label}
              </button>
            ))}
          </div>
          {preset === 'custom' && (
            <div className="flex items-center gap-2">
              <Input
                type="date"
                value={customFrom}
                max={customTo || undefined}
                onChange={(e) => setCustomFrom(e.target.value)}
                className="w-[150px]"
                aria-label="Dari tanggal"
              />
              <span className="text-ink-400">–</span>
              <Input
                type="date"
                value={customTo}
                min={customFrom || undefined}
                onChange={(e) => setCustomTo(e.target.value)}
                className="w-[150px]"
                aria-label="Sampai tanggal"
              />
            </div>
          )}
          <span className="ml-auto text-xs text-ink-400">
            {range.from || 'awal'} – {range.to || todayISO()}
          </span>
        </div>
      </Card>

      {/* KPIs */}
      {loading ? (
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
            value={formatCurrencyCompact(report.ditagih)}
            label="Total Ditagih"
            hint={`${report.count} invoice`}
          />
          <StatCard
            icon={CheckCircle2}
            iconTone="green"
            value={formatCurrencyCompact(report.diterima)}
            label="Total Diterima"
            hint="Invoice lunas"
          />
          <StatCard
            icon={Wallet}
            iconTone="amber"
            value={formatCurrencyCompact(report.outstanding)}
            label="Outstanding"
            hint="Belum dibayar"
          />
          <StatCard
            icon={Percent}
            iconTone="blue"
            value={`${report.rate}%`}
            label="Collection Rate"
            hint="Diterima / ditagih"
          />
        </div>
      )}

      {/* Monthly trend */}
      <Card className="mt-5">
        <CardHeader title="Tren Bulanan" subtitle="Nilai ditagih vs diterima per bulan." />
        <div className="border-t border-ink-100 p-5">
          {loading ? (
            <TableSkeleton rows={4} cols={1} />
          ) : (
            <MonthlyBars data={report.monthly} />
          )}
        </div>
      </Card>

      {/* Chapter + breakdowns */}
      <div className="mt-5 grid grid-cols-1 gap-5 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader title="Rincian per Chapter" subtitle="Klik baris untuk membuka invoice chapter." />
          {loading ? (
            <TableSkeleton rows={6} cols={5} />
          ) : report.chapterRows.length === 0 ? (
            <EmptyState icon={CreditCard} title="Tidak ada data" description="Tidak ada invoice pada periode ini." />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-y border-ink-100 text-left text-xs font-semibold uppercase tracking-wide text-ink-400">
                    <th className="px-5 py-3">Chapter</th>
                    <th className="px-3 py-3 text-center">Invoice</th>
                    <th className="px-3 py-3 text-right">Ditagih</th>
                    <th className="px-3 py-3 text-right">Diterima</th>
                    <th className="px-3 py-3 text-right">Outstanding</th>
                    <th className="px-5 py-3 text-right">Rate</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-ink-50">
                  {report.chapterRows.map((r) => (
                    <tr
                      key={r.id}
                      onClick={() => navigate(`/invoices?chapter=${r.id}`)}
                      className="cursor-pointer hover:bg-ink-50/50"
                    >
                      <td className="px-5 py-3 font-medium text-ink-900">{r.name}</td>
                      <td className="px-3 py-3 text-center text-ink-600">{r.count}</td>
                      <td className="px-3 py-3 text-right text-ink-700">{formatCurrency(r.ditagih)}</td>
                      <td className="px-3 py-3 text-right text-emerald-600">{formatCurrency(r.diterima)}</td>
                      <td className="px-3 py-3 text-right text-amber-600">{formatCurrency(r.outstanding)}</td>
                      <td className="px-5 py-3 text-right font-semibold text-ink-900">{r.rate}%</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr className="border-t border-ink-100 bg-ink-50/50 text-sm font-semibold text-ink-900">
                    <td className="px-5 py-3">Total</td>
                    <td className="px-3 py-3 text-center">{report.count}</td>
                    <td className="px-3 py-3 text-right">{formatCurrency(report.ditagih)}</td>
                    <td className="px-3 py-3 text-right text-emerald-600">{formatCurrency(report.diterima)}</td>
                    <td className="px-3 py-3 text-right text-amber-600">{formatCurrency(report.outstanding)}</td>
                    <td className="px-5 py-3 text-right">{report.rate}%</td>
                  </tr>
                </tfoot>
              </table>
            </div>
          )}
        </Card>

        <div className="space-y-5">
          {/* Type breakdown */}
          <Card>
            <CardHeader title="Per Tipe Invoice" />
            <div className="px-5 pb-6 pt-2">
              {loading ? (
                <TableSkeleton rows={3} cols={1} />
              ) : report.count === 0 ? (
                <p className="py-6 text-center text-sm text-ink-400">Tidak ada data.</p>
              ) : (
                <DonutChart data={report.typeData} centerValue={report.count} centerLabel="Invoice" />
              )}
            </div>
          </Card>

          {/* Payment methods */}
          <Card>
            <CardHeader title="Metode Pembayaran" subtitle="Penerimaan pada periode." />
            <div className="border-t border-ink-100 p-5">
              {loading ? (
                <TableSkeleton rows={3} cols={1} />
              ) : report.methods.length === 0 ? (
                <p className="py-4 text-center text-sm text-ink-400">Belum ada pembayaran.</p>
              ) : (
                <div className="space-y-3">
                  {report.methods.map((m) => (
                    <div key={m.method} className="flex items-center justify-between gap-3 text-sm">
                      <span className="capitalize text-ink-600">{m.method}</span>
                      <div className="text-right">
                        <div className="font-semibold text-ink-900">{formatCurrency(m.amount)}</div>
                        <div className="text-xs text-ink-400">{m.count}× transaksi</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}

function MonthlyBars({ data }: { data: { month: string; issued: number; paid: number }[] }) {
  if (data.length === 0) {
    return <p className="py-6 text-center text-sm text-ink-400">Tidak ada data pada periode ini.</p>
  }
  const max = Math.max(1, ...data.map((d) => Math.max(d.issued, d.paid)))
  return (
    <div>
      <div className="mb-4 flex items-center gap-4 text-xs text-ink-500">
        <span className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-sm bg-ink-300" />
          Ditagih
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-sm bg-brand-500" />
          Diterima
        </span>
      </div>
      <div className="flex items-end gap-2 overflow-x-auto pb-1">
        {data.map((d) => (
          <div key={d.month} className="flex min-w-[36px] flex-1 flex-col items-center gap-2">
            <div className="flex h-[160px] w-full items-end justify-center gap-1">
              <div
                className="w-1/2 max-w-[16px] rounded-t bg-ink-300"
                style={{ height: `${(d.issued / max) * 100}%` }}
                title={`Ditagih ${formatCurrency(d.issued)}`}
              />
              <div
                className="w-1/2 max-w-[16px] rounded-t bg-brand-500"
                style={{ height: `${(d.paid / max) * 100}%` }}
                title={`Diterima ${formatCurrency(d.paid)}`}
              />
            </div>
            <span className="whitespace-nowrap text-[11px] text-ink-400">{monthLabel(d.month)}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
