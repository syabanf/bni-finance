import { useNavigate } from 'react-router-dom'
import { Wallet } from 'lucide-react'
import type { PaymentWithInvoice } from '@/types'
import {
  Avatar,
  Badge,
  Card,
  CardBody,
  EmptyState,
  PageHeader,
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

const METHOD_LABEL: Record<string, string> = {
  virtual_account: 'Virtual Account',
  bank_transfer: 'Transfer Bank',
  qris: 'QRIS',
}

export function PaymentListPage() {
  const navigate = useNavigate()
  const { data: payments, loading } = useAsync<PaymentWithInvoice[]>(() => paymentService.list())

  const total = (payments ?? []).reduce((acc, p) => acc + p.amount, 0)

  return (
    <div>
      <PageHeader
        title="Pembayaran"
        description="Riwayat pembayaran yang diterima melalui webhook Paper.id."
      />

      {!loading && payments && payments.length > 0 && (
        <Card className="mb-5">
          <CardBody className="flex items-center justify-between p-5">
            <div>
              <div className="text-sm text-ink-500">Total Diterima</div>
              <div className="mt-1 text-2xl font-bold text-ink-900">{formatCurrency(total)}</div>
            </div>
            <div className="text-right">
              <div className="text-sm text-ink-500">Jumlah Transaksi</div>
              <div className="mt-1 text-2xl font-bold text-ink-900">{payments.length}</div>
            </div>
          </CardBody>
        </Card>
      )}

      <Card>
        {loading ? (
          <TableSkeleton rows={8} cols={5} />
        ) : !payments || payments.length === 0 ? (
          <EmptyState icon={Wallet} title="Belum ada pembayaran" description="Pembayaran akan muncul di sini setelah diterima dari Paper.id." />
        ) : (
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
        )}
      </Card>
    </div>
  )
}
