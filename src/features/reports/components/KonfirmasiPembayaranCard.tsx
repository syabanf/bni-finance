import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ClipboardCheck } from 'lucide-react'
import type { Invoice, RenewalRequest } from '@/types'
import {
  Card,
  CardHeader,
  EmptyState,
  ErrorState,
  ExportMenu,
  InvoiceStatusBadge,
  Select,
  TableSkeleton,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { invoiceService, renewalService } from '@/services'
import { formatDate, formatDateTime } from '@/lib/format'
import { downloadXlsxSheets } from '@/lib/xlsx'
import { printTableReport } from '@/lib/pdfReport'
import { cn } from '@/lib/cn'

/**
 * Member yang konfirmasi renewal-nya dijawab MC "Diterima", dan sudah sejauh
 * mana pembayarannya.
 *
 * Untuk ST: daftar siapa yang perlu diingatkan. Datanya dibatasi chapter oleh
 * server, jadi ST satu chapter hanya melihat chapternya dan ST nasional melihat
 * semuanya dengan rekap per chapter. Yang dihitung adalah permintaan
 * konfirmasi TERAKHIR tiap member, aturan yang sama dengan yang membuka
 * penerbitan invoice renewal.
 */

type Tahap = 'lunas' | 'belum_bayar' | 'belum_terbit' | 'belum_ditagih'

const TAHAP: { value: Tahap | 'semua'; label: string }[] = [
  { value: 'semua', label: 'Semua' },
  { value: 'belum_bayar', label: 'Belum Bayar' },
  { value: 'belum_ditagih', label: 'Belum Ditagih' },
  { value: 'belum_terbit', label: 'Draft' },
  { value: 'lunas', label: 'Lunas' },
]

const LABEL_TAHAP: Record<Tahap, string> = {
  lunas: 'Lunas',
  belum_bayar: 'Belum bayar',
  belum_terbit: 'Invoice masih draft',
  belum_ditagih: 'Belum ditagih',
}

interface Baris {
  request: RenewalRequest
  invoice: Invoice | null
  tahap: Tahap
}

/** Semua invoice renewal, dibaca per 200 sampai habis supaya rekapnya tidak terpotong. */
async function semuaInvoiceRenewal(): Promise<Invoice[]> {
  const out: Invoice[] = []
  for (;;) {
    const halaman = await invoiceService.listPaged({ type: 'renewal', limit: 200, offset: out.length })
    out.push(...halaman.rows)
    if (halaman.rows.length === 0 || out.length >= halaman.total) return out
  }
}

function tahapDari(inv: Invoice | null): Tahap {
  if (!inv) return 'belum_ditagih'
  if (inv.status === 'paid') return 'lunas'
  if (inv.status === 'draft') return 'belum_terbit'
  return 'belum_bayar'
}

export function KonfirmasiPembayaranCard() {
  const navigate = useNavigate()
  const { toast } = useToast()
  const data = useAsync(
    () => Promise.all([renewalService.list(), semuaInvoiceRenewal()]).then(([r, i]) => ({ requests: r, invoices: i })),
    [],
  )
  const [periode, setPeriode] = useState('semua')
  const [tahap, setTahap] = useState<Tahap | 'semua'>('belum_bayar')

  const daftarPeriode = useMemo(
    () => [...new Set((data.data?.requests ?? []).map((r) => r.period))].sort().reverse(),
    [data.data],
  )

  const baris = useMemo<Baris[]>(() => {
    if (!data.data) return []
    const terakhir = new Map<string, RenewalRequest>()
    for (const r of data.data.requests) {
      const ada = terakhir.get(r.memberId)
      if (!ada || r.requestedAt > ada.requestedAt) terakhir.set(r.memberId, r)
    }
    const invoicePerMember = new Map<string, Invoice[]>()
    for (const i of data.data.invoices) {
      if (i.status === 'cancelled' || i.status === 'terminated') continue
      invoicePerMember.set(i.memberId, [...(invoicePerMember.get(i.memberId) ?? []), i])
    }
    return [...terakhir.values()]
      .filter((r) => r.answer === 'will_renew' && (periode === 'semua' || r.period === periode))
      .map((r) => {
        // Invoice renewal terbaru yang terbit sesudah permintaan konfirmasinya.
        const inv =
          (invoicePerMember.get(r.memberId) ?? [])
            .filter((i) => i.createdAt >= r.requestedAt)
            .sort((a, b) => b.createdAt.localeCompare(a.createdAt))[0] ?? null
        return { request: r, invoice: inv, tahap: tahapDari(inv) }
      })
      .sort(
        (a, b) =>
          (a.request.chapterName ?? '').localeCompare(b.request.chapterName ?? '') ||
          (a.request.memberName ?? '').localeCompare(b.request.memberName ?? ''),
      )
  }, [data.data, periode])

  const rekap = useMemo(() => {
    const per = new Map<string, Record<Tahap | 'total', number>>()
    for (const b of baris) {
      const nama = b.request.chapterName ?? b.request.chapterId
      const r = per.get(nama) ?? { total: 0, lunas: 0, belum_bayar: 0, belum_terbit: 0, belum_ditagih: 0 }
      r.total++
      r[b.tahap]++
      per.set(nama, r)
    }
    return [...per.entries()]
  }, [baris])

  const terlihat = tahap === 'semua' ? baris : baris.filter((b) => b.tahap === tahap)
  const jumlah = (t: Tahap) => baris.filter((b) => b.tahap === t).length

  const KOLOM = ['Member', 'Chapter', 'Periode', 'Dijawab MC', 'Invoice', 'Status', 'Diingatkan', 'Jatuh Tempo']
  const barisEkspor = () =>
    terlihat.map((b) => [
      b.request.memberName ?? b.request.memberId,
      b.request.chapterName ?? b.request.chapterId,
      b.request.period,
      b.request.answeredAt ? formatDate(b.request.answeredAt) : '',
      b.invoice?.number ?? '',
      LABEL_TAHAP[b.tahap],
      b.invoice?.paperIdReminderCount ?? 0,
      b.invoice ? formatDate(b.invoice.dueDate) : '',
    ])
  const judulTahap = TAHAP.find((t) => t.value === tahap)?.label ?? 'Semua'
  const namaBerkas = `konfirmasi-mc-${tahap}${periode === 'semua' ? '' : `-${periode}`}`

  const eksporExcel = () =>
    downloadXlsxSheets(namaBerkas, [
      { name: 'Per Chapter', headers: ['Chapter', 'Diterima MC', 'Lunas', 'Belum Bayar', 'Draft', 'Belum Ditagih'],
        rows: rekap.map(([c, r]) => [c, r.total, r.lunas, r.belum_bayar, r.belum_terbit, r.belum_ditagih]) },
      { name: 'Member', headers: KOLOM, rows: barisEkspor() },
    ])
  const eksporPdf = () => {
    const ok = printTableReport({
      title: 'Konfirmasi MC & Pembayaran',
      subtitle: `${judulTahap} · ${periode === 'semua' ? 'Semua periode' : periode}`,
      meta: [`${terlihat.length} member`, `Dibuat ${formatDateTime(new Date())}`],
      summary: [
        { label: 'Diterima MC', value: String(baris.length) },
        { label: 'Lunas', value: String(jumlah('lunas')) },
        { label: 'Belum Bayar', value: String(jumlah('belum_bayar')) },
        { label: 'Belum Ditagih', value: String(jumlah('belum_ditagih')) },
      ],
      tableHeading: 'Member',
      columns: KOLOM.map((label) => ({ label })),
      rows: barisEkspor(),
      documentTitle: 'Konfirmasi MC & Pembayaran',
    })
    if (!ok) toast('Izinkan popup di browser untuk mengekspor PDF.', 'error')
  }

  return (
    <Card className="mt-5">
      <CardHeader
        title="Konfirmasi MC & Pembayaran"
        subtitle="Member yang konfirmasi renewal terakhirnya dijawab MC &quot;Diterima&quot;, dan status invoicenya. Tidak mengikuti filter periode di atas."
        action={
          <div className="flex flex-wrap items-center gap-2">
            <Select value={periode} onChange={(e) => setPeriode(e.target.value)} aria-label="Periode konfirmasi" className="w-40">
              <option value="semua">Semua periode</option>
              {daftarPeriode.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </Select>
            <ExportMenu onExcel={eksporExcel} onPdf={eksporPdf} disabled={terlihat.length === 0} />
          </div>
        }
      />
      {data.loading ? (
        <TableSkeleton rows={5} cols={5} />
      ) : data.error ? (
        <ErrorState message={data.error} onRetry={data.reload} />
      ) : baris.length === 0 ? (
        <EmptyState
          icon={ClipboardCheck}
          title="Belum ada konfirmasi yang diterima"
          description="Member muncul di sini setelah MC menjawab konfirmasi renewal dengan &quot;Diterima&quot;."
        />
      ) : (
        <div className="border-t border-ink-100">
          {/* Rekap per chapter: ST nasional melihat semua chapter, ST satu chapter hanya miliknya. */}
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-ink-100 text-left text-xs font-semibold uppercase tracking-wide text-ink-400">
                  <th className="px-5 py-3">Chapter</th>
                  <th className="px-3 py-3 text-right">Diterima MC</th>
                  <th className="px-3 py-3 text-right">Lunas</th>
                  <th className="px-3 py-3 text-right">Belum Bayar</th>
                  <th className="px-3 py-3 text-right">Draft</th>
                  <th className="px-5 py-3 text-right">Belum Ditagih</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-50">
                {rekap.map(([c, r]) => (
                  <tr key={c}>
                    <td className="px-5 py-2.5 font-medium text-ink-900">{c}</td>
                    <td className="px-3 py-2.5 text-right text-ink-700">{r.total}</td>
                    <td className="px-3 py-2.5 text-right text-emerald-600">{r.lunas}</td>
                    <td className="px-3 py-2.5 text-right font-semibold text-amber-600">{r.belum_bayar}</td>
                    <td className="px-3 py-2.5 text-right text-ink-500">{r.belum_terbit}</td>
                    <td className="px-5 py-2.5 text-right text-red-600">{r.belum_ditagih}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="flex flex-wrap gap-1.5 border-t border-ink-100 px-5 py-3">
            {TAHAP.map((t) => (
              <button
                key={t.value}
                type="button"
                onClick={() => setTahap(t.value)}
                className={cn(
                  'rounded-lg px-3 py-1.5 text-[13px] font-medium transition-colors',
                  tahap === t.value ? 'bg-brand-500 text-white' : 'bg-ink-100 text-ink-600 hover:bg-ink-200',
                )}
              >
                {t.label}
                <span className="ml-1.5 opacity-70">{t.value === 'semua' ? baris.length : jumlah(t.value)}</span>
              </button>
            ))}
          </div>

          {terlihat.length === 0 ? (
            <p className="px-5 pb-6 pt-2 text-sm text-ink-400">Tidak ada member pada tahap ini.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-y border-ink-100 text-left text-xs font-semibold uppercase tracking-wide text-ink-400">
                    <th className="px-5 py-3">Member</th>
                    <th className="px-3 py-3">Chapter</th>
                    <th className="px-3 py-3">Periode</th>
                    <th className="px-3 py-3">Dijawab MC</th>
                    <th className="px-3 py-3">Invoice</th>
                    <th className="px-3 py-3 text-right">Diingatkan</th>
                    <th className="px-5 py-3">Jatuh Tempo</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-ink-50">
                  {terlihat.map((b) => (
                    <tr
                      key={b.request.id}
                      onClick={b.invoice ? () => navigate(`/invoices/${b.invoice!.id}`) : undefined}
                      className={cn(b.invoice && 'cursor-pointer hover:bg-ink-50/50')}
                    >
                      <td className="px-5 py-3 font-medium text-ink-900">{b.request.memberName ?? b.request.memberId}</td>
                      <td className="px-3 py-3 text-ink-600">{b.request.chapterName ?? b.request.chapterId}</td>
                      <td className="px-3 py-3 text-ink-600">{b.request.period}</td>
                      <td className="px-3 py-3 text-ink-600">
                        {b.request.answeredAt ? formatDate(b.request.answeredAt) : '—'}
                      </td>
                      <td className="px-3 py-3">
                        {b.invoice ? (
                          <span className="flex items-center gap-2">
                            <span className="text-ink-700">{b.invoice.number}</span>
                            <InvoiceStatusBadge status={b.invoice.status} />
                          </span>
                        ) : (
                          <span className="text-red-600">Belum ditagih</span>
                        )}
                      </td>
                      <td className="px-3 py-3 text-right text-ink-600">
                        {b.invoice?.paperIdReminderCount ? `${b.invoice.paperIdReminderCount}×` : '—'}
                      </td>
                      <td className="px-5 py-3 text-ink-600">{b.invoice ? formatDate(b.invoice.dueDate) : '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </Card>
  )
}
