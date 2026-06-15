import { Fragment, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  ArrowRight,
  CalendarClock,
  CheckCircle2,
  MessageCircle,
  Send,
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
import { chapterService, invoiceService, memberService, settingsService } from '@/services'
import { addDays, addYear, todayISO } from '@/lib/date'
import { formatCurrency, formatDate } from '@/lib/format'
import { cn } from '@/lib/cn'

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

function sendOverdueWa(inv: InvoiceWithRelations) {
  const name = inv.member?.name ?? 'Bapak/Ibu'
  const msg = [
    `Halo ${name},`,
    ``,
    `Kami mengingatkan bahwa tagihan BNI Grow Anda telah *melewati jatuh tempo*:`,
    `📋 No. Invoice: *${inv.number}*`,
    `💰 Nominal: *${formatCurrency(inv.amount)}*`,
    `📅 Jatuh Tempo: *${formatDate(inv.dueDate)}*`,
    ``,
    inv.paperIdPaymentUrl
      ? `Silakan segera lakukan pembayaran melalui:\n${inv.paperIdPaymentUrl}`
      : `Silakan segera hubungi kami untuk melakukan pembayaran.`,
    ``,
    `Terima kasih. 🙏`,
  ].join('\n')
  window.open(`https://wa.me/?text=${encodeURIComponent(msg)}`, '_blank')
}

function OverdueSection({ invoices, loading }: { invoices: InvoiceWithRelations[] | null; loading: boolean }) {
  const navigate = useNavigate()
  const { toast } = useToast()
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [sending, setSending] = useState(false)

  const toggle = (id: string) => setSelected((prev) => {
    const next = new Set(prev)
    next.has(id) ? next.delete(id) : next.add(id)
    return next
  })

  const allIds = invoices?.map((i) => i.id) ?? []
  const allSelected = allIds.length > 0 && allIds.every((id) => selected.has(id))

  const handleBulkSend = async () => {
    if (!invoices || selected.size === 0) return
    setSending(true)
    let sent = 0
    try {
      const targets = invoices.filter((i) => selected.has(i.id))
      for (const inv of targets) {
        await invoiceService.resend(inv.id)
        sent++
      }
      toast(`${sent} invoice berhasil dikirim ulang ke Paper.id.`)
      setSelected(new Set())
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal mengirim.', 'error')
    } finally {
      setSending(false)
    }
  }

  return (
    <Card>
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

      {/* Bulk action bar */}
      {selected.size > 0 && (
        <div className="flex flex-col gap-3 border-b border-ink-100 bg-red-50/50 px-5 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="text-sm font-medium text-ink-700">{selected.size} invoice dipilih</div>
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>Batal</Button>
            <Button size="sm" loading={sending} onClick={handleBulkSend}>
              <Send className="h-4 w-4" />
              Kirim {selected.size} ke Paper.id
            </Button>
          </div>
        </div>
      )}

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
                onClick={() => toggle(inv.id)}
                className={cn('flex items-center gap-3 px-4 py-3.5 active:bg-ink-50', selected.has(inv.id) && 'bg-red-50/50')}
              >
                <input
                  type="checkbox"
                  checked={selected.has(inv.id)}
                  onChange={() => toggle(inv.id)}
                  onClick={(e) => e.stopPropagation()}
                  className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                />
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
                  <div className="mt-2 flex items-center gap-2">
                    <DaysOverdueBadge dueDate={inv.dueDate} />
                    <button
                      onClick={(e) => { e.stopPropagation(); sendOverdueWa(inv) }}
                      className="inline-flex items-center gap-1 rounded-lg bg-[#25D366]/10 px-2.5 py-1 text-xs font-medium text-[#128C7E]"
                    >
                      <MessageCircle className="h-3 w-3" />
                      Kirim WA
                    </button>
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
                  <Th>Nominal</Th>
                  <Th>Jatuh Tempo</Th>
                  <Th>Keterlambatan</Th>
                  <Th className="text-right">Aksi</Th>
                </Tr>
              </THead>
              <TBody>
                {invoices.map((inv) => (
                  <Tr key={inv.id} onClick={() => toggle(inv.id)} className={cn(selected.has(inv.id) && 'bg-red-50/40')}>
                    <Td>
                      <input
                        type="checkbox"
                        checked={selected.has(inv.id)}
                        onChange={() => toggle(inv.id)}
                        onClick={(e) => e.stopPropagation()}
                        className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                      />
                    </Td>
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
                          onClick={(e) => { e.stopPropagation(); sendOverdueWa(inv) }}
                          className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-[#128C7E] bg-[#25D366]/10 hover:bg-[#25D366]/20"
                        >
                          <MessageCircle className="h-3 w-3" />
                          WA
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); navigate(`/invoices/${inv.id}`) }}
                          className="inline-flex items-center gap-1 rounded-lg px-3 py-1.5 text-xs font-medium text-brand-500 hover:bg-brand-50"
                        >
                          Lihat Detail <ArrowRight className="h-3 w-3" />
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
    <Card>
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
      {selected.size > 0 && (
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

      {selected.size > 0 && (
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
  const [chapterId, setChapterId] = useState<string>('all')

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

  const filteredOverdue = useMemo(
    () => (chapterId === 'all' ? overdue : overdue?.filter((i) => i.chapterId === chapterId)) ?? null,
    [overdue, chapterId],
  )
  const filteredRenewal = useMemo(
    () => (chapterId === 'all' ? renewalDue : renewalDue?.filter((m) => m.chapterId === chapterId)) ?? null,
    [renewalDue, chapterId],
  )
  const filteredEligible = useMemo(
    () => (chapterId === 'all' ? eligible : eligible?.filter((m) => m.chapterId === chapterId)) ?? null,
    [eligible, chapterId],
  )

  const totalUrgent =
    (filteredOverdue?.length ?? 0) +
    (filteredRenewal?.length ?? 0) +
    (filteredEligible?.length ?? 0)

  return (
    <div>
      <PageHeader
        title="Perlu Tindakan"
        description="Tagihan overdue, renewal jatuh tempo, dan pendaftaran yang belum diproses."
        action={
          <div className="flex items-center gap-3">
            <Select
              value={chapterId}
              onChange={(e) => setChapterId(e.target.value)}
              className="w-44"
            >
              <option value="all">Semua Chapter</option>
              {chapters?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.displayName}
                </option>
              ))}
            </Select>
            {totalUrgent > 0 && (
              <div className="flex items-center gap-2 rounded-xl bg-red-50 px-4 py-2">
                <AlertTriangle className="h-4 w-4 text-red-500" />
                <span className="text-sm font-semibold text-red-600">{totalUrgent} item</span>
              </div>
            )}
          </div>
        }
      />

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
