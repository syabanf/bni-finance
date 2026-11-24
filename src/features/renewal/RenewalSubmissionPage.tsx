import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Globe2, Undo2 } from 'lucide-react'
import type { InvoiceWithRelations } from '@/types'
import {
  Button,
  Card,
  EmptyState,
  ErrorState,
  Input,
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
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { usePaginasi } from '@/hooks/usePaginasi'
import { chapterService, invoiceService } from '@/services'
import { Saringan } from '@/features/payments/components/Saringan'
import { formatCurrency, formatDate } from '@/lib/format'
import { todayISO } from '@/lib/date'

type Tab = 'pending' | 'done'

/**
 * Langkah 6 alur renewal klien: invoice renewal yang sudah lunas menunggu
 * diajukan finance ke BNI Global. Finance mencentang yang sudah diajukan dan
 * menandainya sekaligus, supaya daftar "belum diajukan" habis.
 */
export function RenewalSubmissionPage() {
  const { toast } = useToast()
  const [tab, setTab] = useState<Tab>('pending')
  const [tanggal, setTanggal] = useState(todayISO())
  const [terpilih, setTerpilih] = useState<Set<string>>(new Set())
  const [sibuk, setSibuk] = useState(false)

  const [cari, setCari] = useState('')
  const [chapterId, setChapterId] = useState('all')
  const chapters = useAsync(() => chapterService.list().catch(() => []), [])
  const belum = useAsync(() => invoiceService.list({ submission: 'pending' }), [])
  const sudah = useAsync(() => invoiceService.list({ submission: 'done' }), [])
  const aktif = tab === 'pending' ? belum : sudah
  const cocok = useMemo(() => {
    const q = cari.trim().toLowerCase()
    return (i: InvoiceWithRelations) =>
      (chapterId === 'all' || i.chapterId === chapterId) &&
      (!q || i.number.toLowerCase().includes(q) || (i.member?.name ?? '').toLowerCase().includes(q))
  }, [cari, chapterId])
  const daftar = useMemo(
    () =>
      (aktif.data ?? []).filter(cocok).sort((a, b) =>
        tab === 'pending'
          ? (a.paidAt ?? '').localeCompare(b.paidAt ?? '')
          : (b.renewalSubmittedAt ?? '').localeCompare(a.renewalSubmittedAt ?? ''),
      ),
    [aktif.data, tab, cocok],
  )
  const { potongan, paginasi } = usePaginasi(daftar)

  useEffect(() => setTerpilih(new Set()), [tab, belum.data, cari, chapterId])

  const muatUlang = () => {
    belum.reload()
    sudah.reload()
  }

  const tandai = async () => {
    setSibuk(true)
    try {
      const hasil = await invoiceService.markRenewalSubmitted([...terpilih], tanggal)
      toast(
        hasil.dilewati > 0
          ? `${hasil.ditandai} ditandai, ${hasil.dilewati} dilewati karena sudah diajukan atau belum lunas.`
          : `${hasil.ditandai} renewal ditandai sudah diajukan ke BNI Global.`,
      )
      muatUlang()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menandai pengajuan.', 'error')
    } finally {
      setSibuk(false)
    }
  }

  const batalkan = async (id: string, nomor: string) => {
    if (!window.confirm(`Batalkan tanda pengajuan ${nomor}?`)) return
    try {
      await invoiceService.unmarkRenewalSubmitted(id)
      toast(`Tanda pengajuan ${nomor} dibatalkan.`)
      muatUlang()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal membatalkan tanda.', 'error')
    }
  }

  const alihkan = (id: string) =>
    setTerpilih((lama) => {
      const baru = new Set(lama)
      if (baru.has(id)) baru.delete(id)
      else baru.add(id)
      return baru
    })

  return (
    <div>
      <PageHeader
        title="Pengajuan ke BNI Global"
        description="Invoice renewal yang sudah lunas. Tandai setelah finance mengajukan renewal member ke BNI Global."
      />
      <Card>
        <TabBar
          tabs={[
            { value: 'pending', label: 'Belum diajukan', count: belum.data?.filter(cocok).length },
            { value: 'done', label: 'Sudah diajukan', count: sudah.data?.filter(cocok).length },
          ]}
          value={tab}
          onChange={setTab}
        />
        <Saringan
          cari={cari}
          onCari={setCari}
          chapterId={chapterId}
          onChapter={setChapterId}
          chapters={chapters.data ?? []}
        />
        {tab === 'pending' && terpilih.size > 0 && (
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-ink-100 bg-brand-50/40 px-5 py-3">
            <span className="text-sm font-medium text-ink-700">{terpilih.size} dipilih</span>
            <div className="flex flex-wrap items-center gap-2">
              <label className="flex items-center gap-2 text-sm text-ink-600">
                Diajukan tanggal
                <Input
                  type="date"
                  value={tanggal}
                  max={todayISO()}
                  onChange={(e) => setTanggal(e.target.value)}
                  className="w-40"
                />
              </label>
              <Button size="sm" loading={sibuk} onClick={tandai}>
                <Globe2 className="h-4 w-4" />
                Tandai sudah diajukan
              </Button>
            </div>
          </div>
        )}
        {aktif.loading ? (
          <TableSkeleton />
        ) : aktif.error ? (
          <ErrorState message={aktif.error} onRetry={aktif.reload} />
        ) : daftar.length === 0 ? (
          <EmptyState
            title={tab === 'pending' ? 'Semua renewal yang lunas sudah diajukan' : 'Belum ada renewal yang diajukan'}
            description="Invoice renewal muncul di sini setelah lunas."
          />
        ) : (
          <Table>
            <THead>
              <Tr>
                {tab === 'pending' && (
                  <Th className="w-10">
                    <input
                      type="checkbox"
                      aria-label="Pilih semua"
                      checked={terpilih.size === daftar.length}
                      onChange={(e) => setTerpilih(e.target.checked ? new Set(daftar.map((i) => i.id)) : new Set())}
                      className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                    />
                  </Th>
                )}
                <Th>Member</Th>
                <Th>Invoice</Th>
                <Th>Lunas</Th>
                {tab === 'done' && <Th>Diajukan</Th>}
                {tab === 'done' && <Th className="text-right">Tindakan</Th>}
              </Tr>
            </THead>
            <TBody>
              {potongan.map((i) => (
                <Tr key={i.id}>
                  {tab === 'pending' && (
                    <Td>
                      <input
                        type="checkbox"
                        aria-label={`Pilih ${i.member?.name ?? i.number}`}
                        checked={terpilih.has(i.id)}
                        onChange={() => alihkan(i.id)}
                        className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                      />
                    </Td>
                  )}
                  <Td className="whitespace-nowrap">
                    <div className="font-medium text-ink-900">{i.member?.name ?? i.memberId}</div>
                    <div className="text-xs text-ink-400">{i.chapter?.displayName ?? i.chapterId}</div>
                  </Td>
                  <Td className="whitespace-nowrap">
                    <Link to={`/invoices/${i.id}`} className="font-medium text-brand-600 hover:underline">
                      {i.number}
                    </Link>
                    <div className="text-xs text-ink-500">{formatCurrency(i.paidAmount ?? i.amount)}</div>
                  </Td>
                  <Td className="whitespace-nowrap text-ink-600">{i.paidAt ? formatDate(i.paidAt) : '—'}</Td>
                  {tab === 'done' && (
                    <Td className="whitespace-nowrap text-ink-600">
                      {i.renewalSubmittedAt ? formatDate(i.renewalSubmittedAt) : '—'}
                      {i.renewalSubmittedBy && <div className="text-xs text-ink-400">{i.renewalSubmittedBy}</div>}
                    </Td>
                  )}
                  {tab === 'done' && (
                    <Td className="text-right">
                      <Button size="sm" variant="outline" onClick={() => batalkan(i.id, i.number)}>
                        <Undo2 className="h-4 w-4" />
                        Batalkan tanda
                      </Button>
                    </Td>
                  )}
                </Tr>
              ))}
            </TBody>
          </Table>
        )}
        <Pagination {...paginasi} satuan="invoice" />
      </Card>
    </div>
  )
}
