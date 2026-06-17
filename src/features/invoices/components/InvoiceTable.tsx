import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowRight, Eye } from 'lucide-react'
import type { InvoiceWithRelations } from '@/types'
import {
  Avatar,
  InvoiceStatusBadge,
  InvoiceTypeBadge,
  Table,
  TBody,
  Td,
  Th,
  THead,
  Tr,
  WhatsAppIcon,
} from '@/components/ui'
import { formatCurrency, formatDate } from '@/lib/format'
import { buildInvoiceWhatsAppUrl } from '@/lib/whatsapp'
import { daysUntil } from '@/lib/date'
import { cn } from '@/lib/cn'

/** Relative due-date hint, e.g. "3 hari lagi" / "terlambat 5 hari". Only for unpaid issued invoices. */
function DueHint({ dueDate, status }: { dueDate: string; status: InvoiceWithRelations['status'] }) {
  if ((status !== 'sent' && status !== 'overdue') || !dueDate) return null
  const d = daysUntil(dueDate)
  const text = d === 0 ? 'jatuh tempo hari ini' : d > 0 ? `${d} hari lagi` : `terlambat ${Math.abs(d)} hari`
  const tone = d < 0 ? 'text-red-500' : d <= 7 ? 'text-amber-600' : 'text-ink-400'
  return <div className={cn('text-xs', tone)}>{text}</div>
}

type SortKey = 'number' | 'member' | 'amount' | 'dueDate' | 'status'

interface InvoiceTableProps {
  invoices: InvoiceWithRelations[]
  /** Hide type/actions columns for tighter dashboard view. */
  compact?: boolean
  selected?: Set<string>
  onSelectChange?: (s: Set<string>) => void
}

/** Opens wa.me in a new tab with a prefilled invoice message. */
function WhatsAppAction({
  url,
  size = 'md',
}: {
  url: string
  size?: 'sm' | 'md'
}) {
  const dim = size === 'sm' ? 'h-7 w-7' : 'h-8 w-8'
  const icon = size === 'sm' ? 'h-[18px] w-[18px]' : 'h-4 w-4'
  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={(e) => e.stopPropagation()}
      className={cn(
        'inline-flex items-center justify-center rounded-lg text-emerald-600 transition-colors hover:bg-emerald-50',
        dim,
      )}
      aria-label="Kirim WhatsApp"
      title="Kirim invoice via WhatsApp"
    >
      <WhatsAppIcon className={icon} />
    </a>
  )
}

