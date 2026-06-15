import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { FileText, Plus, Search } from 'lucide-react'
import type { Chapter, InvoiceStatus, InvoiceType, InvoiceWithRelations } from '@/types'
import {
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  Select,
  TableSkeleton,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, invoiceService } from '@/services'
import { InvoiceTable } from './components/InvoiceTable'

const STATUS_OPTIONS: { value: InvoiceStatus | 'all'; label: string }[] = [
  { value: 'all', label: 'Semua Status' },
  { value: 'draft', label: 'Draft' },
  { value: 'sent', label: 'Terkirim' },
  { value: 'paid', label: 'Lunas' },
  { value: 'overdue', label: 'Overdue' },
  { value: 'cancelled', label: 'Dibatalkan' },
]

const TYPE_OPTIONS: { value: InvoiceType | 'all'; label: string }[] = [
  { value: 'all', label: 'Semua Tipe' },
  { value: 'registration', label: 'Pendaftaran' },
  { value: 'renewal', label: 'Renewal' },
]

export function InvoiceListPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const initialStatus = (searchParams.get('status') as InvoiceStatus | null) ?? 'all'

  const { data: invoices, loading } = useAsync<InvoiceWithRelations[]>(() => invoiceService.list())
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())

  const [status, setStatus] = useState<InvoiceStatus | 'all'>(initialStatus)
  const [type, setType] = useState<InvoiceType | 'all'>('all')
  const [chapterId, setChapterId] = useState<string>('all')
  const [search, setSearch] = useState('')

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
          <Button onClick={() => navigate('/invoices/new')}>
            <Plus className="h-4 w-4" />
            Buat Invoice
          </Button>
        }
      />

      <Card>
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
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:flex">
            <Select value={status} onChange={(e) => setStatus(e.target.value as InvoiceStatus | 'all')} className="lg:w-44">
              {STATUS_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
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
            <InvoiceTable invoices={filtered} />
            <div className="px-5 py-3 text-xs text-ink-400">
              Menampilkan {filtered.length} dari {invoices?.length ?? 0} invoice
            </div>
          </>
        )}
      </Card>
    </div>
  )
}
