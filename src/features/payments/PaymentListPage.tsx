import { useNavigate } from 'react-router-dom'
import { ArrowRight, Wallet } from 'lucide-react'
import type { PaymentWithInvoice } from '@/types'
import {
  Avatar,
  Badge,
  Card,
  EmptyState,
  ExportMenu,
  PageHeader,
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

  const list = payments ?? []
  const total = list.reduce((acc, p) => acc + p.amount, 0)
  const ym = new Date().toISOString().slice(0, 7)
  const thisMonth = list.filter((p) => (p.paidAt ?? '').slice(0, 7) === ym)
  const thisMonthTotal = thisMonth.reduce((acc, p) => acc + p.amount, 0)

  const methodLabel = (p: PaymentWithInvoice) =>
    METHOD_LABEL[p.paymentMethod ?? ''] ?? p.paymentMethod ?? '—'

  const exportCsv = () => {
    downloadCsv(
      'pembayaran.csv',
      ['Member', 'No. Invoice', 'Nominal', 'Metode', 'Waktu Bayar'],
      list.map((p) => [
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
      meta: [`${list.length} transaksi`, `Dibuat ${formatDateTime(new Date())}`],
      columns: [
        { label: 'Member' },
        { label: 'No. Invoice' },
        { label: 'Nominal', align: 'right' },
        { label: 'Metode' },
        { label: 'Waktu Bayar' },
      ],
      rows: list.map((p) => [
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

  return (
    <div>
      <PageHeader
        title="Pembayaran"
        description="Riwayat pembayaran yang diterima melalui webhook Paper.id."
        action={<ExportMenu onCsv={exportCsv} onPdf={exportPdf} disabled={list.length === 0} />}
      />

      {!loading && payments && payments.length > 0 && (
        <div className="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-3">
          <SummaryCard label="Total Diterima" value={formatCurrency(total)} tone="green" />
          <SummaryCard label="Jumlah Transaksi" value={payments.length} tone="brand" />
          <SummaryCard
            label="Bulan Ini"
            value={formatCurrency(thisMonthTotal)}
            sub={`${thisMonth.length} transaksi`}
          />
        </div>
      )}

      <Card>
        {loading ? (
          <TableSkeleton rows={8} cols={5} />
        ) : !payments || payments.length === 0 ? (
          <EmptyState icon={Wallet} title="Belum ada pembayaran" description="Pembayaran akan muncul di sini setelah diterima dari Paper.id." />
        ) : (
          <>
            {/* Mobile card list */}
            <div className="divide-y divide-ink-100 lg:hidden">
              {payments.map((p) => (
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
                        {METHOD_LABEL[p.paymentMethod ?? ''] ?? p.paymentMethod ?? '—'}
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
                  {payments.map((p) => (
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
                          {METHOD_LABEL[p.paymentMethod ?? ''] ?? p.paymentMethod ?? '—'}
                        </Badge>
                      </Td>
                      <Td className="whitespace-nowrap text-ink-600">{formatDateTime(p.paidAt)}</Td>
                    </Tr>
                  ))}
                </TBody>
              </Table>
            </div>
          </>
        )}
      </Card>
    </div>
  )
}