export function InvoiceTable({ invoices, compact = false, selected, onSelectChange }: InvoiceTableProps) {
  const selectable = !!onSelectChange
  const navigate = useNavigate()
  const [sortKey, setSortKey] = useState<SortKey>('dueDate')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')

  const sorted = useMemo(() => {
    const copy = [...invoices]
    copy.sort((a, b) => {
      let cmp = 0
      switch (sortKey) {
        case 'number':
          cmp = a.number.localeCompare(b.number)
          break
        case 'member':
          cmp = (a.member?.name ?? '').localeCompare(b.member?.name ?? '')
          break
        case 'amount':
          cmp = a.amount - b.amount
          break
        case 'status':
          cmp = a.status.localeCompare(b.status)
          break
        case 'dueDate':
          cmp = a.dueDate.localeCompare(b.dueDate)
          break
      }
      return sortDir === 'asc' ? cmp : -cmp
    })
    return copy
  }, [invoices, sortKey, sortDir])

  const toggleSort = (key: SortKey) => {
    if (key === sortKey) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  const sortProps = (key: SortKey) => ({
    sortable: true,
    sortDir: sortKey === key ? sortDir : null,
    onSort: () => toggleSort(key),
  })

  const allSelected = selectable && invoices.length > 0 && invoices.every((inv) => selected?.has(inv.id))

  const toggleAll = () => {
    if (!onSelectChange) return
    if (allSelected) {
      const next = new Set(selected)
      invoices.forEach((inv) => next.delete(inv.id))
      onSelectChange(next)
    } else {
      const next = new Set(selected)
      invoices.forEach((inv) => next.add(inv.id))
      onSelectChange(next)
    }
  }

  const toggleOne = (id: string) => {
    if (!onSelectChange || !selected) return
    const next = new Set(selected)
    next.has(id) ? next.delete(id) : next.add(id)
    onSelectChange(next)
  }

  return (
    <>
      {/* Mobile card list */}
      <div className="divide-y divide-ink-100 lg:hidden">
        {selectable && (
          <div className="flex items-center gap-3 px-4 py-2.5 bg-ink-50">
            <input
              type="checkbox"
              checked={!!allSelected}
              onChange={toggleAll}
              className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
            />
            <span className="text-xs text-ink-500">Pilih semua</span>
          </div>
        )}
        {sorted.map((inv) => {
          const waUrl = buildInvoiceWhatsAppUrl(inv)
          return (
            <div
              key={inv.id}
              onClick={() => (selectable ? toggleOne(inv.id) : navigate(`/invoices/${inv.id}`))}
              className={cn(
                'flex items-start gap-3 px-4 py-3.5 active:bg-ink-50',
                selectable && selected?.has(inv.id) ? 'bg-brand-50/50' : '',
              )}
            >
              {selectable && (
                <input
                  type="checkbox"
                  checked={selected?.has(inv.id) ?? false}
                  onChange={() => toggleOne(inv.id)}
                  onClick={(e) => e.stopPropagation()}
                  className="mt-0.5 h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                />
              )}
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
                    <DueHint dueDate={inv.dueDate} status={inv.status} />
                  </div>
                </div>
                <div className="mt-2 flex items-center gap-2">
                  <InvoiceStatusBadge status={inv.status} />
                  <InvoiceTypeBadge type={inv.type} />
                  {waUrl && (
                    <span className="ml-auto">
                      <WhatsAppAction url={waUrl} size="sm" />
                    </span>
                  )}
                </div>
              </div>
              {!selectable && <ArrowRight className="mt-1 h-4 w-4 shrink-0 text-ink-300" />}
            </div>
          )
        })}
      </div>

      {/* Desktop table */}
      <div className="hidden lg:block">
        <Table>
          <THead>
            <Tr>
              {selectable && (
                <Th className="w-10">
                  <input
                    type="checkbox"
                    checked={!!allSelected}
                    onChange={toggleAll}
                    className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                  />
                </Th>
              )}
              <Th {...sortProps('member')}>Member</Th>
              <Th>Chapter</Th>
              {!compact && <Th {...sortProps('number')}>No. Invoice</Th>}
              {!compact && <Th>Tipe</Th>}
              <Th {...sortProps('amount')}>Nominal</Th>
              <Th {...sortProps('status')}>Status</Th>
              <Th {...sortProps('dueDate')}>Jatuh Tempo</Th>
              {!compact && <Th className="text-right">Aksi</Th>}
            </Tr>
          </THead>
          <TBody>
            {sorted.map((inv) => {
              const waUrl = buildInvoiceWhatsAppUrl(inv)
              return (
                <Tr
                  key={inv.id}
                  onClick={() => (selectable ? toggleOne(inv.id) : navigate(`/invoices/${inv.id}`))}
                  className={selectable && selected?.has(inv.id) ? 'bg-brand-50/40' : ''}
                >
                  {selectable && (
                    <Td>
                      <input
                        type="checkbox"
                        checked={selected?.has(inv.id) ?? false}
                        onChange={() => toggleOne(inv.id)}
                        onClick={(e) => e.stopPropagation()}
                        className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                      />
                    </Td>
                  )}
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
                  {!compact && (
                    <Td>
                      <span className="font-mono text-[13px] text-ink-600">{inv.number}</span>
                    </Td>
                  )}
                  {!compact && (
                    <Td>
                      <InvoiceTypeBadge type={inv.type} />
                    </Td>
                  )}
                  <Td className="font-medium text-ink-900">{formatCurrency(inv.amount)}</Td>
                  <Td>
                    <InvoiceStatusBadge status={inv.status} />
                  </Td>
                  <Td className="whitespace-nowrap text-ink-600">
                    <div className="leading-tight">
                      {formatDate(inv.dueDate)}
                      <DueHint dueDate={inv.dueDate} status={inv.status} />
                    </div>
                  </Td>
                  {!compact && (
                    <Td className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        {waUrl && <WhatsAppAction url={waUrl} />}
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            navigate(`/invoices/${inv.id}`)
                          }}
                          className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-ink-400 transition-colors hover:bg-ink-100 hover:text-brand-500"
                          aria-label="Lihat detail"
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                      </div>
                    </Td>
                  )}
                </Tr>
              )
            })}
          </TBody>
        </Table>
      </div>
    </>
  )
}
