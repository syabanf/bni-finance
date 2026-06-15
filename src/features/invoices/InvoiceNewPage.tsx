import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, Check, Search, Send, UserRound } from 'lucide-react'
import type { FeeSettings, InvoiceType, MemberWithChapter } from '@/types'
import {
  Avatar,
  Button,
  Card,
  CardHeader,
  Field,
  Input,
  LoadingState,
  PageHeader,
  Textarea,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { invoiceService, memberService, settingsService } from '@/services'
import { addDays, addYear, todayISO } from '@/lib/date'
import { formatCurrency, formatDate } from '@/lib/format'
import { cn } from '@/lib/cn'

export function InvoiceNewPage() {
  const navigate = useNavigate()
  const { toast } = useToast()

  const { data: fees } = useAsync<FeeSettings>(() => settingsService.getFees())

  const [type, setType] = useState<InvoiceType>('registration')
  const { data: members, loading: membersLoading } = useAsync<MemberWithChapter[]>(
    () => (type === 'registration' ? memberService.eligibleForRegistration() : memberService.list()),
    [type],
  )

  const [memberId, setMemberId] = useState<string>('')
  const [search, setSearch] = useState('')
  const [amount, setAmount] = useState<number>(0)
  const [periodStart, setPeriodStart] = useState<string>(todayISO())
  const [notes, setNotes] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const selectedMember = members?.find((m) => m.id === memberId) ?? null
  const dueDate = todayISO()
  const periodEnd = useMemo(() => addYear(periodStart), [periodStart])

  // Auto-fill the amount from fee settings whenever type/fees change.
  useEffect(() => {
    if (!fees) return
    setAmount(type === 'registration' ? fees.registrationFee : fees.renewalFee)
  }, [fees, type])

  // Compute the membership period start (renewal continues the previous period).
  useEffect(() => {
    let active = true
    if (type === 'registration' || !memberId) {
      setPeriodStart(todayISO())
      return
    }
    invoiceService.listByMember(memberId).then((list) => {
      if (!active) return
      const last = list
        .filter((i) => i.status !== 'cancelled')
        .sort((a, b) => b.periodEnd.localeCompare(a.periodEnd))[0]
      setPeriodStart(last ? addDays(last.periodEnd, 1) : todayISO())
    })
    return () => {
      active = false
    }
  }, [type, memberId])

  const filteredMembers = useMemo(() => {
    if (!members) return []
    const q = search.trim().toLowerCase()
    if (!q) return members
    return members.filter(
      (m) => m.name.toLowerCase().includes(q) || m.id.toLowerCase().includes(q),
    )
  }, [members, search])

  const handleSubmit = async (send: boolean) => {
    if (!memberId) {
      toast('Pilih member terlebih dahulu.', 'error')
      return
    }
    setSubmitting(true)
    try {
      const invoice = await invoiceService.create({
        memberId,
        type,
        amount,
        dueDate,
        periodStart,
        periodEnd,
        notes: notes.trim() || undefined,
      })
      if (send) {
        await invoiceService.send(invoice.id)
        toast('Invoice dibuat & dikirim ke Paper.id.')
      } else {
        toast('Invoice draft berhasil dibuat.')
      }
      navigate(`/invoices/${invoice.id}`)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal membuat invoice.', 'error')
      setSubmitting(false)
    }
  }

  return (
    <div>
      <PageHeader
        title="Buat Invoice"
        description="Terbitkan invoice pendaftaran atau renewal untuk member."
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

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-5">
        {/* Left: type + member picker */}
        <div className="space-y-5 lg:col-span-3">
          <Card>
            <CardHeader title="1. Tipe Invoice" />
            <div className="grid grid-cols-2 gap-3 px-5 pb-5">
              {(['registration', 'renewal'] as InvoiceType[]).map((t) => (
                <button
                  key={t}
                  onClick={() => {
                    setType(t)
                    setMemberId('')
                  }}
                  className={cn(
                    'rounded-xl border-2 p-4 text-left transition-colors',
                    type === t
                      ? 'border-brand-500 bg-brand-50/50'
                      : 'border-ink-200 hover:border-ink-300',
                  )}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-ink-900">
                      {t === 'registration' ? 'Pendaftaran' : 'Renewal'}
                    </span>
                    {type === t && (
                      <span className="flex h-5 w-5 items-center justify-center rounded-full bg-brand-500 text-white">
                        <Check className="h-3 w-3" />
                      </span>
                    )}
                  </div>
                  <p className="mt-1 text-xs text-ink-500">
                    {t === 'registration'
                      ? 'Visitor resmi bergabung jadi member.'
                      : 'Perpanjangan keanggotaan tahunan.'}
                  </p>
                </button>
              ))}
            </div>
          </Card>

          <Card>
            <CardHeader
              title="2. Pilih Member"
              subtitle={
                type === 'registration'
                  ? 'Member yang belum punya invoice pendaftaran aktif.'
                  : 'Pilih member untuk diperpanjang.'
              }
            />
            <div className="px-5 pb-3">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
                <Input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Cari nama atau ID member…"
                  className="pl-10"
                />
              </div>
            </div>
            <div className="max-h-[360px] overflow-y-auto px-3 pb-3">
              {membersLoading ? (
                <LoadingState />
              ) : filteredMembers.length === 0 ? (
                <p className="px-2 py-8 text-center text-sm text-ink-400">
                  Tidak ada member yang cocok.
                </p>
              ) : (
                <div className="space-y-1">
                  {filteredMembers.map((m) => (
                    <button
                      key={m.id}
                      onClick={() => setMemberId(m.id)}
                      className={cn(
                        'flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left transition-colors',
                        memberId === m.id ? 'bg-brand-50 ring-1 ring-brand-200' : 'hover:bg-ink-50',
                      )}
                    >
                      <Avatar name={m.name} size="sm" />
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-medium text-ink-900">{m.name}</div>
                        <div className="text-xs text-ink-400">
                          {m.id} · {m.chapter?.displayName ?? '—'}
                        </div>
                      </div>
                      {memberId === m.id && (
                        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-brand-500 text-white">
                          <Check className="h-3 w-3" />
                        </span>
                      )}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </Card>
        </div>

        {/* Right: review & submit */}
        <div className="lg:col-span-2">
          <Card className="sticky top-20">
            <CardHeader title="3. Ringkasan Invoice" />
            <div className="space-y-4 px-5 pb-5">
              {selectedMember ? (
                <div className="flex items-center gap-3 rounded-xl bg-ink-50 p-3">
                  <Avatar name={selectedMember.name} />
                  <div className="min-w-0">
                    <div className="truncate font-medium text-ink-900">{selectedMember.name}</div>
                    <div className="text-xs text-ink-400">
                      {selectedMember.chapter?.displayName ?? '—'}
                    </div>
                  </div>
                </div>
              ) : (
                <div className="flex items-center gap-3 rounded-xl border border-dashed border-ink-200 p-3 text-sm text-ink-400">
                  <UserRound className="h-5 w-5" />
                  Belum ada member dipilih
                </div>
              )}

              <Field label="Nominal (Rp)">
                <Input
                  type="number"
                  value={amount}
                  onChange={(e) => setAmount(Number(e.target.value))}
                  min={0}
                  step={50000}
                />
              </Field>

              <div className="space-y-2.5 rounded-xl bg-ink-50 p-4 text-sm">
                <Row label="Tanggal terbit" value={formatDate(dueDate)} />
                <Row label="Masa berlaku" value={`${formatDate(periodStart)} → ${formatDate(periodEnd)}`} />
                <Row label="Tipe" value={type === 'registration' ? 'Pendaftaran' : 'Renewal'} />
                <div className="my-1 border-t border-ink-200" />
                <Row label="Total" value={formatCurrency(amount)} bold />
              </div>

              <Field label="Catatan (opsional)">
                <Textarea
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="Catatan internal untuk invoice ini…"
                />
              </Field>

              <div className="space-y-2 pt-1">
                <Button
                  className="w-full"
                  loading={submitting}
                  disabled={!memberId}
                  onClick={() => handleSubmit(true)}
                >
                  <Send className="h-4 w-4" />
                  Buat & Kirim ke Paper.id
                </Button>
                <Button
                  variant="outline"
                  className="w-full"
                  disabled={submitting || !memberId}
                  onClick={() => handleSubmit(false)}
                >
                  Simpan sebagai Draft
                </Button>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}

function Row({ label, value, bold }: { label: string; value: string; bold?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-ink-500">{label}</span>
      <span className={cn('text-right', bold ? 'text-base font-bold text-ink-900' : 'font-medium text-ink-700')}>
        {value}
      </span>
    </div>
  )
}
