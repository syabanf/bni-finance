import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Download, FileText, Mail, Plus, Search, Send, X } from 'lucide-react'
import type { Chapter, InvoiceStatus, InvoiceType, InvoiceWithRelations } from '@/types'
import {
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  Select,
  SummaryCard,
  TableSkeleton,
  useToast,
  WhatsAppIcon,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, invoiceService } from '@/services'
import { isSelfPaymentMode } from '@/services/supabase/paymentGateway'
import { InvoiceTable } from './components/InvoiceTable'
import { cn } from '@/lib/cn'
import { formatCurrency, formatCurrencyCompact, formatDate } from '@/lib/format'
import { isOutstanding } from '@/lib/status'
import { normalizePhone } from '@/lib/whatsapp'
import { downloadCsv } from '@/lib/csv'

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

export function InvoiceListPage() {
  const navigate = useNavigate()
  const { toast } = useToast()
  const [searchParams] = useSearchParams()
  const initialStatus = (searchParams.get('status') as StatusFilter | null) ?? 'all'
  const initialType = (searchParams.get('type') as InvoiceType | null) ?? 'all'
  const initialChapter = searchParams.get('chapter') ?? 'all'

  const { data: invoices, loading, reload } = useAsync<InvoiceWithRelations[]>(() => invoiceService.list())
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())
  const { data: selfPayment } = useAsync<boolean>(() => isSelfPaymentMode())

  const [status, setStatus] = useState<StatusFilter>(initialStatus)
  const [type, setType] = useState<InvoiceType | 'all'>(initialType)
  const [chapterId, setChapterId] = useState<string>(initialChapter)
  const [search, setSearch] = useState('')
  const [dueFrom, setDueFrom] = useState('')
  const [dueTo, setDueTo] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [bulkSending, setBulkSending] = useState(false)

  const countByStatus = useMemo(() => {
    if (!invoices) return {} as Record<string, number>
    return invoices.reduce<Record<string, number>>((acc, inv) => {
      acc[inv.status] = (acc[inv.status] ?? 0) + 1
      return acc
    }, {})
  }, [invoices])

  const summary = useMemo(() => {
    const list = invoices ?? []
    const amt = (pred: (i: InvoiceWithRelations) => boolean) =>
      list.filter(pred).reduce((a, i) => a + i.amount, 0)
    return {
      total: { count: list.length, amount: amt(() => true) },
      outstanding: {
        count: list.filter((i) => isOutstanding(i.status)).length,
        amount: amt((i) => isOutstanding(i.status)),
      },
      overdue: { count: countByStatus.overdue ?? 0, amount: amt((i) => i.status === 'overdue') },
      paid: { count: countByStatus.paid ?? 0, amount: amt((i) => i.status === 'paid') },
    }
  }, [invoices, countByStatus])

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
          window.open(`https://wa.me/${phone}?text=${encodeURIComponent(msg)}`, '_blank', 'noopener')
        } else {
          const email = inv.member?.email
          if (!email) {
            skipped++
            continue
          }
          const subject = `Tagihan BNI ${inv.number}`
          const body = `Halo ${name},\n\nBerikut tagihan BNI Anda ${inv.number} sebesar ${formatCurrency(inv.amount)}.\nSilakan lakukan pembayaran melalui tautan berikut:\n${payUrl}\n\nTerima kasih.`
          window.open(`mailto:${email}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`, '_blank')
        }
        opened++
      }
      const ch = channel === 'whatsapp' ? 'WhatsApp' : 'email'
      const lack = channel === 'whatsapp' ? 'no. HP' : 'email'
      toast(`${opened} ${ch} disiapkan${skipped ? `, ${skipped} dilewati (tanpa ${lack})` : ''}.`)
      setSelected(new Set())
      reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyiapkan pengiriman.', 'error')
    } finally {
      setBulkSending(false)
    }
  }

  const filtered = useMemo(() => {
    if (!invoices) return []
    const q = search.trim().toLowerCase()
    return invoices.filter((inv) => {
      if (status === 'outstanding') {
        if (!isOutstanding(inv.status)) return false
      } else if (status !== 'all' && inv.status !== status) return false
      if (type !== 'all' && inv.type !== type) return false
      if (chapterId !== 'all' && inv.chapterId !== chapterId) return false
      if (q && !inv.number.toLowerCase().includes(q) && !(inv.member?.name ?? '').toLowerCase().includes(q))
        return false
      if (dueFrom && (inv.dueDate ?? '') < dueFrom) return false
      if (dueTo && (inv.dueDate ?? '') > dueTo) return false
      return true
    })
  }, [invoices, status, type, chapterId, search, dueFrom, dueTo])

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
                ? invoices?.length
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
