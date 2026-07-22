import { useMemo, useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ChevronLeft, ChevronRight, FileText, Mail, Plus, Search, Send, X } from 'lucide-react'
import type { Chapter, InvoiceStatus, InvoiceType, InvoiceWithRelations } from '@/types'
import {
  Button,
  Card,
  EmptyState,
  ExportMenu,
  Input,
  PageHeader,
  Select,
  ErrorState,
  SummaryCard,
  TableSkeleton,
  useToast,
  WhatsAppIcon,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { useCan } from '@/features/auth/usePermission'
import { chapterService, invoiceService } from '@/services'
import { isSelfPaymentMode } from '@/services/api/paymentGateway'
import { InvoiceTable } from './components/InvoiceTable'
import { cn } from '@/lib/cn'
import { formatCurrency, formatCurrencyCompact, formatDate, formatDateTime } from '@/lib/format'
import { isOutstanding, INVOICE_STATUS_LABEL } from '@/lib/status'
import { normalizePhone } from '@/lib/whatsapp'
import { daysUntil } from '@/lib/date'
import { downloadCsv } from '@/lib/csv'
import { downloadXlsx } from '@/lib/xlsx'
import { printTableReport } from '@/lib/pdfReport'

type StatusFilter = InvoiceStatus | 'all' | 'outstanding'

const STATUS_TABS: { value: StatusFilter; label: string; dot?: string }[] = [
  { value: 'all', label: 'Semua' },
  { value: 'outstanding', label: 'Outstanding', dot: 'bg-amber-400' },
  { value: 'overdue', label: 'Overdue', dot: 'bg-red-500' },
  { value: 'paid', label: 'Lunas', dot: 'bg-emerald-500' },
  { value: 'draft', label: 'Draft', dot: 'bg-ink-300' },
  { value: 'cancelled', label: 'Dibatalkan' },
]

const TYPE_OPTIONS: { value: InvoiceType | 'all'; label: string }[] = [
  { value: 'all', label: 'Semua Tipe' },
  { value: 'registration', label: 'Pendaftaran' },
  { value: 'renewal', label: 'Renewal' },
]

/** Umur tunggakan (AR aging) — hanya relevan untuk invoice yang belum dibayar. */
type Aging = 'all' | '1-30' | '31-60' | '60+'

const AGING_OPTIONS: { value: Aging; label: string }[] = [
  { value: 'all', label: 'Semua Umur' },
  { value: '1-30', label: 'Telat 1–30 hari' },
  { value: '31-60', label: 'Telat 31–60 hari' },
  { value: '60+', label: 'Telat > 60 hari' },
]

export function InvoiceListPage() {
  const navigate = useNavigate()
  const { toast } = useToast()
  const [searchParams] = useSearchParams()
  const initialStatus = (searchParams.get('status') as StatusFilter | null) ?? 'all'
  const initialType = (searchParams.get('type') as InvoiceType | null) ?? 'all'
  const initialChapter = searchParams.get('chapter') ?? 'all'

  const { data: invoices, loading, error, reload } = useAsync<InvoiceWithRelations[]>(() =>
    invoiceService.list(),
  )
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())
  const { data: selfPayment } = useAsync<boolean>(() => isSelfPaymentMode())

  const canCreate = useCan('invoice:create')
  const canManage = useCan('invoice:manage')

  const [status, setStatus] = useState<StatusFilter>(initialStatus)
  const [type, setType] = useState<InvoiceType | 'all'>(initialType)
  const [chapterId, setChapterId] = useState<string>(initialChapter)
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [dueFrom, setDueFrom] = useState('')
  const [dueTo, setDueTo] = useState('')
  const [aging, setAging] = useState<Aging>('all')
  const [issuedFrom, setIssuedFrom] = useState('')
  const [issuedTo, setIssuedTo] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [bulkSending, setBulkSending] = useState(false)
  const [page, setPage] = useState(1)
  const PAGE_SIZE = 25

  // Everything except the status filter. Drives the summary cards and the
  // status-tab counts, so they reflect type/chapter/search/due-date while still
  // showing the full per-status breakdown the user is choosing between.
  const baseFiltered = useMemo(() => {
    if (!invoices) return []
    const q = search.trim().toLowerCase()
    return invoices.filter((inv) => {
      if (type !== 'all' && inv.type !== type) return false
      if (chapterId !== 'all' && inv.chapterId !== chapterId) return false
      if (q && !inv.number.toLowerCase().includes(q) && !(inv.member?.name ?? '').toLowerCase().includes(q))
        return false
      // Missing due date is out-of-range for BOTH bounds (symmetric).
      if (dueFrom && (!inv.dueDate || inv.dueDate < dueFrom)) return false
      if (dueTo && (!inv.dueDate || inv.dueDate > dueTo)) return false
      // Tanggal terbit (createdAt) — sumbu waktu yang dipakai Laporan.
      const issued = inv.createdAt ? inv.createdAt.slice(0, 10) : ''
      if (issuedFrom && (!issued || issued < issuedFrom)) return false
      if (issuedTo && (!issued || issued > issuedTo)) return false
      // Umur tunggakan: hanya invoice belum dibayar yang punya "umur".
      if (aging !== 'all') {
        if (!isOutstanding(inv.status) || !inv.dueDate) return false
        const late = -daysUntil(inv.dueDate) // positif = jumlah hari telat
        if (late < 1) return false
        if (aging === '1-30' && late > 30) return false
        if (aging === '31-60' && (late < 31 || late > 60)) return false
        if (aging === '60+' && late <= 60) return false
      }
      return true
    })
  }, [invoices, type, chapterId, search, dueFrom, dueTo, issuedFrom, issuedTo, aging])

  const countByStatus = useMemo(() => {
    return baseFiltered.reduce<Record<string, number>>((acc, inv) => {
      acc[inv.status] = (acc[inv.status] ?? 0) + 1
      return acc
    }, {})
  }, [baseFiltered])

  const summary = useMemo(() => {
    const list = baseFiltered
    const amt = (pred: (i: InvoiceWithRelations) => boolean) =>
      list.filter(pred).reduce((a, i) => a + i.amount, 0)
    return {
      // Exclude cancelled from the headline total (matches the dashboard — a
      // voided invoice shouldn't inflate "total billed").
      total: {
        count: list.filter((i) => i.status !== 'cancelled').length,
        amount: amt((i) => i.status !== 'cancelled'),
      },
      outstanding: {
        count: list.filter((i) => isOutstanding(i.status)).length,
        amount: amt((i) => isOutstanding(i.status)),
      },
      overdue: { count: countByStatus.overdue ?? 0, amount: amt((i) => i.status === 'overdue') },
      paid: { count: countByStatus.paid ?? 0, amount: amt((i) => i.status === 'paid') },
    }
  }, [baseFiltered, countByStatus])

  const filtered = useMemo(() => {
    return baseFiltered.filter((inv) => {
      if (status === 'outstanding') return isOutstanding(inv.status)
      if (status !== 'all') return inv.status === status
      return true
    })
  }, [baseFiltered, status])

  // Bulk actions operate only on currently-visible (filtered) selected rows —
  // so tightening a filter can't leave hidden invoices in the batch.
  const selectedInvoices = useMemo(
    () => filtered.filter((inv) => selected.has(inv.id)),
    [filtered, selected],
  )
  const selectedSendable = useMemo(
    () => selectedInvoices.filter((inv) => inv.status === 'draft' || inv.status === 'sent' || inv.status === 'overdue'),
    [selectedInvoices],
  )
  const selectedTotal = useMemo(
    () => selectedInvoices.reduce((acc, inv) => acc + inv.amount, 0),
    [selectedInvoices],
  )

  const handleBulkSend = async () => {
    if (selectedSendable.length === 0) return
    setBulkSending(true)
    let sent = 0
    try {
      for (const inv of selectedSendable) {
        if (inv.status === 'draft') {
          await invoiceService.send(inv.id)
        } else {
          await invoiceService.resend(inv.id)
        }
        sent++
      }
      toast(`${sent} invoice berhasil dikirim ke Paper.id.`)
      setSelected(new Set())
      reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal mengirim invoice.', 'error')
    } finally {
      setBulkSending(false)
    }
  }

  // Self-payment (Xendit) mode: kirim link pembayaran /pay/:id ke tiap member
  // lewat Email atau WhatsApp. Draft diterbitkan dulu agar link-nya aktif.
  const bulkShare = async (channel: 'email' | 'whatsapp') => {
    if (selectedSendable.length === 0) return
    setBulkSending(true)
    try {
      for (const inv of selectedSendable) {
        if (inv.status === 'draft') await invoiceService.send(inv.id)
      }
      let opened = 0
      let blocked = 0
      let skipped = 0
      for (const inv of selectedSendable) {
        const payUrl = `${window.location.origin}/pay/${inv.id}`
        const name = inv.member?.name ?? 'Bapak/Ibu'
        if (channel === 'whatsapp') {
          const phone = normalizePhone(inv.member?.phone)
          if (!phone) {
            skipped++
            continue
          }
          const msg = `Halo ${name}, berikut tagihan BNI Anda *${inv.number}* sebesar *${formatCurrency(inv.amount)}*. Silakan lakukan pembayaran melalui tautan berikut: ${payUrl}`
          // window.open after an await loses user-activation, so the browser
          // blocks all but the first tab — detect it (null return) and report.
          const w = window.open(`https://wa.me/${phone}?text=${encodeURIComponent(msg)}`, '_blank', 'noopener,noreferrer')
          if (w) opened++
          else blocked++
        } else {
          const email = inv.member?.email
          if (!email) {
            skipped++
            continue
          }
          const subject = `Tagihan BNI ${inv.number}`
          const body = `Halo ${name},\n\nBerikut tagihan BNI Anda ${inv.number} sebesar ${formatCurrency(inv.amount)}.\nSilakan lakukan pembayaran melalui tautan berikut:\n${payUrl}\n\nTerima kasih.`
          // email is DB-sourced (BNI VM) — encode it so it can't inject extra
          // mailto fields (?cc=/&bcc=). mailto success can't be reliably detected.
          window.open(
            `mailto:${encodeURIComponent(email)}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`,
            '_blank',
            'noopener,noreferrer',
          )
          opened++
        }
      }
      const ch = channel === 'whatsapp' ? 'WhatsApp' : 'email'
      const lack = channel === 'whatsapp' ? 'no. HP' : 'email'
      let msg = `${opened} ${ch} disiapkan`
      if (skipped) msg += `, ${skipped} dilewati (tanpa ${lack})`
      if (blocked) msg += `, ${blocked} diblokir popup (izinkan popup lalu ulangi)`
      toast(msg + '.', blocked ? 'error' : undefined)
      setSelected(new Set())
      reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyiapkan pengiriman.', 'error')
    } finally {
      setBulkSending(false)
    }
  }

  // Reset ke halaman 1 setiap kali filter berubah
  useEffect(() => {
    setPage(1)
  }, [filtered])

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const paginated = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  const typeLabel = (t: InvoiceType) => (t === 'registration' ? 'Pendaftaran' : 'Renewal')

  const EXPORT_HEADERS = ['No. Invoice', 'Member', 'Chapter', 'Tipe', 'Nominal', 'Status', 'Jatuh Tempo']
  // Raw rows (amount stays numeric so Excel can sum it) — always the FILTERED set.
  const exportRows = () =>
    filtered.map((inv) => [
      inv.number,
      inv.member?.name ?? '',
      inv.chapter?.displayName ?? '',
      typeLabel(inv.type),
      inv.amount,
      INVOICE_STATUS_LABEL[inv.status],
      formatDate(inv.dueDate),
    ])

  const exportCsv = () => downloadCsv('invoice.csv', EXPORT_HEADERS, exportRows())
  const exportExcel = () => downloadXlsx('invoice', 'Daftar Invoice', EXPORT_HEADERS, exportRows())

  const exportPdf = () => {
    const total = filtered.reduce((a, i) => a + i.amount, 0)
    const ok = printTableReport({
      title: 'Daftar Invoice',
      subtitle: `Status: ${STATUS_TABS.find((t) => t.value === status)?.label ?? 'Semua'}`,
      meta: [`${filtered.length} invoice`, `Dibuat ${formatDateTime(new Date())}`],
      columns: [
        { label: 'No. Invoice' },
        { label: 'Member' },
        { label: 'Chapter' },
        { label: 'Tipe' },
        { label: 'Nominal', align: 'right' },
        { label: 'Status' },
        { label: 'Jatuh Tempo' },
      ],
      rows: filtered.map((inv) => [
        inv.number,
        inv.member?.name ?? '—',
        inv.chapter?.displayName ?? '—',
        typeLabel(inv.type),
        formatCurrency(inv.amount),
        INVOICE_STATUS_LABEL[inv.status],
        formatDate(inv.dueDate),
      ]),
      totals: ['', '', '', 'Total', formatCurrency(total), '', ''],
      documentTitle: 'Daftar Invoice — BNI Finance',
    })
    if (!ok) toast('Izinkan popup di browser untuk mengekspor PDF.', 'error')
  }

  return (
    <div>
      <PageHeader
        title="Invoice"
        description="Kelola seluruh invoice pendaftaran dan renewal."
        action={
          <div className="flex items-center gap-2">
            <ExportMenu onExcel={exportExcel} onCsv={exportCsv} onPdf={exportPdf} disabled={filtered.length === 0} />
            {canCreate && (
              <Button onClick={() => navigate('/invoices/new')}>
                <Plus className="h-4 w-4" />
                Buat Invoice
              </Button>
            )}
          </div>
        }
      />

      {/* Summary cards (also filter the table by status) */}
      <div className="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <SummaryCard
          label="Total Invoice"
          value={summary.total.count}
          sub={formatCurrencyCompact(summary.total.amount)}
          tone="brand"
          active={status === 'all'}
          onClick={() => setStatus('all')}
        />
        <SummaryCard
          label="Outstanding"
          value={summary.outstanding.count}
          sub={formatCurrencyCompact(summary.outstanding.amount)}
          tone="amber"
          active={status === 'outstanding'}
          onClick={() => setStatus('outstanding')}
        />
        <SummaryCard
          label="Overdue"
          value={summary.overdue.count}
          sub={formatCurrencyCompact(summary.overdue.amount)}
          tone="red"
          active={status === 'overdue'}
          onClick={() => setStatus('overdue')}
        />
        <SummaryCard
          label="Lunas"
          value={summary.paid.count}
          sub={formatCurrencyCompact(summary.paid.amount)}
          tone="green"
          active={status === 'paid'}
          onClick={() => setStatus('paid')}
        />
      </div>

      <Card>
        {/* Status tab pills */}
        <div className="flex gap-1 overflow-x-auto border-b border-ink-100 px-4 pt-3 pb-0">
          {STATUS_TABS.map((tab) => {
            const count =
              tab.value === 'all'
                ? baseFiltered.length
                : tab.value === 'outstanding'
                  ? (countByStatus.sent ?? 0) + (countByStatus.overdue ?? 0)
                  : countByStatus[tab.value]
            return (
              <button
                key={tab.value}
                onClick={() => setStatus(tab.value)}
                className={cn(
                  'flex shrink-0 items-center gap-1.5 rounded-t-lg border-b-2 px-3 pb-2.5 pt-2 text-[13px] font-medium transition-colors',
                  status === tab.value
                    ? 'border-brand-500 text-brand-600'
                    : 'border-transparent text-ink-500 hover:text-ink-800',
                )}
              >
                {tab.dot && (
                  <span className={cn('h-2 w-2 rounded-full', tab.dot)} />
                )}
                {tab.label}
                {count !== undefined && count > 0 && (
                  <span
                    className={cn(
                      'rounded-full px-1.5 py-0.5 text-[11px] font-semibold leading-none',
                      status === tab.value
                        ? 'bg-brand-100 text-brand-600'
                        : 'bg-ink-100 text-ink-500',
                    )}
                  >
                    {count}
                  </span>
                )}
              </button>
            )
          })}
        </div>

        {/* Filter bar */}
        <div className="space-y-3 border-b border-ink-100 p-4">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari nomor invoice atau nama member…"
              className="pl-10"
            />
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Select value={type} onChange={(e) => setType(e.target.value as InvoiceType | 'all')} className="w-full sm:w-40">
              {TYPE_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
            <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)} className="w-full sm:w-44">
              <option value="all">Semua Chapter</option>
              {chapters?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.displayName}
                </option>
              ))}
            </Select>
            {/* Umur tunggakan (aging) */}
            <Select
              value={aging}
              onChange={(e) => setAging(e.target.value as Aging)}
              className="w-full sm:w-44"
              aria-label="Umur tunggakan"
            >
              {AGING_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
          </div>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-3">
            {/* Filter jatuh tempo (rentang tanggal) */}
            <div className="flex items-center gap-2">
              <span className="text-[13px] text-ink-500">Jatuh tempo</span>
              <Input
                type="date"
                value={dueFrom}
                max={dueTo || undefined}
                onChange={(e) => setDueFrom(e.target.value)}
                className="w-[150px]"
                aria-label="Jatuh tempo dari"
              />
              <span className="text-ink-400">–</span>
              <Input
                type="date"
                value={dueTo}
                min={dueFrom || undefined}
                onChange={(e) => setDueTo(e.target.value)}
                className="w-[150px]"
                aria-label="Jatuh tempo sampai"
              />
              {(dueFrom || dueTo) && (
                <button
                  type="button"
                  onClick={() => {
                    setDueFrom('')
                    setDueTo('')
                  }}
                  className="rounded-lg p-1.5 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-700"
                  aria-label="Reset filter jatuh tempo"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>

            {/* Filter tanggal terbit (createdAt) — sumbu waktu Laporan */}
            <div className="flex items-center gap-2">
              <span className="text-[13px] text-ink-500">Tanggal terbit</span>
              <Input
                type="date"
                value={issuedFrom}
                max={issuedTo || undefined}
                onChange={(e) => setIssuedFrom(e.target.value)}
                className="w-[150px]"
                aria-label="Tanggal terbit dari"
              />
              <span className="text-ink-400">–</span>
              <Input
                type="date"
                value={issuedTo}
                min={issuedFrom || undefined}
                onChange={(e) => setIssuedTo(e.target.value)}
                className="w-[150px]"
                aria-label="Tanggal terbit sampai"
              />
              {(issuedFrom || issuedTo) && (
                <button
                  type="button"
                  onClick={() => {
                    setIssuedFrom('')
                    setIssuedTo('')
                  }}
                  className="rounded-lg p-1.5 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-700"
                  aria-label="Reset filter tanggal terbit"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>
        </div>

        {/* Bulk action bar */}
        {canManage && selected.size > 0 && (
          <div className="flex flex-col gap-3 border-b border-ink-100 bg-brand-50/50 px-5 py-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="text-sm font-medium text-ink-700">
              {selected.size} dipilih
              {selectedTotal > 0 && ` · ${formatCurrency(selectedTotal)}`}
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>
                <X className="h-4 w-4" />
                Batal
              </Button>
              {selectedSendable.length > 0 &&
                (selfPayment ? (
                  <>
                    <Button variant="outline" size="sm" loading={bulkSending} onClick={() => bulkShare('email')}>
                      <Mail className="h-4 w-4" />
                      Kirim {selectedSendable.length} via Email
                    </Button>
                    <Button size="sm" loading={bulkSending} onClick={() => bulkShare('whatsapp')}>
                      <WhatsAppIcon className="h-4 w-4" />
                      Kirim {selectedSendable.length} via WhatsApp
                    </Button>
                  </>
                ) : (
                  <Button size="sm" loading={bulkSending} onClick={handleBulkSend}>
                    <Send className="h-4 w-4" />
                    Kirim {selectedSendable.length} ke Paper.id
                  </Button>
                ))}
            </div>
          </div>
        )}

        {error ? (
          <ErrorState message={error} onRetry={reload} />
        ) : loading ? (
          <TableSkeleton rows={8} cols={7} />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={FileText}
            title="Tidak ada invoice"
            description="Tidak ada invoice yang cocok dengan filter saat ini. Coba ubah filter atau buat invoice baru."
            action={
              <Button variant="outline" onClick={() => navigate('/invoices/new')}>
                <Plus className="h-4 w-4" />
                Buat Invoice
              </Button>
            }
          />
        ) : (
          <>
            <InvoiceTable
              invoices={paginated}
              selected={canManage ? selected : undefined}
              onSelectChange={canManage ? setSelected : undefined}
            />
            <div className="flex items-center justify-between gap-4 border-t border-ink-100 px-5 py-3">
              <span className="text-xs text-ink-400">
                {filtered.length === 0
                  ? 'Tidak ada invoice'
                  : `${(page - 1) * PAGE_SIZE + 1}–${Math.min(page * PAGE_SIZE, filtered.length)} dari ${filtered.length} invoice`}
              </span>
              {totalPages > 1 && (
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                    className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-ink-500 transition-colors hover:bg-ink-100 disabled:opacity-30 disabled:cursor-not-allowed"
                    aria-label="Halaman sebelumnya"
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </button>
                  {Array.from({ length: totalPages }, (_, i) => i + 1)
                    .filter((p) => p === 1 || p === totalPages || Math.abs(p - page) <= 1)
                    .reduce<(number | 'ellipsis')[]>((acc, p, idx, arr) => {
                      if (idx > 0 && p - (arr[idx - 1] as number) > 1) acc.push('ellipsis')
                      acc.push(p)
                      return acc
                    }, [])
                    .map((item, idx) =>
                      item === 'ellipsis' ? (
                        <span key={`e${idx}`} className="px-1 text-xs text-ink-400">…</span>
                      ) : (
                        <button
                          key={item}
                          onClick={() => setPage(item as number)}
                          className={`inline-flex h-8 min-w-[2rem] items-center justify-center rounded-lg px-2 text-xs font-medium transition-colors ${
                            page === item
                              ? 'bg-brand-500 text-white'
                              : 'text-ink-600 hover:bg-ink-100'
                          }`}
                        >
                          {item}
                        </button>
                      ),
                    )}
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages}
                    className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-ink-500 transition-colors hover:bg-ink-100 disabled:opacity-30 disabled:cursor-not-allowed"
                    aria-label="Halaman berikutnya"
                  >
                    <ChevronRight className="h-4 w-4" />
                  </button>
                </div>
              )}
            </div>
          </>
        )}
      </Card>
    </div>
  )
}
