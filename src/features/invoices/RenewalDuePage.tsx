import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, CalendarClock, CheckCircle2, Search, X } from 'lucide-react'
import type { FeeSettings, RenewalDueMember } from '@/types'
import {
  Avatar,
  Badge,
  Button,
  Card,
  EmptyState,
  ExportMenu,
  Input,
  PageHeader,
  Select,
  Table,
  TBody,
  Td,
  Th,
  THead,
  Tr,
  TableSkeleton,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { useCan } from '@/features/auth/usePermission'
import { invoiceService, settingsService } from '@/services'
import { addDays, addYear, todayISO } from '@/lib/date'
import { formatCurrency, formatDate, formatDateTime } from '@/lib/format'
import { makeExportHandlers } from '@/lib/exporters'

type Urgency = 'all' | 'overdue' | 'week' | 'later'

const URGENCY_OPTIONS: { value: Urgency; label: string }[] = [
  { value: 'all', label: 'Semua Urgensi' },
  { value: 'overdue', label: 'Sudah terlewat' },
  { value: 'week', label: '≤ 7 hari' },
  { value: 'later', label: '> 7 hari' },
]

function DueBadge({ days }: { days: number }) {
  if (days < 0) return <Badge tone="red">Terlewat {Math.abs(days)} hari</Badge>
  if (days <= 7) return <Badge tone="red">{days} hari lagi</Badge>
  return <Badge tone="amber">{days} hari lagi</Badge>
}

export function RenewalDuePage() {
  const navigate = useNavigate()
  const { toast } = useToast()
  const { data: members, loading, reload } = useAsync<RenewalDueMember[]>(() =>
    invoiceService.renewalDue(30),
  )
  const { data: fees } = useAsync<FeeSettings>(() => settingsService.getFees())

  const canCreate = useCan('invoice:create')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [generating, setGenerating] = useState(false)
  const [search, setSearch] = useState('')
  const [urgency, setUrgency] = useState<Urgency>('all')

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return (members ?? []).filter((m) => {
      if (urgency === 'overdue' && m.daysUntilDue >= 0) return false
      if (urgency === 'week' && !(m.daysUntilDue >= 0 && m.daysUntilDue <= 7)) return false
      if (urgency === 'later' && m.daysUntilDue <= 7) return false
      if (
        q &&
        !m.name.toLowerCase().includes(q) &&
        !(m.email ?? '').toLowerCase().includes(q) &&
        !(m.chapter?.displayName ?? '').toLowerCase().includes(q)
      )
        return false
      return true
    })
  }, [members, search, urgency])

  const allSelected = filtered.length > 0 && filtered.every((m) => selected.has(m.id))

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })

  // Select-all applies to the CURRENTLY FILTERED rows only.
  const toggleAll = () => {
    setSelected(allSelected ? new Set() : new Set(filtered.map((m) => m.id)))
  }

  const selectedTotal = useMemo(
    () => (fees ? selected.size * fees.renewalFee : 0),
    [selected, fees],
  )

  const handleGenerate = async () => {
    if (!fees || selected.size === 0) return
    setGenerating(true)
    try {
      // Only act on rows still visible under the active filter.
      const targets = filtered.filter((m) => selected.has(m.id))
      for (const m of targets) {
        const periodStart = addDays(m.lastInvoice.periodEnd, 1)
        await invoiceService.create({
          memberId: m.id,
          type: 'renewal',
          amount: fees.renewalFee,
          dueDate: todayISO(),
          periodStart,
          periodEnd: addYear(periodStart),
        })
      }
      toast(`${targets.length} invoice renewal berhasil dibuat (draft).`)
      setSelected(new Set())
      reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal membuat invoice.', 'error')
    } finally {
      setGenerating(false)
    }
  }

  const exportHandlers = makeExportHandlers({
    filename: 'renewal-due',
    title: 'Renewal Due',
    subtitle: `Urgensi: ${URGENCY_OPTIONS.find((u) => u.value === urgency)?.label ?? 'Semua'}`,
    meta: [`${filtered.length} member`, `Dibuat ${formatDateTime(new Date())}`],
    columns: [
      { label: 'Member' },
      { label: 'Chapter' },
      { label: 'Email' },
      { label: 'Berakhir' },
      { label: 'Sisa Hari', align: 'right' },
    ],
    rows: filtered.map((m) => [
      m.name,
      m.chapter?.displayName ?? '',
      m.email ?? '',
      formatDate(m.lastInvoice.periodEnd),
      m.daysUntilDue,
    ]),
    onPopupBlocked: () => toast('Izinkan popup di browser untuk mengekspor PDF.', 'error'),
  })

  return (
    <div>
      <PageHeader
        title="Renewal Due"
        description="Member yang masa keanggotaannya berakhir dalam 30 hari ke depan."
        action={<ExportMenu {...exportHandlers} disabled={filtered.length === 0} />}
        breadcrumb={
          <button
            onClick={() => navigate('/invoices')}
            className="inline-flex items-center gap-1.5 text-sm text-ink-500 hover:text-ink-800"
          >
            <ArrowLeft className="h-4 w-4" />
            Kembali ke Invoice
          </button>
        }
      />

      <Card>
        {/* Filter: pencarian + urgensi */}
        <div className="space-y-3 border-b border-ink-100 p-4">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari nama member, email, atau chapter…"
              className="pl-10"
            />
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Select
              value={urgency}
              onChange={(e) => setUrgency(e.target.value as Urgency)}
              className="w-full sm:w-48"
            >
              {URGENCY_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
            {(search || urgency !== 'all') && (
              <button
                type="button"
                onClick={() => {
                  setSearch('')
                  setUrgency('all')
                }}
                className="rounded-lg p-1.5 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-700"
                aria-label="Reset filter renewal"
              >
                <X className="h-4 w-4" />
              </button>
            )}
            <span className="ml-auto text-sm text-ink-400">{filtered.length} member</span>
          </div>
        </div>

        {/* Bulk action bar */}
        {canCreate && selected.size > 0 && (
          <div className="flex flex-col gap-3 border-b border-ink-100 bg-brand-50/50 px-5 py-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="text-sm font-medium text-ink-700">
              {selected.size} member dipilih · Total {formatCurrency(selectedTotal)}
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>
                Batal
              </Button>
              <Button size="sm" loading={generating} onClick={handleGenerate}>
                <CheckCircle2 className="h-4 w-4" />
                Buat {selected.size} Invoice Renewal
              </Button>
            </div>
          </div>
        )}

        {loading ? (
          <TableSkeleton rows={6} cols={5} />
        ) : !members || members.length === 0 ? (
          <EmptyState
            icon={CalendarClock}
            title="Tidak ada renewal jatuh tempo"
            description="Semua member masih dalam masa keanggotaan aktif untuk 30 hari ke depan."
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={CalendarClock}
            title="Tidak ada hasil"
            description="Tidak ada member yang cocok dengan filter saat ini."
          />
        ) : (
          <>
            {/* Mobile cards */}
            <div className="divide-y divide-ink-100 lg:hidden">
              {filtered.map((m) => (
                <div
                  key={m.id}
                  onClick={() => toggle(m.id)}
                  className={`flex items-center gap-3 px-4 py-3.5 active:bg-ink-50 ${selected.has(m.id) ? 'bg-brand-50/50' : ''}`}
                >
                  <input
                    type="checkbox"
                    checked={selected.has(m.id)}
                    onChange={() => toggle(m.id)}
                    onClick={(e) => e.stopPropagation()}
                    className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                  />
                  <Avatar name={m.name} size="sm" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="truncate font-medium text-ink-900 text-sm">{m.name}</div>
                        <div className="text-xs text-ink-400">{m.chapter?.displayName ?? '—'}</div>
                      </div>
                      <div className="shrink-0 text-right">
                        <DueBadge days={m.daysUntilDue} />
                        <div className="text-xs text-ink-400 mt-1">Berakhir {formatDate(m.lastInvoice.periodEnd)}</div>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
            {/* Desktop table */}
            <div className="hidden lg:block">
              <Table>
                <THead>
                  <Tr>
                    <Th className="w-10">
                      <input
                        type="checkbox"
                        checked={allSelected}
                        onChange={toggleAll}
                        className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                      />
                    </Th>
                    <Th>Member</Th>
                    <Th>Chapter</Th>
                    <Th>Berakhir</Th>
                    <Th>Status</Th>
                    <Th>Invoice Terakhir</Th>
                  </Tr>
                </THead>
                <TBody>
                  {filtered.map((m) => (
                    <Tr key={m.id} onClick={() => toggle(m.id)} className={selected.has(m.id) ? 'bg-brand-50/40' : ''}>
                      <Td>
                        <input
                          type="checkbox"
                          checked={selected.has(m.id)}
                          onChange={() => toggle(m.id)}
                          onClick={(e) => e.stopPropagation()}
                          className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                        />
                      </Td>
                      <Td>
                        <div className="flex items-center gap-3">
                          <Avatar name={m.name} size="sm" />
                          <div className="leading-tight">
                            <div className="font-medium text-ink-900">{m.name}</div>
                            <div className="text-xs text-ink-400">{m.id}</div>
                          </div>
                        </div>
                      </Td>
                      <Td className="text-ink-600">{m.chapter?.displayName ?? '—'}</Td>
                      <Td className="whitespace-nowrap text-ink-600">{formatDate(m.lastInvoice.periodEnd)}</Td>
                      <Td>
                        <DueBadge days={m.daysUntilDue} />
                      </Td>
                      <Td>
                        <span className="font-mono text-[13px] text-ink-500">{m.lastInvoice.number}</span>
                      </Td>
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
