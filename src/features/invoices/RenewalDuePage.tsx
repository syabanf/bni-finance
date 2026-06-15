import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, CalendarClock, CheckCircle2 } from 'lucide-react'
import type { FeeSettings, RenewalDueMember } from '@/types'
import {
  Avatar,
  Badge,
  Button,
  Card,
  EmptyState,
  PageHeader,
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
import { invoiceService, settingsService } from '@/services'
import { addDays, addYear, todayISO } from '@/lib/date'
import { formatCurrency, formatDate } from '@/lib/format'

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

  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [generating, setGenerating] = useState(false)

  const allSelected = !!members && members.length > 0 && selected.size === members.length

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })

  const toggleAll = () => {
    if (!members) return
    setSelected(allSelected ? new Set() : new Set(members.map((m) => m.id)))
  }

  const selectedTotal = useMemo(
    () => (fees ? selected.size * fees.renewalFee : 0),
    [selected, fees],
  )

  const handleGenerate = async () => {
    if (!members || !fees || selected.size === 0) return
    setGenerating(true)
    try {
      const targets = members.filter((m) => selected.has(m.id))
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

  return (
    <div>
      <PageHeader
        title="Renewal Due"
        description="Member yang masa keanggotaannya berakhir dalam 30 hari ke depan."
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
        {/* Bulk action bar */}
        {selected.size > 0 && (
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
        ) : (
          <>
            {/* Mobile cards */}
            <div className="divide-y divide-ink-100 lg:hidden">
              {members.map((m) => (
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
                  {members.map((m) => (
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
