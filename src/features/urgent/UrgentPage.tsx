import { Fragment, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  ArrowRight,
  BellRing,
  CalendarClock,
  CheckCircle2,
  Download,
  Search,
  UserPlus,
} from 'lucide-react'
import type { Chapter, FeeSettings, InvoiceWithRelations, MemberWithChapter, RenewalDueMember } from '@/types'
import {
  Avatar,
  Badge,
  Button,
  Card,
  CardHeader,
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
import { chapterService, invoiceService, memberService, settingsService } from '@/services'
import { addDays, addYear, todayISO } from '@/lib/date'
import { formatCurrency, formatDate, formatDateTime } from '@/lib/format'
import { makeExportHandlers } from '@/lib/exporters'
import { cn } from '@/lib/cn'
import { downloadInvoice } from '@/features/invoices/lib/invoiceDocument'

function DaysOverdueBadge({ dueDate }: { dueDate: string }) {
  const days = Math.floor((Date.now() - new Date(dueDate).getTime()) / 86400000)
  if (days <= 0) return <Badge tone="amber">Hari ini</Badge>
  return <Badge tone="red">Telat {days} hari</Badge>
}

function DaysUntilBadge({ days }: { days: number }) {
  if (days < 0) return <Badge tone="red">Terlewat {Math.abs(days)} hari</Badge>
  if (days === 0) return <Badge tone="red">Hari ini</Badge>
  if (days <= 7) return <Badge tone="red">{days} hari lagi</Badge>
  return <Badge tone="amber">{days} hari lagi</Badge>
}

// ---------------------------------------------------------------------------
// Section: Overdue
// ---------------------------------------------------------------------------

/**
 * Pengingat tunggakan — lewat Paper.id, bukan WhatsApp manual.
 *
 * Dulu tombol ini menyusun pesan lalu membuka wa.me: admin mengirim sendiri
 * dari nomornya sendiri, tidak ada catatan bahwa pengingat pernah dikirim, dan
 * tidak ada satu pun jejak di blackbox. Halaman inilah tempat pengingat paling
 * sering dipakai, jadi justru di sini jejaknya paling dibutuhkan.
 */
function RemindButton({ invoice, className }: { invoice: InvoiceWithRelations; className: string }) {
  const [busy, setBusy] = useState(false)
  const { toast } = useToast()

  const click = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (busy) return
    setBusy(true)
    try {
      await invoiceService.resend(invoice.id)
      toast(`Pengingat ${invoice.number} dikirim lewat Paper.id.`, 'success')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Pengingat gagal dikirim.', 'error')
    } finally {
      setBusy(false)
    }
  }

  return (
    <button onClick={click} disabled={busy} className={className}>
      <BellRing className="h-3 w-3" />
      {busy ? 'Mengirim…' : 'Ingatkan'}
    </button>
  )
}

