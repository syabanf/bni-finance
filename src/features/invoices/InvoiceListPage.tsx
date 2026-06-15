import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Download, FileText, Plus, Search, Send, X } from 'lucide-react'
import type { Chapter, InvoiceStatus, InvoiceType, InvoiceWithRelations } from '@/types'
import {
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  Select,
  TableSkeleton,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, invoiceService } from '@/services'
import { InvoiceTable } from './components/InvoiceTable'
import { cn } from '@/lib/cn'
import { formatCurrency, formatDate } from '@/lib/format'

function downloadCsv(filename: string, headers: string[], rows: string[][]) {
  const escape = (v: string) => `"${v.replace(/"/g, '""')}"`
  const lines = [headers, ...rows].map((r) => r.map(escape).join(','))
  const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  URL.revokeObjectURL(url)
}

const STATUS_TABS: { value: InvoiceStatus | 'all'; label: string; dot?: string }[] = [
  { value: 'all', label: 'Semua' },
  { value: 'overdue', label: 'Overdue', dot: 'bg-red-500' },
  { value: 'sent', label: 'Outstanding', dot: 'bg-amber-400' },
  { value: 'draft', label: 'Draft', dot: 'bg-ink-300' },
  { value: 'paid', label: 'Lunas', dot: 'bg-emerald-500' },
  { value: 'cancelled', label: 'Dibatalkan' },
]

const TYPE_OPTIONS: { value: InvoiceType | 'all'; label: string }[] = [
  { value: 'all', label: 'Semua Tipe' },
  { value: 'registration', label: 'Pendaftaran' },
  { value: 'renewal', label: 'Renewal' },
]

export function InvoiceListPage() {
  const navigate = useNavigate()
  const { toast } = useToast()
  const [searchParams] = useSearchParams()
  const initialStatus = (searchParams.get('status') as InvoiceStatus | null) ?? 'all'

  const { data: invoices, loading, reload } = useAsync<InvoiceWithRelations[]>(() => invoiceService.list())
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())

  const [status, setStatus] = useState<InvoiceStatus | 'all'>(initialStatus)
  const [type, setType] = useState<InvoiceType | 'all'>('all')
  const [chapterId, setChapterId] = useState<string>('all')
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [bulkSending, setBulkSending] = useState(false)

  const countByStatus = useMemo(() => {
    if (!invoices) return {} as Record<string, number>
    return invoices.reduce<Record<string, number>>((acc, inv) => {
      acc[inv.status] = (acc[inv.status] ?? 0) + 1
      return acc
    }, {})
  }, [invoices])

  const selectedInvoices = useMemo(
    () => (invoices ?? []).filter((inv) => selected.has(inv.id)),
    [invoices, selected],
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

  const filtered = useMemo(() => {
    if (!invoices) return []
    const q = search.trim().toLowerCase()
    return invoices.filter((inv) => {
      if (status !== 'all' && inv.status !== status) return false
      if (type !== 'all' && inv.type !== type) return false
      if (chapterId !== 'all' && inv.chapterId !== chapterId) return false
      if (q && !inv.number.toLowerCase().includes(q) && !(inv.member?.name ?? '').toLowerCase().includes(q))
        return false
      return true
    })
  }, [invoices, status, type, chapterId, search])

  return (
    <div>
      <PageHeader
        title="Invoice"
        description="Kelola seluruh invoice pendaftaran dan renewal."
        action={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={() =>
                downloadCsv(
                  'invoice.csv',
                  ['No. Invoice', 'Member', 'Chapter', 'Tipe', 'Nominal', 'Status', 'Jatuh Tempo'],
                  (filtered).map((inv) => [
                    inv.number,
                    inv.member?.name ?? '',
                    inv.chapter?.displayName ?? '',
                    inv.type,
                    String(inv.amount),
                    inv.status,
                    formatDate(inv.dueDate),
                  ]),
                )
              }
            >
              <Download className="h-4 w-4" />
              Export CSV
            </Button>
            <Button onClick={() => navigate('/invoices/new')}>
              <Plus className="h-4 w-4" />
              Buat Invoice
            </Button>
          </div>
        }
      />

      <Card>
        {/* Status tab pills */}
        <div className="flex gap-1 overflow-x-auto border-b border-ink-100 px-4 pt-3 pb-0">
          {STATUS_TABS.map((tab) => {
            const count = tab.value === 'all' ? invoices?.length : countByStatus[tab.value]
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
        <div className="flex flex-col gap-3 border-b border-ink-100 p-4 lg:flex-row lg:items-center">
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari nomor invoice atau nama member…"
              className="pl-10"
            />
          </div>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-2 lg:flex">
            <Select value={type} onChange={(e) => setType(e.target.value as InvoiceType | 'all')} className="lg:w-40">
              {TYPE_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
            <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)} className="lg:w-44">
              <option value="all">Semua Chapter</option>
              {chapters?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.displayName}
                </option>
              ))}
            </Select>
          </div>
        </div>

        {/* Bulk action bar */}
        {selected.size > 0 && (
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
              {selectedSendable.length > 0 && (
                <Button size="sm" loading={bulkSending} onClick={handleBulkSend}>
                  <Send className="h-4 w-4" />
                  Kirim {selectedSendable.length} ke Paper.id
                </Button>
              )}
            </div>
          </div>
        )}

        {loading ? (
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
              invoices={filtered}
              selected={selected}
              onSelectChange={setSelected}
            />
            <div className="px-5 py-3 text-xs text-ink-400">
              Menampilkan {filtered.length} dari {invoices?.length ?? 0} invoice
            </div>
          </>
        )}
      </Card>
    </div>
  )
}
