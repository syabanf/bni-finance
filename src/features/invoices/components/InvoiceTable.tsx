import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Eye } from 'lucide-react'
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
} from '@/components/ui'
import { formatCurrency, formatDate } from '@/lib/format'

type SortKey = 'number' | 'member' | 'amount' | 'dueDate' | 'status'

interface InvoiceTableProps {
  invoices: InvoiceWithRelations[]
  /** Hide type/actions columns for tighter dashboard view. */
  compact?: boolean
}

export function InvoiceTable({ invoices, compact = false }: InvoiceTableProps) {
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

  return (
    <Table>
      <THead>
        <Tr>
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
        {sorted.map((inv) => (
          <Tr key={inv.id} onClick={() => navigate(`/invoices/${inv.id}`)}>
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
            <Td className="whitespace-nowrap text-ink-600">{formatDate(inv.dueDate)}</Td>
            {!compact && (
              <Td className="text-right">
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
              </Td>
            )}
          </Tr>
        ))}
      </TBody>
    </Table>
  )
}