function OverdueSection({ invoices, loading }: { invoices: InvoiceWithRelations[] | null; loading: boolean }) {
  const navigate = useNavigate()

  return (
    <Card data-tour="urgent-overdue">
      <CardHeader
        title={
          <span className="flex items-center gap-2">
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-red-500 text-[11px] font-bold text-white">
              {loading ? '…' : invoices?.length ?? 0}
            </span>
            Invoice Overdue
          </span>
        }
        subtitle="Tagihan yang sudah melewati jatuh tempo dan belum dibayar."
        action={
          <button
            onClick={() => navigate('/invoices?status=overdue')}
            className="inline-flex items-center gap-1 text-sm font-medium text-brand-500 hover:text-brand-600"
          >
            Lihat semua <ArrowRight className="h-4 w-4" />
          </button>
        }
      />

      {loading ? (
        <TableSkeleton rows={4} cols={5} />
      ) : !invoices || invoices.length === 0 ? (
        <EmptyState
          icon={CheckCircle2}
          title="Tidak ada invoice overdue"
          description="Semua tagihan dalam kondisi baik."
        />
      ) : (
        <>
          {/* Mobile cards */}
          <div className="divide-y divide-ink-100 lg:hidden">
            {invoices.map((inv) => (
              <div
                key={inv.id}
                className="flex items-center gap-3 px-4 py-3.5"
              >
                <Avatar name={inv.member?.name ?? '?'} size="sm" />
                <div className="min-w-0 flex-1">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <div className="truncate font-medium text-ink-900 text-sm">{inv.member?.name ?? '—'}</div>
                      <div className="text-xs text-ink-400">{inv.chapter?.displayName ?? '—'} · {inv.number}</div>
                    </div>
                    <div className="shrink-0 text-right">
                      <div className="font-semibold text-ink-900 text-sm">{formatCurrency(inv.amount)}</div>
                      <div className="text-xs text-ink-400 mt-0.5">{formatDate(inv.dueDate)}</div>
                    </div>
                  </div>
                  <div className="mt-2 flex flex-wrap items-center gap-2">
                    <DaysOverdueBadge dueDate={inv.dueDate} />
                    <button
                      onClick={(e) => { e.stopPropagation(); downloadInvoice(inv) }}
                      className="inline-flex items-center gap-1 rounded-lg bg-brand-50 px-2.5 py-1 text-xs font-medium text-brand-600"
                    >
                      <Download className="h-3 w-3" />
                      Invoice
                    </button>
                    <RemindButton
                      invoice={inv}
                      className="inline-flex items-center gap-1 rounded-lg bg-brand-50 px-2.5 py-1 text-xs font-medium text-brand-600 disabled:opacity-50"
                    />
                    <button
                      onClick={(e) => { e.stopPropagation(); navigate(`/invoices/${inv.id}`) }}
                      className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-brand-500"
                    >
                      Detail <ArrowRight className="h-3 w-3" />
                    </button>
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
                  <Th>Member</Th>
                  <Th>Chapter</Th>
                  <Th>Nominal</Th>
                  <Th>Jatuh Tempo</Th>
                  <Th>Keterlambatan</Th>
                  <Th className="text-right">Aksi</Th>
                </Tr>
              </THead>
              <TBody>
                {invoices.map((inv) => (
                  <Tr key={inv.id}>
                    <Td>
                      <div className="flex items-center gap-3">
                        <Avatar name={inv.member?.name ?? '?'} size="sm" />
                        <div className="leading-tight">
                          <div className="font-medium text-ink-900">{inv.member?.name ?? '—'}</div>
                          <div className="text-xs text-ink-400">{inv.number}</div>
                        </div>
                      </div>
                    </Td>
                    <Td className="text-ink-600">{inv.chapter?.displayName ?? '—'}</Td>
                    <Td className="font-medium text-ink-900">{formatCurrency(inv.amount)}</Td>
                    <Td className="whitespace-nowrap text-ink-600">{formatDate(inv.dueDate)}</Td>
                    <Td>
                      <DaysOverdueBadge dueDate={inv.dueDate} />
                    </Td>
                    <Td className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={(e) => { e.stopPropagation(); downloadInvoice(inv) }}
                          className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-brand-600 bg-brand-50 hover:bg-brand-100"
                        >
                          <Download className="h-3 w-3" />
                          Invoice
                        </button>
                        <RemindButton
                          invoice={inv}
                          className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-brand-600 bg-brand-50 hover:bg-brand-100 disabled:opacity-50"
                        />
                        <button
                          onClick={(e) => { e.stopPropagation(); navigate(`/invoices/${inv.id}`) }}
                          className="inline-flex items-center gap-1 rounded-lg px-3 py-1.5 text-xs font-medium text-brand-500 hover:bg-brand-50"
                        >
                          Detail <ArrowRight className="h-3 w-3" />
                        </button>
                      </div>
                    </Td>
                  </Tr>
                ))}
              </TBody>
            </Table>
          </div>
        </>
      )}
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Section: Renewal Due
// ---------------------------------------------------------------------------

function RenewalSection({
  members,
  loading,
  fees,
  onGenerated,
}: {
  members: RenewalDueMember[] | null
  loading: boolean
  fees: FeeSettings | null
  onGenerated: () => void
}) {
  const navigate = useNavigate()
  const { toast } = useToast()
  const canCreate = useCan('invoice:create')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [generating, setGenerating] = useState(false)

  const urgent = useMemo(() => members?.filter((m) => m.daysUntilDue <= 7) ?? [], [members])
  const upcoming = useMemo(() => members?.filter((m) => m.daysUntilDue > 7) ?? [], [members])
  const allIds = useMemo(() => members?.map((m) => m.id) ?? [], [members])
  const allSelected = allIds.length > 0 && selected.size === allIds.length

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })

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
      onGenerated()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal membuat invoice.', 'error')
    } finally {
      setGenerating(false)
    }
  }

  const selectedTotal = fees ? selected.size * fees.renewalFee : 0

  return (
    <Card data-tour="urgent-renewal">
      <CardHeader
        title={
          <span className="flex items-center gap-2">
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-amber-500 text-[11px] font-bold text-white">
              {loading ? '…' : members?.length ?? 0}
            </span>
            Renewal Jatuh Tempo
          </span>
        }
        subtitle="Member yang masa keanggotaannya berakhir dalam 30 hari ke depan."
        action={
          !loading && members && members.length > 0 ? (
            <button
              onClick={() =>
                setSelected(allSelected ? new Set() : new Set(allIds))
              }
              className="text-sm font-medium text-brand-500 hover:text-brand-600"
            >
              {allSelected ? 'Batal semua' : 'Pilih semua'}
            </button>
          ) : null
        }
      />

      {/* Bulk action bar */}
      {canCreate && selected.size > 0 && (
        <div className="flex flex-col gap-3 border-b border-ink-100 bg-amber-50/50 px-5 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="text-sm font-medium text-ink-700">
            {selected.size} member dipilih · Total {formatCurrency(selectedTotal)}
          </div>
          <div className="flex gap-2">
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
        <TableSkeleton rows={5} cols={5} />
      ) : !members || members.length === 0 ? (
        <EmptyState
          icon={CalendarClock}
          title="Tidak ada renewal jatuh tempo"
          description="Semua member masih dalam masa keanggotaan aktif untuk 30 hari ke depan."
        />
      ) : (
        <>
          {/* Mobile cards */}
          <div className="lg:hidden">
            {urgent.length > 0 && (
              <div className="border-b border-red-100 bg-red-50/40 px-4 py-2 text-xs font-semibold uppercase tracking-wide text-red-600">
                Sangat Mendesak (≤ 7 hari)
              </div>
            )}
            <div className="divide-y divide-ink-100">
              {[...urgent, ...upcoming].map((m, idx) => {
                const isFirstUpcoming = idx === urgent.length && urgent.length > 0
                return (
                  <Fragment key={m.id}>
                    {isFirstUpcoming && (
                      <div className="border-b border-amber-100 bg-amber-50/30 px-4 py-2 text-xs font-semibold uppercase tracking-wide text-amber-600">
                        Dalam 30 Hari
                      </div>
                    )}
                    <div
                      onClick={() => toggle(m.id)}
                      className={cn('flex items-center gap-3 px-4 py-3.5 active:bg-ink-50', selected.has(m.id) && 'bg-brand-50/50')}
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
                            <DaysUntilBadge days={m.daysUntilDue} />
                            <div className="text-xs text-ink-400 mt-1">Berakhir {formatDate(m.lastInvoice.periodEnd)}</div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </Fragment>
                )
              })}
            </div>
          </div>

          {/* Desktop table */}
          <div className="hidden lg:block">
            {urgent.length > 0 && (
              <div className="border-b border-red-100 bg-red-50/40 px-5 py-2 text-xs font-semibold uppercase tracking-wide text-red-600">
                Sangat Mendesak (≤ 7 hari)
              </div>
            )}
            <Table>
              <THead>
                <Tr>
                  <Th className="w-10">
                    <input
                      type="checkbox"
                      checked={allSelected}
                      onChange={() => setSelected(allSelected ? new Set() : new Set(allIds))}
                      className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                    />
                  </Th>
                  <Th>Member</Th>
                  <Th>Chapter</Th>
                  <Th>Berakhir</Th>
                  <Th>Sisa Waktu</Th>
                  <Th className="text-right">Aksi</Th>
                </Tr>
              </THead>
              <TBody>
                {[...urgent, ...upcoming].map((m, idx) => {
                  const isFirstUpcoming = idx === urgent.length && urgent.length > 0
                  return (
                    <Fragment key={m.id}>
                      {isFirstUpcoming && (
                        <tr className="border-b border-amber-100 bg-amber-50/30">
                          <td colSpan={6} className="px-5 py-2 text-xs font-semibold uppercase tracking-wide text-amber-600">
                            Dalam 30 Hari
                          </td>
                        </tr>
                      )}
                      <Tr onClick={() => toggle(m.id)} className={cn(selected.has(m.id) && 'bg-brand-50/40')}>
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
                          <DaysUntilBadge days={m.daysUntilDue} />
                        </Td>
                        <Td className="text-right">
                          <button
                            onClick={(e) => { e.stopPropagation(); navigate(`/members/${m.id}`) }}
                            className="inline-flex items-center gap-1 rounded-lg px-3 py-1.5 text-xs font-medium text-brand-500 hover:bg-brand-50"
                          >
                            Profil <ArrowRight className="h-3 w-3" />
                          </button>
                        </Td>
                      </Tr>
                    </Fragment>
                  )
                })}
              </TBody>
            </Table>
          </div>
        </>
      )}
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Section: Eligible for Registration
// ---------------------------------------------------------------------------

function RegistrationSection({
  members,
  loading,
  fees,
}: {
  members: MemberWithChapter[] | null
  loading: boolean
  fees: FeeSettings | null
}) {
  const navigate = useNavigate()
  const { toast } = useToast()
  const canCreate = useCan('invoice:create')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [generating, setGenerating] = useState(false)

  const allSelected = !!members && members.length > 0 && selected.size === members.length

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })

  const handleGenerate = async () => {
    if (!members || !fees || selected.size === 0) return
    setGenerating(true)
    try {
      const targets = members.filter((m) => selected.has(m.id))
      const today = todayISO()
      for (const m of targets) {
        await invoiceService.create({
          memberId: m.id,
          type: 'registration',
          amount: fees.registrationFee,
          dueDate: today,
          periodStart: today,
          periodEnd: addYear(today),
        })
      }
      toast(`${targets.length} invoice pendaftaran berhasil dibuat (draft).`)
      setSelected(new Set())
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal membuat invoice.', 'error')
    } finally {
      setGenerating(false)
    }
  }

  const selectedTotal = fees ? selected.size * fees.registrationFee : 0

  return (
    <Card>
      <CardHeader
        title={
          <span className="flex items-center gap-2">
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-blue-500 text-[11px] font-bold text-white">
              {loading ? '…' : members?.length ?? 0}
            </span>
            Perlu Invoice Pendaftaran
          </span>
        }
        subtitle="Member/visitor yang belum memiliki invoice pendaftaran aktif."
        action={
          !loading && members && members.length > 0 ? (
            <button
              onClick={() => setSelected(allSelected ? new Set() : new Set(members.map((m) => m.id)))}
              className="text-sm font-medium text-brand-500 hover:text-brand-600"
            >
              {allSelected ? 'Batal semua' : 'Pilih semua'}
            </button>
          ) : null
        }
      />

      {canCreate && selected.size > 0 && (
        <div className="flex flex-col gap-3 border-b border-ink-100 bg-blue-50/50 px-5 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="text-sm font-medium text-ink-700">
            {selected.size} member dipilih · Total {formatCurrency(selectedTotal)}
          </div>
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>
              Batal
            </Button>
            <Button size="sm" loading={generating} onClick={handleGenerate}>
              <UserPlus className="h-4 w-4" />
              Buat {selected.size} Invoice Pendaftaran
            </Button>
          </div>
        </div>
      )}

      {loading ? (
        <TableSkeleton rows={4} cols={4} />
      ) : !members || members.length === 0 ? (
        <EmptyState
          icon={CheckCircle2}
          title="Semua member sudah memiliki invoice pendaftaran"
          description="Tidak ada visitor/member baru yang perlu dibuatkan invoice pendaftaran."
        />
      ) : (
        <>
          {/* Mobile cards */}
          <div className="divide-y divide-ink-100 lg:hidden">
            {members.map((m) => (
              <div
                key={m.id}
                onClick={() => toggle(m.id)}
                className={cn('flex items-center gap-3 px-4 py-3.5 active:bg-ink-50', selected.has(m.id) && 'bg-brand-50/50')}
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
                  <div className="truncate font-medium text-ink-900 text-sm">{m.name}</div>
                  <div className="text-xs text-ink-400">{m.chapter?.displayName ?? '—'}{m.joinedDate ? ` · Bergabung ${formatDate(m.joinedDate)}` : ''}</div>
                  {m.email && <div className="truncate text-xs text-ink-400">{m.email}</div>}
                </div>
                <button
                  onClick={(e) => { e.stopPropagation(); navigate(`/members/${m.id}`) }}
                  className="shrink-0 rounded-lg p-1.5 text-brand-500 hover:bg-brand-50"
                >
                  <ArrowRight className="h-4 w-4" />
                </button>
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
                      onChange={() => setSelected(allSelected ? new Set() : new Set(members.map((m) => m.id)))}
                      className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                    />
                  </Th>
                  <Th>Member</Th>
                  <Th>Chapter</Th>
                  <Th>Bergabung</Th>
                  <Th className="text-right">Aksi</Th>
                </Tr>
              </THead>
              <TBody>
                {members.map((m) => (
                  <Tr key={m.id} onClick={() => toggle(m.id)} className={cn(selected.has(m.id) && 'bg-brand-50/40')}>
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
                          <div className="text-xs text-ink-400">{m.email ?? m.id}</div>
                        </div>
                      </div>
                    </Td>
                    <Td className="text-ink-600">{m.chapter?.displayName ?? '—'}</Td>
                    <Td className="whitespace-nowrap text-ink-600">{m.joinedDate ? formatDate(m.joinedDate) : '—'}</Td>
                    <Td className="text-right">
                      <button
                        onClick={(e) => { e.stopPropagation(); navigate(`/members/${m.id}`) }}
                        className="inline-flex items-center gap-1 rounded-lg px-3 py-1.5 text-xs font-medium text-brand-500 hover:bg-brand-50"
                      >
                        Profil <ArrowRight className="h-3 w-3" />
                      </button>
                    </Td>
                  </Tr>
                ))}
              </TBody>
            </Table>
          </div>
        </>
      )}
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function UrgentPage() {
  const { toast } = useToast()
  const [chapterId, setChapterId] = useState<string>('all')
  const [search, setSearch] = useState('')

  const {
    data: overdue,
    loading: loadingOverdue,
  } = useAsync<InvoiceWithRelations[]>(() => invoiceService.list({ status: 'overdue' }))

  const {
    data: renewalDue,
    loading: loadingRenewal,
    reload: reloadRenewal,
  } = useAsync<RenewalDueMember[]>(() => invoiceService.renewalDue(30))

  const {
    data: eligible,
    loading: loadingEligible,
  } = useAsync<MemberWithChapter[]>(() => memberService.eligibleForRegistration())

  const { data: fees } = useAsync<FeeSettings>(() => settingsService.getFees())
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())

  const q = search.trim().toLowerCase()

  const filteredOverdue = useMemo(
    () =>
      overdue?.filter(
        (i) =>
          (chapterId === 'all' || i.chapterId === chapterId) &&
          (!q ||
            i.number.toLowerCase().includes(q) ||
            (i.member?.name ?? '').toLowerCase().includes(q)),
      ) ?? null,
    [overdue, chapterId, q],
  )
  const filteredRenewal = useMemo(
    () =>
      renewalDue?.filter(
        (m) =>
          (chapterId === 'all' || m.chapterId === chapterId) &&
          (!q || m.name.toLowerCase().includes(q) || (m.email ?? '').toLowerCase().includes(q)),
      ) ?? null,
    [renewalDue, chapterId, q],
  )
  const filteredEligible = useMemo(
    () =>
      eligible?.filter(
        (m) =>
          (chapterId === 'all' || m.chapterId === chapterId) &&
          (!q || m.name.toLowerCase().includes(q) || (m.email ?? '').toLowerCase().includes(q)),
      ) ?? null,
    [eligible, chapterId, q],
  )

  const totalUrgent =
    (filteredOverdue?.length ?? 0) +
    (filteredRenewal?.length ?? 0) +
    (filteredEligible?.length ?? 0)

  // One combined "action list" export covering all three sections, respecting
  // the active chapter + search filters.
  const exportHandlers = makeExportHandlers({
    filename: 'perlu-tindakan',
    title: 'Perlu Tindakan',
    subtitle: `Chapter: ${chapterId === 'all' ? 'Semua' : (chapters?.find((c) => c.id === chapterId)?.displayName ?? chapterId)}`,
    meta: [`${totalUrgent} item`, `Dibuat ${formatDateTime(new Date())}`],
    columns: [
      { label: 'Jenis' },
      { label: 'Nama' },
      { label: 'Chapter' },
      { label: 'Detail' },
      { label: 'Tanggal' },
    ],
    rows: [
      ...(filteredOverdue ?? []).map((i) => [
        'Overdue',
        i.member?.name ?? '',
        i.chapter?.displayName ?? '',
        `${i.number} · ${formatCurrency(i.amount)}`,
        formatDate(i.dueDate),
      ]),
      ...(filteredRenewal ?? []).map((m) => [
        'Renewal',
        m.name,
        m.chapter?.displayName ?? '',
        `Berakhir ${formatDate(m.lastInvoice.periodEnd)} · ${m.daysUntilDue} hari`,
        formatDate(m.lastInvoice.periodEnd),
      ]),
      ...(filteredEligible ?? []).map((m) => [
        'Pendaftaran',
        m.name,
        m.chapter?.displayName ?? '',
        m.email ?? '',
        m.joinedDate ? formatDate(m.joinedDate) : '',
      ]),
    ],
    onPopupBlocked: () => toast('Izinkan popup di browser untuk mengekspor PDF.', 'error'),
  })

  return (
    <div>
      <PageHeader
        title="Perlu Tindakan"
        description="Tagihan overdue, renewal jatuh tempo, dan pendaftaran yang belum diproses."
        action={
          <div className="flex flex-wrap items-center gap-2 sm:gap-3">
            <Select
              value={chapterId}
              onChange={(e) => setChapterId(e.target.value)}
              className="w-full sm:w-44"
            >
              <option value="all">Semua Chapter</option>
              {chapters?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.displayName}
                </option>
              ))}
            </Select>
            <ExportMenu {...exportHandlers} disabled={totalUrgent === 0} />
            {totalUrgent > 0 && (
              <div className="flex items-center gap-2 rounded-xl bg-red-50 px-4 py-2">
                <AlertTriangle className="h-4 w-4 text-red-500" />
                <span className="text-sm font-semibold text-red-600">{totalUrgent} item</span>
              </div>
            )}
          </div>
        }
      />

      {/* Pencarian lintas-seksi */}
      <Card className="mb-5">
        <div className="relative p-4">
          <Search className="pointer-events-none absolute left-7 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Cari nama member atau nomor invoice…"
            className="pl-10"
          />
        </div>
      </Card>

      <div className="space-y-5">
        <OverdueSection invoices={filteredOverdue} loading={loadingOverdue} />
        <RenewalSection
          members={filteredRenewal}
          loading={loadingRenewal}
          fees={fees}
          onGenerated={reloadRenewal}
        />
        <RegistrationSection members={filteredEligible} loading={loadingEligible} fees={fees} />
      </div>
    </div>
  )
}
