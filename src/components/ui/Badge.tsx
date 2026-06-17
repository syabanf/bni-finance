import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'
import type { InvoiceStatus, InvoiceType, MemberStatus } from '@/types'
import { INVOICE_STATUS_LABEL } from '@/lib/status'

type Tone = 'green' | 'red' | 'amber' | 'blue' | 'gray' | 'purple'

const tones: Record<Tone, { wrap: string; dot: string }> = {
  green: { wrap: 'bg-emerald-50 text-emerald-700 ring-emerald-600/10', dot: 'bg-emerald-500' },
  red: { wrap: 'bg-red-50 text-red-600 ring-red-600/10', dot: 'bg-red-500' },
  amber: { wrap: 'bg-amber-50 text-amber-700 ring-amber-600/10', dot: 'bg-amber-500' },
  blue: { wrap: 'bg-blue-50 text-blue-700 ring-blue-600/10', dot: 'bg-blue-500' },
  gray: { wrap: 'bg-ink-100 text-ink-600 ring-ink-500/10', dot: 'bg-ink-400' },
  purple: { wrap: 'bg-violet-50 text-violet-700 ring-violet-600/10', dot: 'bg-violet-500' },
}

interface BadgeProps {
  tone?: Tone
  dot?: boolean
  children: ReactNode
  className?: string
}

export function Badge({ tone = 'gray', dot = true, children, className }: BadgeProps) {
  const t = tones[tone]
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset',
        t.wrap,
        className,
      )}
    >
      {dot && <span className={cn('h-1.5 w-1.5 rounded-full', t.dot)} />}
      {children}
    </span>
  )
}

// --- Domain-specific badges -------------------------------------------------

const INVOICE_STATUS_TONE: Record<InvoiceStatus, Tone> = {
  draft: 'gray',
  sent: 'amber',
  paid: 'green',
  overdue: 'red',
  cancelled: 'gray',
}

export function InvoiceStatusBadge({ status }: { status: InvoiceStatus }) {
  return <Badge tone={INVOICE_STATUS_TONE[status]}>{INVOICE_STATUS_LABEL[status]}</Badge>
}

const MEMBER_STATUS: Record<MemberStatus, { tone: Tone; label: string }> = {
  active: { tone: 'green', label: 'Active' },
  inactive: { tone: 'gray', label: 'Inactive' },
  pending: { tone: 'amber', label: 'Pending' },
}

export function MemberStatusBadge({ status }: { status: MemberStatus }) {
  const { tone, label } = MEMBER_STATUS[status]
  return <Badge tone={tone}>{label}</Badge>
}

const INVOICE_TYPE: Record<InvoiceType, { tone: Tone; label: string }> = {
  registration: { tone: 'blue', label: 'Pendaftaran' },
  renewal: { tone: 'purple', label: 'Renewal' },
}

export function InvoiceTypeBadge({ type }: { type: InvoiceType }) {
  const { tone, label } = INVOICE_TYPE[type]
  return (
    <Badge tone={tone} dot={false}>
      {label}
    </Badge>
  )
}
