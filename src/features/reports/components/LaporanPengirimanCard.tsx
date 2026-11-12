import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Send } from 'lucide-react'
import type { InvoiceWithRelations } from '@/types'
import { Card, CardHeader, EmptyState, ExportMenu, InvoiceStatusBadge, TableSkeleton } from '@/components/ui'
import { formatCurrency, formatDate } from '@/lib/format'
import { INVOICE_STATUS_LABEL } from '@/lib/status'
import { downloadXlsx } from '@/lib/xlsx'

/**
 * Invoice yang dikirim ke Paper.id pada periode laporan, dihitung dari tanggal
 * kirim. Paper.id hanya memberi kabar saat invoice lunas, jadi laporan ini
 * menyebut kapan invoice dikirim dan diingatkan, lalu apakah sudah dibayar.
 */
export function LaporanPengirimanCard({
  invoices,
  inRange,
  loading,
  namaBerkas,
}: {
  invoices: InvoiceWithRelations[]
  inRange: (iso?: string) => boolean
  loading: boolean
  namaBerkas: string
}) {
  const navigate = useNavigate()
  const baris = useMemo(
    () =>
      invoices
        .filter((i) => inRange(i.paperIdSentAt))
        .sort((a, b) => (b.paperIdSentAt ?? '').localeCompare(a.paperIdSentAt ?? '')),
    [invoices, inRange],
  )
  const lunas = baris.filter((i) => i.status === 'paid').length

  const ekspor = () =>
    downloadXlsx(
      `${namaBerkas}-pengiriman`,
      'Pengiriman',
      ['Nomor', 'Member', 'Chapter', 'Dikirim', 'Pengingat', 'Jatuh Tempo', 'Status', 'Dibayar', 'Nominal'],
      baris.map((i) => [
        i.number,
        i.member?.name ?? '',
        i.chapter?.displayName ?? '',
        (i.paperIdSentAt ?? '').slice(0, 10),
        i.paperIdReminderCount ?? 0,
        i.dueDate.slice(0, 10),
        INVOICE_STATUS_LABEL[i.status],
        (i.paidAt ?? '').slice(0, 10),
        i.amount,
      ]),
    )

  return (
    <Card className="mt-5">
      <CardHeader
        title="Laporan Pengiriman"
        subtitle={`Invoice yang dikirim ke Paper.id pada periode ini · ${lunas} dari ${baris.length} sudah dibayar.`}
        action={<ExportMenu onExcel={ekspor} disabled={baris.length === 0} />}
      />
      {loading ? (
        <TableSkeleton rows={4} cols={6} />
      ) : baris.length === 0 ? (
        <EmptyState icon={Send} title="Belum ada pengiriman" description="Tidak ada invoice yang dikirim pada periode ini." />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-y border-ink-100 text-left text-xs font-semibold uppercase tracking-wide text-ink-400">
                <th className="px-5 py-3">Nomor</th>
                <th className="px-3 py-3">Member</th>
                <th className="px-3 py-3">Dikirim</th>
                <th className="px-3 py-3 text-right">Pengingat</th>
                <th className="px-3 py-3">Status</th>
                <th className="px-3 py-3">Dibayar</th>
                <th className="px-5 py-3 text-right">Nominal</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-ink-50">
              {baris.map((i) => (
                <tr key={i.id} onClick={() => navigate(`/invoices/${i.id}`)} className="cursor-pointer hover:bg-ink-50/50">
                  <td className="whitespace-nowrap px-5 py-3 font-medium text-ink-900">{i.number}</td>
                  <td className="px-3 py-3 text-ink-700">
                    {i.member?.name ?? '—'}
                    <div className="text-xs text-ink-400">{i.chapter?.displayName ?? ''}</div>
                  </td>
                  <td className="whitespace-nowrap px-3 py-3 text-ink-600">{formatDate(i.paperIdSentAt)}</td>
                  <td className="px-3 py-3 text-right text-ink-600">{i.paperIdReminderCount ?? 0}</td>
                  <td className="px-3 py-3">
                    <InvoiceStatusBadge status={i.status} />
                  </td>
                  <td className="whitespace-nowrap px-3 py-3 text-ink-600">{i.paidAt ? formatDate(i.paidAt) : '—'}</td>
                  <td className="whitespace-nowrap px-5 py-3 text-right text-ink-700">{formatCurrency(i.amount)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  )
}
