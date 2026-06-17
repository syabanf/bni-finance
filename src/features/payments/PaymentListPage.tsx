import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowRight, Search, Wallet, X } from 'lucide-react'
import type { PaymentWithInvoice } from '@/types'
import {
  Avatar,
  Badge,
  Card,
  EmptyState,
  ExportMenu,
  Input,
  PageHeader,
  Select,
  SummaryCard,
  Table,
  TBody,
  Td,
  Th,
  THead,
  Tr,
  TableSkeleton,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { paymentService } from '@/services'
import { formatCurrency, formatDateTime } from '@/lib/format'
import { downloadCsv } from '@/lib/csv'
import { printTableReport } from '@/lib/pdfReport'

const METHOD_LABEL: Record<string, string> = {
  virtual_account: 'Virtual Account',
  bank_transfer: 'Transfer Bank',
  qris: 'QRIS',
}

export function PaymentListPage() {
  const navigate = useNavigate()
  const { data: payments, loading } = useAsync<PaymentWithInvoice[]>(() => paymentService.list())

  const [search, setSearch] = useState('')
  const [method, setMethod] = useState('all')
  const [dueFrom, setDueFrom] = useState('')
  const [dueTo, setDueTo] = useState('')

  const methodOptions = useMemo(() => {
    const set = new Set<string>()
    ;(payments ?? []).forEach((p) => {
      if (p.paymentMethod) set.add(p.paymentMethod)
    })
    return Array.from(set).sort()
  }, [payments])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return (payments ?? []).filter((p) => {
      if (method !== 'all' && (p.paymentMethod ?? '') !== method) return false
      const day = (p.paidAt ?? '').slice(0, 10)
      if (dueFrom && day < dueFrom) return false
      if (dueTo && day > dueTo) return false
      if (
        q &&
        !(p.member?.name ?? '').toLowerCase().includes(q) &&
        !(p.invoice?.number ?? '').toLowerCase().includes(q)
      )
        return false
      return true
    })
  }, [payments, search, method, dueFrom, dueTo])

  // Summary reflects the active filters.
  const total = filtered.reduce((acc, p) => acc + p.amount, 0)
  const ym = new Date().toISOString().slice(0, 7)
  const thisMonth = filtered.filter((p) => (p.paidAt ?? '').slice(0, 7) === ym)
  const thisMonthTotal = thisMonth.reduce((acc, p) => acc + p.amount, 0)

  const methodLabel = (p: PaymentWithInvoice) =>
    METHOD_LABEL[p.paymentMethod ?? ''] ?? p.paymentMethod ?? '—'

  const exportCsv = () => {
    downloadCsv(
      'pembayaran.csv',
      ['Member', 'No. Invoice', 'Nominal', 'Metode', 'Waktu Bayar'],
      filtered.map((p) => [
        p.member?.name ?? '',
        p.invoice?.number ?? '',
        p.amount,
        methodLabel(p),
        formatDateTime(p.paidAt),
      ]),
    )
  }

  const exportPdf = () => {
    printTableReport({
      title: 'Riwayat Pembayaran',
      meta: [`${filtered.length} transaksi`, `Dibuat ${formatDateTime(new Date())}`],
      columns: [
        { label: 'Member' },
        { label: 'No. Invoice' },
        { label: 'Nominal', align: 'right' },
        { label: 'Metode' },
        { label: 'Waktu Bayar' },
      ],
      rows: filtered.map((p) => [
        p.member?.name ?? '—',
        p.invoice?.number ?? '—',
        formatCurrency(p.amount),
        methodLabel(p),
        formatDateTime(p.paidAt),
      ]),
      totals: ['Total', '', formatCurrency(total), '', ''],
      documentTitle: 'Riwayat Pembayaran — BNI Finance',
    })
  }

  const hasData = !loading && payments && payments.length > 0

  return (
    <div>
      <PageHeader
        title="Pembayaran"
        description="Riwayat pembayaran yang diterima melalui webhook Paper.id."
        action={<ExportMenu onCsv={exportCsv} onPdf={exportPdf} disabled={filtered.length === 0} />}
      />

      {hasData && (
        <div className="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-3">
          <SummaryCard label="Total Diterima" value={formatCurrency(total)} tone="green" />
          <SummaryCard label="Jumlah Transaksi" value={filtered.length} tone="brand" />
          <SummaryCard
            label="Bulan Ini"
            value={formatCurrency(thisMonthTotal)}
            sub={`${thisMonth.length} transaksi`}
          />
        </div>
      )}

      <Card>
        {/* Filter bar */}
        {hasData && (
          <div className="space-y-3 border-b border-ink-100 p-4">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Cari nama member atau nomor invoice…"
                className="pl-10"
              />
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Select value={method} onChange={(e) => setMethod(e.target.value)} className="w-full sm:w-48">
                <option value="all">Semua Metode</option>
                {methodOptions.map((m) => (
                  <option key={m} value={m}>
                    {METHOD_LABEL[m] ?? m}
                  </option>
                ))}
              </Select>
              {/* Filter waktu bayar (rentang tanggal) */}
              <div className="flex items-center gap-2">
                <span className="text-[13px] text-ink-500">Waktu bayar</span>
                <Input
                  type="date"
                  value={dueFrom}
                  max={dueTo || undefined}
                  onChange={(e) => setDueFrom(e.target.value)}
                  className="w-[150px]"
                  aria-label="Waktu bayar dari"
                />
                <span className="text-ink-400">–</span>
                <Input
                  type="date"
                  value={dueTo}
                  min={dueFrom || undefined}
                  onChange={(e) => setDueTo(e.target.value)}
                  className="w-[150px]"
                  aria-label="Waktu bayar sampai"
                />
                {(dueFrom || dueTo) && (
                  <button
                    type="button"
                    onClick={() => {
                      setDueFrom('')
                      setDueTo('')
                    }}
                    className="rounded-lg p-1.5 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-700"
                    aria-label="Reset filter waktu bayar"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
          </div>
        )}

        {loading ? (
          <TableSkeleton rows={8} cols={5} />
        ) : !payments || payments.length === 0 ? (
          <EmptyState
            icon={Wallet}
            title="Belum ada pembayaran"
            description="Pembayaran akan muncul di sini setelah diterima dari Paper.id."
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={Wallet}
            title="Tidak ada hasil"
            description="Tidak ada pembayaran yang cocok dengan filter saat ini."
          />
        ) : (
          <>
            {/* Mobile card list */}
            <div className="divide-y divide-ink-100 lg:hidden">
              {filtered.map((p) => (
                <div
                  key={p.id}
                  onClick={() => p.invoice && navigate(`/invoices/${p.invoice.id}`)}
                  className="flex items-center gap-3 px-4 py-3.5 active:bg-ink-50"
                >
                  <Avatar name={p.member?.name ?? '?'} size="sm" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="truncate font-medium text-ink-900 text-sm">{p.member?.name ?? '—'}</div>
                        <div className="font-mono text-xs text-ink-400">{p.invoice?.number ?? '—'}</div>
                      </div>
                      <div className="shrink-0 text-right">
                        <div className="font-semibold text-emerald-600 text-sm">{formatCurrency(p.amount)}</div>
                        <div className="text-xs text-ink-400 mt-0.5">{formatDateTime(p.paidAt)}</div>
                      </div>
                    </div>
                    <div className="mt-2">
                      <Badge tone="blue" dot={false}>
                        {methodLabel(p)}
                      </Badge>
                    </div>
                  </div>
                  <ArrowRight className="h-4 w-4 shrink-0 text-ink-300" />
                </div>
              ))}
            </div>

            {/* Desktop table */}
            <div className="hidden lg:block">
              <Table>
                <THead>
                  <Tr>
                    <Th>Member</Th>
                    <Th>No. Invoice</Th>
                    <Th>Nominal</Th>
                    <Th>Metode</Th>
                    <Th>Waktu Bayar</Th>
                  </Tr>
                </THead>
                <TBody>
                  {filtered.map((p) => (
                    <Tr key={p.id} onClick={() => p.invoice && navigate(`/invoices/${p.invoice.id}`)}>
                      <Td>
                        <div className="flex items-center gap-3">
                          <Avatar name={p.member?.name ?? '?'} size="sm" />
                          <span className="font-medium text-ink-900">{p.member?.name ?? '—'}</span>
                        </div>
                      </Td>
                      <Td>
                        <span className="font-mono text-[13px] text-ink-600">{p.invoice?.number ?? '—'}</span>
                      </Td>
                      <Td className="font-medium text-emerald-600">{formatCurrency(p.amount)}</Td>
                      <Td>
                        <Badge tone="blue" dot={false}>
                          {methodLabel(p)}
                        </Badge>
                      </Td>
                      <Td className="whitespace-nowrap text-ink-600">{formatDateTime(p.paidAt)}</Td>
                    </Tr>
                  ))}
                </TBody>
              </Table>
            </div>
            <div className="px-5 py-3 text-xs text-ink-400">
              Menampilkan {filtered.length} dari {payments.length} pembayaran
            </div>
          </>
        )}
      </Card>
    </div>
  )
}
