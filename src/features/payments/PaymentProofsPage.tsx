import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import type { ProofStatus } from '@/types'
import {
  Card,
  EmptyState,
  ErrorState,
  PageHeader,
  Pagination,
  TabBar,
  TBody,
  THead,
  Table,
  TableSkeleton,
  Td,
  Th,
  Tr,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { usePaginasi } from '@/hooks/usePaginasi'
import { useCan } from '@/features/auth/usePermission'
import { chapterService, proofService } from '@/services'
import { formatCurrency, formatDate, formatDateTime } from '@/lib/format'
import { paymentMethodLabel } from '@/lib/paymentMethod'
import { AksiBukti, BadgeBukti } from './components/AksiBukti'
import { Saringan } from './components/Saringan'

const TABS: { value: ProofStatus; label: string }[] = [
  { value: 'pending', label: 'Menunggu verifikasi' },
  { value: 'approved', label: 'Disetujui' },
  { value: 'rejected', label: 'Ditolak' },
]

/**
 * Bukti bayar yang dikirim ST ke finance (langkah 5 alur renewal klien).
 *
 * Finance memverifikasi di sini, dan persetujuannya yang menandai invoice
 * lunas. ST melihat bukti chapternya sendiri dan alasan bila ditolak; server
 * yang membatasi chapternya.
 */
export function PaymentProofsPage() {
  const bolehPeriksa = useCan('payment:record')
  const [status, setStatus] = useState<ProofStatus>('pending')
  const [cari, setCari] = useState('')
  const [chapterId, setChapterId] = useState('all')
  const daftar = useAsync(() => proofService.list(), [])
  const chapters = useAsync(() => chapterService.list().catch(() => []), [])

  // Pencarian dan chapter menyaring semua tab, jadi angka di tab ikut berubah.
  const tersaring = useMemo(() => {
    const q = cari.trim().toLowerCase()
    return (daftar.data ?? []).filter(
      (b) =>
        (chapterId === 'all' || b.chapterId === chapterId) &&
        (!q || b.invoiceNumber.toLowerCase().includes(q) || (b.memberName ?? '').toLowerCase().includes(q)),
    )
  }, [daftar.data, cari, chapterId])
  const jumlah = useMemo(() => {
    const n: Record<ProofStatus, number> = { pending: 0, approved: 0, rejected: 0 }
    for (const b of tersaring) n[b.status]++
    return n
  }, [tersaring])
  const tampil = useMemo(() => tersaring.filter((b) => b.status === status), [tersaring, status])
  const { potongan, paginasi } = usePaginasi(tampil)

  return (
    <div>
      <PageHeader
        title="Bukti Bayar"
        description={
          bolehPeriksa
            ? 'Bukti transfer yang dikirim ST. Setujui untuk mencatat pembayaran dan menandai invoice lunas.'
            : 'Bukti transfer yang Anda kirim ke finance, beserta hasil pemeriksaannya.'
        }
      />
      <Card>
        <TabBar tabs={TABS.map((t) => ({ ...t, count: jumlah[t.value] }))} value={status} onChange={setStatus} />
        <Saringan
          cari={cari}
          onCari={setCari}
          chapterId={chapterId}
          onChapter={setChapterId}
          chapters={chapters.data ?? []}
        />
        {daftar.loading ? (
          <TableSkeleton />
        ) : daftar.error ? (
          <ErrorState message={daftar.error} onRetry={daftar.reload} />
        ) : tampil.length === 0 ? (
          <EmptyState
            title={status === 'pending' ? 'Tidak ada bukti yang menunggu' : 'Belum ada bukti di sini'}
            description="ST mengirim bukti bayar dari halaman detail invoice yang sudah terkirim."
          />
        ) : (
          <Table>
            <THead>
              <Tr>
                <Th>Invoice</Th>
                <Th>Nominal</Th>
                <Th>Dibayar</Th>
                <Th>Dikirim</Th>
                <Th>Status</Th>
                <Th className="text-right">Tindakan</Th>
              </Tr>
            </THead>
            <TBody>
              {potongan.map((b) => (
                <Tr key={b.id}>
                  <Td className="whitespace-nowrap">
                    <Link to={`/invoices/${b.invoiceId}`} className="font-medium text-brand-600 hover:underline">
                      {b.invoiceNumber}
                    </Link>
                    <div className="text-xs text-ink-500">{b.memberName ?? '—'}</div>
                    <div className="text-xs text-ink-400">{b.chapterName ?? b.chapterId}</div>
                  </Td>
                  <Td className="whitespace-nowrap">
                    <div className="font-medium text-ink-900">{formatCurrency(b.amount)}</div>
                    {b.amount !== b.invoiceAmount && (
                      <div className="text-xs text-amber-700">Tagihan {formatCurrency(b.invoiceAmount)}</div>
                    )}
                  </Td>
                  <Td className="whitespace-nowrap text-ink-600">
                    {formatDate(b.paidAt)}
                    <div className="text-xs text-ink-400">{paymentMethodLabel(b.paymentMethod)}</div>
                  </Td>
                  <Td className="whitespace-nowrap text-ink-600">
                    {b.submittedBy ?? 'ST'}
                    <div className="text-xs text-ink-400">{formatDateTime(b.submittedAt)}</div>
                  </Td>
                  <Td className="whitespace-nowrap">
                    <BadgeBukti status={b.status} />
                    {b.reviewNote && <div className="mt-1 max-w-xs whitespace-normal text-xs text-ink-500">{b.reviewNote}</div>}
                  </Td>
                  <Td className="text-right">
                    <div className="flex flex-wrap items-center justify-end gap-2">
                      <a
                        href={b.proofUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="text-xs font-medium text-brand-500 hover:underline"
                      >
                        Lihat bukti
                      </a>
                      <AksiBukti bukti={b} onBerubah={daftar.reload} />
                    </div>
                  </Td>
                </Tr>
              ))}
            </TBody>
          </Table>
        )}
        <Pagination {...paginasi} satuan="bukti" />
      </Card>
    </div>
  )
}
