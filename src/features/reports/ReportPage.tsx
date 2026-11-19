import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CheckCircle2, CreditCard, Percent, Wallet } from 'lucide-react'
import type { ReportDetailRow, ReportFilters, ReportSummary } from '@/types'
import {
  Card,
  CardHeader,
  DateRangeFilter,
  DonutChart,
  EmptyState,
  ErrorState,
  ExportMenu,
  InvoiceStatusBadge,
  PageHeader,
  StatCard,
  TableSkeleton,
  type DonutSegment,
  useToast,
  Pagination,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { reportService } from '@/services'
import { formatCurrency, formatCurrencyCompact, formatDate, formatDateTime } from '@/lib/format'
import { paymentMethodLabel } from '@/lib/paymentMethod'
import { todayISO } from '@/lib/date'
import { downloadXlsxSheets } from '@/lib/xlsx'
import { LaporanPengirimanCard } from './components/LaporanPengirimanCard'
import { printTableReport } from '@/lib/pdfReport'
import { cn } from '@/lib/cn'
import { InvoiceTypeFilter, type InvoiceTypeFilterValue } from '@/features/invoices/components/InvoiceTypeFilter'
import { KonfirmasiPembayaranCard } from './components/KonfirmasiPembayaranCard'
import { INVOICE_STATUS_LABEL } from '@/lib/status'
import { usePaginasi } from '@/hooks/usePaginasi'
import { useHalamanServer } from '@/hooks/useHalamanServer'

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

/** Angka sebelum jawaban server tiba. */
const KOSONG: ReportSummary = {
  count: 0, ditagih: 0, diterima: 0, outstanding: 0, rate: 0,
  chapters: [], types: { registration: 0, renewal: 0 }, monthly: [], methods: [], detailDiterima: 0,
}

const TIPE_LABEL = { registration: 'Pendaftaran', renewal: 'Renewal' } as const

export function ReportPage() {
  const navigate = useNavigate()
  const { toast } = useToast()
  const [preset, setPreset] = useState<Preset>('this-year')
  const [tipe, setTipe] = useState<InvoiceTypeFilterValue>('all')
  const [customFrom, setCustomFrom] = useState('')
  const [customTo, setCustomTo] = useState('')

  const range = preset === 'custom' ? { from: customFrom, to: customTo } : presetRange(preset)

  // Stabil antar-render: useHalamanServer membaca identitas baru sebagai
  // saringan baru dan mengembalikan tabel detail ke halaman satu.
  const filter = useMemo<ReportFilters>(
    () => ({ from: range.from, to: range.to, type: tipe }),
    [range.from, range.to, tipe],
  )

  // Seluruh angka dihitung server dari view v_laporan_invoice dan
  // v_laporan_pembayaran. Sebelumnya seluruh invoice dan pembayaran ditarik ke
  // browser untuk dijumlahkan di sini, satu permintaan per 200 baris.
  const {
    data: ringkas,
    loading,
    error: galatRingkas,
    reload: muatRingkas,
  } = useAsync<ReportSummary>(() => reportService.summary(filter), [filter])
  const detailHal = useHalamanServer((f: ReportFilters) => reportService.detailPaged(f), filter)
  const error = galatRingkas || detailHal.error
  const reloadAll = () => {
    muatRingkas()
    detailHal.reload()
  }

  const report = useMemo(() => {
    const r = ringkas ?? KOSONG
    return {
      ...r,
      // Nilai turun dulu, lalu abjad. Tanpa pengurutan kedua, chapter yang
      // sama-sama nol muncul dalam urutan acak yang berubah tiap muat.
      chapterRows: [...r.chapters].sort((a, b) => b.ditagih - a.ditagih || a.name.localeCompare(b.name)),
      typeData: [
        { label: 'Pendaftaran', value: r.types.registration, color: '#ef4444' },
        { label: 'Renewal', value: r.types.renewal, color: '#3b82f6' },
      ] satisfies DonutSegment[],
    }
  }, [ringkas])
  const chapterHal = usePaginasi(report.chapterRows, 10)
  const detail = detailHal.halaman?.rows ?? []
  const totalDetail = detailHal.halaman?.total ?? 0

  const periodLabel = PRESETS.find((p) => p.value === preset)?.label ?? 'Semua'
  const tipeLabel = tipe === 'all' ? '' : tipe === 'renewal' ? ' · Renewal' : ' · Pendaftaran'
  const periodRange = `${range.from ? formatDate(range.from) : 'awal'} – ${formatDate(range.to || todayISO())}`

  const EXPORT_HEADERS = [
    'Chapter',
    'Jumlah Invoice',
    'Renewal',
    'Pendaftaran',
    'Ditagih',
    'Diterima',
    'Outstanding',
    'Collection Rate',
  ]
  // Raw rows (amounts numeric so Excel can sum) — reflects the active period.
  const exportRows = () =>
    report.chapterRows.map((r) => [
      r.name, r.count, r.renewal, r.pendaftaran, r.ditagih, r.diterima, r.outstanding, `${r.rate}%`,
    ])
  const exportBase = `laporan${tipe === 'all' ? '' : `-${tipe}`}-${range.from || 'awal'}_${range.to || todayISO()}`

  const DETAIL_HEADERS = ['Nomor', 'Member', 'Chapter', 'Tipe', 'Terbit', 'Jatuh Tempo', 'Status', 'Ditagih', 'Diterima']
  const detailRows = (semua: ReportDetailRow[], uang: (n: number) => string | number = (n) => n) =>
    semua.map((d) => [
      d.number,
      d.member ?? '—',
      d.chapter ?? '—',
      TIPE_LABEL[d.type],
      formatDate(d.terbit),
      formatDate(d.jatuhTempo),
      INVOICE_STATUS_LABEL[d.status],
      uang(d.ditagih),
      uang(d.diterima),
    ])

  // Dua lembar: rekap per chapter dan detail per invoice. Detail di lembar
  // sendiri supaya rekapnya tetap bisa dijumlahkan tanpa tercampur.
  // Tabel detail di layar berhalaman, jadi ekspor meminta seluruh barisnya.
  const [mengekspor, setMengekspor] = useState(false)
  const ekspor = async (jenis: 'excel' | 'pdf') => {
    setMengekspor(true)
    try {
      const semua = await reportService.detail(filter)
      if (jenis === 'excel') exportExcel(semua)
      else exportPdf(semua)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal mengekspor laporan.', 'error')
    } finally {
      setMengekspor(false)
    }
  }

  const exportExcel = (semua: ReportDetailRow[]) =>
    downloadXlsxSheets(exportBase, [
      { name: 'Per Chapter', headers: EXPORT_HEADERS, rows: exportRows() },
      { name: 'Detail Invoice', headers: DETAIL_HEADERS, rows: detailRows(semua) },
    ])

  const exportPdf = (semua: ReportDetailRow[]) => {
    const ok = printTableReport({
      title: 'Laporan Keuangan',
      subtitle: `${periodLabel}${tipeLabel} · ${periodRange}`,
      meta: [`${report.count} invoice`, `Dibuat ${formatDateTime(new Date())}`],
      summary: [
        { label: 'Total Ditagih', value: formatCurrency(report.ditagih) },
        { label: 'Total Diterima', value: formatCurrency(report.diterima) },
        { label: 'Outstanding', value: formatCurrency(report.outstanding) },
        { label: 'Collection Rate', value: `${report.rate}%` },
      ],
      tableHeading: 'Rincian per Chapter',
      columns: [
        { label: 'Chapter' },
        { label: 'Invoice', align: 'center' },
        { label: 'Renewal', align: 'right' },
        { label: 'Pendaftaran', align: 'right' },
        { label: 'Ditagih', align: 'right' },
        { label: 'Diterima', align: 'right' },
        { label: 'Outstanding', align: 'right' },
        { label: 'Rate', align: 'right' },
      ],
      rows: report.chapterRows.map((r) => [
        r.name,
        r.count,
        formatCurrency(r.renewal),
        formatCurrency(r.pendaftaran),
        formatCurrency(r.ditagih),
        formatCurrency(r.diterima),
        formatCurrency(r.outstanding),
        `${r.rate}%`,
      ]),
      totals: [
        'Total',
        report.count,
        formatCurrency(report.chapterRows.reduce((a, r) => a + r.renewal, 0)),
        formatCurrency(report.chapterRows.reduce((a, r) => a + r.pendaftaran, 0)),
        formatCurrency(report.ditagih),
        formatCurrency(report.diterima),
        formatCurrency(report.outstanding),
        `${report.rate}%`,
      ],
      extraSections: [
        {
          heading: 'Detail Invoice',
          columns: [
            { label: 'Nomor' },
            { label: 'Member' },
            { label: 'Chapter' },
            { label: 'Tipe' },
            { label: 'Terbit' },
            { label: 'Jatuh Tempo' },
            { label: 'Status' },
            { label: 'Ditagih', align: 'right' },
            { label: 'Diterima', align: 'right' },
          ],
          rows: detailRows(semua, formatCurrency),
        },
        {
          heading: 'Per Tipe Invoice',
          columns: [{ label: 'Tipe' }, { label: 'Jumlah', align: 'right' }],
          rows: report.typeData.map((t) => [t.label, t.value]),
        },
        {
          heading: 'Metode Pembayaran',
          columns: [
            { label: 'Metode' },
            { label: 'Transaksi', align: 'right' },
            { label: 'Nominal', align: 'right' },
          ],
          rows: report.methods.map((m) => [
            paymentMethodLabel(m.method),
            m.count,
            formatCurrency(m.amount),
          ]),
        },
      ],
      documentTitle: 'Laporan Keuangan — Business Network International Finance',
    })
    if (!ok) toast('Izinkan popup di browser untuk mengekspor PDF.', 'error')
  }

  if (error) {
    return (
      <div>
        <PageHeader
          title="Laporan Keuangan"
          description="Ringkasan penagihan dan penerimaan per periode."
        />
        <Card>
          <ErrorState message={error} onRetry={reloadAll} />
        </Card>
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Laporan Keuangan"
        description="Ringkasan penagihan dan penerimaan per periode."
        action={
          <span data-tour="report-export">
            <ExportMenu
              onExcel={() => void ekspor('excel')}
              onPdf={() => void ekspor('pdf')}
              disabled={report.chapterRows.length === 0 || mengekspor}
            />
          </span>
        }
      />

      {/* Period filter */}
      <Card data-tour="report-range" className="mb-5">
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
            <DateRangeFilter
              label=""
              from={customFrom}
              to={customTo}
              onFrom={setCustomFrom}
              onTo={setCustomTo}
            />
          )}
          <span className="ml-auto text-xs text-ink-400">
            {periodRange}
          </span>
        </div>
        <div className="border-t border-ink-100 px-4 py-3">
          <InvoiceTypeFilter value={tipe} onChange={setTipe} />
        </div>
      </Card>

      {/* KPIs */}
      {loading ? (
        <div className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="surface h-[148px] animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
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
            hint="Diterima periode ini"
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

      <KonfirmasiPembayaranCard />

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
                    <th className="px-3 py-3 text-right">Renewal</th>
                    <th className="px-3 py-3 text-right">Pendaftaran</th>
                    <th className="px-3 py-3 text-right">Ditagih</th>
                    <th className="px-3 py-3 text-right">Diterima</th>
                    <th className="px-3 py-3 text-right">Outstanding</th>
                    <th className="px-5 py-3 text-right">Rate</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-ink-50">
                  {chapterHal.potongan.map((r) => (
                    <tr
                      key={r.id}
                      onClick={() => navigate(`/invoices?chapter=${r.id}`)}
                      className="cursor-pointer hover:bg-ink-50/50"
                    >
                      <td className="px-5 py-3 font-medium text-ink-900">{r.name}</td>
                      <td className="px-3 py-3 text-center text-ink-600">{r.count}</td>
                      <td className="px-3 py-3 text-right text-ink-600">{formatCurrency(r.renewal)}</td>
                      <td className="px-3 py-3 text-right text-ink-600">{formatCurrency(r.pendaftaran)}</td>
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
                    <td className="px-3 py-3 text-right text-ink-600">
                      {formatCurrency(report.chapterRows.reduce((a, r) => a + r.renewal, 0))}
                    </td>
                    <td className="px-3 py-3 text-right text-ink-600">
                      {formatCurrency(report.chapterRows.reduce((a, r) => a + r.pendaftaran, 0))}
                    </td>
                    <td className="px-3 py-3 text-right">{formatCurrency(report.ditagih)}</td>
                    <td className="px-3 py-3 text-right text-emerald-600">{formatCurrency(report.diterima)}</td>
                    <td className="px-3 py-3 text-right text-amber-600">{formatCurrency(report.outstanding)}</td>
                    <td className="px-5 py-3 text-right">{report.rate}%</td>
                  </tr>
                </tfoot>
              </table>
            </div>
          )}
          {!loading && <Pagination {...chapterHal.paginasi} satuan="chapter" />}
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
                      <span className="text-ink-600">{paymentMethodLabel(m.method)}</span>
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

      {/* Detail per invoice */}
      <Card className="mt-5">
        <CardHeader
          title="Detail Invoice"
          subtitle="Setiap invoice yang terbit pada periode, berikut penerimaannya. Klik baris untuk membukanya."
        />
        {detailHal.loading && !detailHal.halaman ? (
          <TableSkeleton rows={6} cols={6} />
        ) : totalDetail === 0 ? (
          <EmptyState icon={CreditCard} title="Tidak ada data" description="Tidak ada invoice pada periode ini." />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-y border-ink-100 text-left text-xs font-semibold uppercase tracking-wide text-ink-400">
                  <th className="px-5 py-3">Nomor</th>
                  <th className="px-3 py-3">Member</th>
                  <th className="px-3 py-3">Chapter</th>
                  <th className="px-3 py-3">Tipe</th>
                  <th className="px-3 py-3">Terbit</th>
                  <th className="px-3 py-3">Jatuh Tempo</th>
                  <th className="px-3 py-3">Status</th>
                  <th className="px-3 py-3 text-right">Ditagih</th>
                  <th className="px-5 py-3 text-right">Diterima</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-50">
                {detail.map((d) => (
                  <tr
                    key={d.id}
                    onClick={() => navigate(`/invoices/${d.id}`)}
                    className="cursor-pointer hover:bg-ink-50/50"
                  >
                    <td className="px-5 py-3 font-medium text-ink-900">{d.number}</td>
                    <td className="px-3 py-3 text-ink-700">{d.member ?? '—'}</td>
                    <td className="px-3 py-3 text-ink-600">{d.chapter ?? '—'}</td>
                    <td className="px-3 py-3 text-ink-600">{TIPE_LABEL[d.type]}</td>
                    <td className="px-3 py-3 text-ink-600">{formatDate(d.terbit)}</td>
                    <td className="px-3 py-3 text-ink-600">{formatDate(d.jatuhTempo)}</td>
                    <td className="px-3 py-3">
                      <InvoiceStatusBadge status={d.status} />
                    </td>
                    <td className="px-3 py-3 text-right text-ink-700">{formatCurrency(d.ditagih)}</td>
                    <td className="px-5 py-3 text-right text-emerald-600">
                      {d.diterima > 0 ? formatCurrency(d.diterima) : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr className="border-t border-ink-100 bg-ink-50/50 text-sm font-semibold text-ink-900">
                  <td className="px-5 py-3" colSpan={7}>
                    Total · {totalDetail.toLocaleString('id-ID')} invoice
                  </td>
                  <td className="px-3 py-3 text-right">{formatCurrency(report.ditagih)}</td>
                  <td className="px-5 py-3 text-right text-emerald-600">
                    {formatCurrency(report.detailDiterima)}
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>
        )}
        <Pagination {...detailHal.paginasi} satuan="invoice" />
      </Card>

      <LaporanPengirimanCard dari={range.from} sampai={range.to} tipe={tipe} namaBerkas={exportBase} />
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
          /* Tooltip menempel pada SATU kolom bulan, bukan pada tiap batang.
             Angka yang dicari orang adalah perbandingannya — "bulan itu ditagih
             berapa dan masuk berapa" — dan tooltip per batang memaksa mereka
             mengarahkan kursor dua kali lalu mengingat angka pertama. */
          <div
            key={d.month}
            className="group relative flex min-w-[36px] flex-1 flex-col items-center gap-2"
          >
            <div className="flex h-[160px] w-full items-end justify-center gap-1">
              <div
                className="w-1/2 max-w-[16px] rounded-t bg-ink-300 transition-opacity group-hover:opacity-80"
                style={{ height: `${(d.issued / max) * 100}%` }}
              />
              <div
                className="w-1/2 max-w-[16px] rounded-t bg-brand-500 transition-opacity group-hover:opacity-80"
                style={{ height: `${(d.paid / max) * 100}%` }}
              />
            </div>
            <span className="whitespace-nowrap text-[11px] text-ink-400">{monthLabel(d.month)}</span>

            <div className="pointer-events-none absolute bottom-full left-1/2 z-10 mb-2 hidden -translate-x-1/2 whitespace-nowrap rounded-lg bg-ink-900 px-3 py-2 text-xs text-white shadow-lg group-hover:block">
              <div className="mb-1 font-semibold">{monthLabel(d.month)}</div>
              <div className="flex items-center gap-1.5">
                <span className="h-2 w-2 rounded-sm bg-ink-300" />
                Ditagih {formatCurrency(d.issued)}
              </div>
              <div className="mt-0.5 flex items-center gap-1.5">
                <span className="h-2 w-2 rounded-sm bg-brand-500" />
                Diterima {formatCurrency(d.paid)}
              </div>
              {d.issued > 0 && (
                <div className="mt-1 border-t border-white/20 pt-1 text-white/70">
                  {Math.round((d.paid / d.issued) * 100)}% tertagih
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
