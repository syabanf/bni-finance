/**
 * Deterministic seed data for the mock backend.
 *
 * Dates are anchored around "today" = 2026-06-15 (the date in the project
 * plan) so renewal-due / overdue logic produces a realistic spread. The data
 * is generated once and held in memory by the store; the UI can mutate it
 * during a session (create / send / pay / cancel invoices).
 */

import type {
  AuditLogEntry,
  Chapter,
  FeeSettings,
  Invoice,
  InvoiceStatus,
  Member,
  Payment,
} from '@/types'
import { addDays, addYear } from '@/lib/date'

const NOW = '2026-06-15'
const SYNCED_AT = '2026-06-15T07:30:00Z'

// ---------------------------------------------------------------------------
// Chapters
// ---------------------------------------------------------------------------

export const seedChapters: Chapter[] = [
  { id: 'ch-garuda', name: 'garuda', displayName: 'Garuda', areaName: 'Jakarta Pusat', cityName: 'Jakarta', syncedAt: SYNCED_AT },
  { id: 'ch-magnify', name: 'magnify', displayName: 'Magnify', areaName: 'Jakarta Selatan', cityName: 'Jakarta', syncedAt: SYNCED_AT },
  { id: 'ch-amplify', name: 'amplify', displayName: 'Amplify', areaName: 'Bandung Kota', cityName: 'Bandung', syncedAt: SYNCED_AT },
  { id: 'ch-rise', name: 'rise', displayName: 'Rise', areaName: 'Surabaya Timur', cityName: 'Surabaya', syncedAt: SYNCED_AT },
  { id: 'ch-glorify', name: 'glorify', displayName: 'Glorify', areaName: 'Semarang', cityName: 'Semarang', syncedAt: SYNCED_AT },
  { id: 'ch-victory', name: 'victory', displayName: 'Victory', areaName: 'Denpasar', cityName: 'Bali', syncedAt: SYNCED_AT },
]

// ---------------------------------------------------------------------------
// Members — name, chapter, joined date drive the generated invoices below.
// ---------------------------------------------------------------------------

interface MemberSeed {
  name: string
  chapterId: string
  joined: string
  status?: Member['status']
  // controls the generated invoice history
  history: InvoiceStatus[] // ['paid'] = 1 reg invoice paid, ['paid','overdue'] = reg paid + renewal overdue
}

const memberSeeds: MemberSeed[] = [
  // Garuda
  { name: 'Ahmad Wijaya', chapterId: 'ch-garuda', joined: '2025-06-01', history: ['paid'] },
  { name: 'Rina Kusuma', chapterId: 'ch-garuda', joined: '2025-06-05', history: ['paid'] },
  { name: 'Bayu Setiawan', chapterId: 'ch-garuda', joined: '2024-05-12', history: ['paid', 'paid'] },
  { name: 'Lestari Dewi', chapterId: 'ch-garuda', joined: '2026-05-28', history: ['sent'] },
  { name: 'Fajar Ramadhan', chapterId: 'ch-garuda', joined: '2024-06-18', history: ['paid', 'overdue'] },
  { name: 'Putri Anggraini', chapterId: 'ch-garuda', joined: '2026-06-02', history: ['draft'] },

  // Magnify
  { name: 'Siti Nurhaliza', chapterId: 'ch-magnify', joined: '2025-06-15', history: ['paid'] },
  { name: 'Andi Pratama', chapterId: 'ch-magnify', joined: '2024-03-22', history: ['paid', 'paid'] },
  { name: 'Maya Sari', chapterId: 'ch-magnify', joined: '2026-02-10', history: ['paid'] },
  { name: 'Reza Mahendra', chapterId: 'ch-magnify', joined: '2026-05-20', history: ['sent'] },
  { name: 'Indah Permata', chapterId: 'ch-magnify', joined: '2024-06-25', history: ['paid', 'sent'] },
  { name: 'Yoga Pranata', chapterId: 'ch-magnify', joined: '2026-06-08', history: ['draft'] },

  // Amplify
  { name: 'Budi Santoso', chapterId: 'ch-amplify', joined: '2026-06-01', history: ['sent'] },
  { name: 'Citra Lestari', chapterId: 'ch-amplify', joined: '2026-03-05', history: ['paid'] },
  { name: 'Dimas Aryo', chapterId: 'ch-amplify', joined: '2024-04-10', history: ['paid', 'paid'] },
  { name: 'Nadia Safira', chapterId: 'ch-amplify', joined: '2024-05-05', history: ['paid', 'overdue'] },
  { name: 'Galih Nugroho', chapterId: 'ch-amplify', joined: '2026-04-15', history: ['paid'] },
  { name: 'Wulan Maharani', chapterId: 'ch-amplify', joined: '2025-06-20', history: ['paid'] },

  // Rise
  { name: 'Dewi Lestari', chapterId: 'ch-rise', joined: '2026-04-10', history: ['paid'] },
  { name: 'Hadi Susanto', chapterId: 'ch-rise', joined: '2024-02-14', history: ['paid', 'paid'] },
  { name: 'Sinta Permatasari', chapterId: 'ch-rise', joined: '2025-06-12', history: ['paid'] },
  { name: 'Eko Prasetyo', chapterId: 'ch-rise', joined: '2024-06-30', history: ['paid', 'sent'] },
  { name: 'Ratna Juwita', chapterId: 'ch-rise', joined: '2026-06-05', history: ['draft'] },
  { name: 'Bagus Wicaksono', chapterId: 'ch-rise', joined: '2026-05-25', history: ['paid'] },

  // Glorify
  { name: 'Hendra Pratama', chapterId: 'ch-glorify', joined: '2024-05-01', history: ['paid', 'overdue'] },
  { name: 'Vina Oktaviani', chapterId: 'ch-glorify', joined: '2026-05-08', history: ['paid'] },
  { name: 'Rangga Saputra', chapterId: 'ch-glorify', joined: '2026-05-15', history: ['sent'] },
  { name: 'Tari Melati', chapterId: 'ch-glorify', joined: '2024-07-08', history: ['paid', 'paid'] },
  { name: 'Joko Widodo', chapterId: 'ch-glorify', joined: '2026-06-10', history: ['draft'] },
  { name: 'Ayu Lestari', chapterId: 'ch-glorify', joined: '2025-09-02', history: ['paid'] },

  // Victory
  { name: 'Kadek Surya', chapterId: 'ch-victory', joined: '2026-06-03', history: ['paid'] },
  { name: 'Made Ariani', chapterId: 'ch-victory', joined: '2024-03-30', history: ['paid', 'paid'] },
  { name: 'Wayan Gunawan', chapterId: 'ch-victory', joined: '2026-05-22', history: ['sent'] },
  { name: 'Komang Ayu', chapterId: 'ch-victory', joined: '2024-06-20', history: ['paid', 'cancelled'] },
  { name: 'Ketut Sariasih', chapterId: 'ch-victory', joined: '2026-06-12', history: ['draft'] },
  { name: 'Putu Wirawan', chapterId: 'ch-victory', joined: '2026-06-09', history: ['paid'] },
]

// ---------------------------------------------------------------------------
// Fee settings
// ---------------------------------------------------------------------------

export const seedFeeSettings: FeeSettings = {
  id: 'fee-default',
  registrationFee: 1_500_000,
  renewalFee: 1_000_000,
  currency: 'IDR',
  notes: 'Biaya pendaftaran berlaku untuk visitor yang resmi bergabung. Renewal dibayar tahunan.',
  updatedBy: 'admin-national',
  updatedAt: '2026-01-05T03:00:00Z',
  createdAt: '2026-01-05T03:00:00Z',
}

// ---------------------------------------------------------------------------
// Generation: members → invoices → payments → audit log
// ---------------------------------------------------------------------------

function emailFor(name: string): string {
  return name.toLowerCase().replace(/[^a-z]+/g, '.') + '@email.com'
}

function phoneFor(i: number): string {
  return '+62812' + String(34567000 + i * 137).slice(0, 7)
}

interface BuiltData {
  members: Member[]
  invoices: Invoice[]
  payments: Payment[]
  auditLog: AuditLogEntry[]
}

export function buildSeedData(): BuiltData {
  const members: Member[] = []
  const invoices: Invoice[] = []
  const payments: Payment[] = []
  const auditLog: AuditLogEntry[] = []

  let invoiceSeq = 0
  let paymentSeq = 0
  let auditSeq = 0

  memberSeeds.forEach((seed, idx) => {
    const memberId = `m${String(idx + 1).padStart(3, '0')}`
    const member: Member = {
      id: memberId,
      chapterId: seed.chapterId,
      name: seed.name,
      email: emailFor(seed.name),
      phone: phoneFor(idx),
      status: seed.status ?? (seed.history.includes('overdue') ? 'pending' : 'active'),
      joinedDate: seed.joined,
      renewalDate: null,
      syncedAt: SYNCED_AT,
    }
    members.push(member)

    // Walk the member's invoice history. First entry = registration, the rest = renewals.
    let periodStart = seed.joined
    seed.history.forEach((status, hIdx) => {
      invoiceSeq += 1
      const type = hIdx === 0 ? 'registration' : 'renewal'
      const amount = type === 'registration' ? seedFeeSettings.registrationFee : seedFeeSettings.renewalFee
      const periodEnd = addYear(periodStart)
      // Issued shortly before the period starts (registration on join day; renewal ~ when prior ends).
      const dueDate = hIdx === 0 ? periodStart : addDays(periodStart, -2)
      const createdAt = `${dueDate}T02:00:00Z`
      const number = `INV-${dueDate.slice(0, 4)}-${String(invoiceSeq).padStart(3, '0')}`

      const invoiceId = `inv-${String(invoiceSeq).padStart(4, '0')}`
      const sent = status === 'sent' || status === 'paid' || status === 'overdue'
      const paid = status === 'paid'

      const invoice: Invoice = {
        id: invoiceId,
        number,
        memberId,
        chapterId: seed.chapterId,
        type,
        amount,
        currency: 'IDR',
        dueDate,
        periodStart,
        periodEnd,
        status,
        paperIdInvoiceId: sent ? `PPR-${invoiceSeq}${dueDate.slice(2, 4)}` : undefined,
        paperIdInvoiceUrl: sent ? `https://app.paper.id/invoice/PPR-${invoiceSeq}` : undefined,
        paperIdPaymentUrl: sent ? `https://pay.paper.id/PPR-${invoiceSeq}` : undefined,
        paperIdSentAt: sent ? `${addDays(dueDate, 0)}T03:15:00Z` : undefined,
        paidAt: paid ? `${addDays(dueDate, 4)}T08:20:00Z` : undefined,
        paidAmount: paid ? amount : undefined,
        notes: undefined,
        createdBy: 'admin-national',
        cancelledBy: status === 'cancelled' ? 'admin-national' : undefined,
        cancelledAt: status === 'cancelled' ? `${addDays(dueDate, 3)}T05:00:00Z` : undefined,
        cancelReason: status === 'cancelled' ? 'Member mengundurkan diri sebelum pembayaran.' : undefined,
        createdAt,
        updatedAt: paid ? `${addDays(dueDate, 4)}T08:20:00Z` : createdAt,
      }
      invoices.push(invoice)

      // Audit trail
      const pushAudit = (action: AuditLogEntry['action'], oldS: InvoiceStatus | undefined, newS: InvoiceStatus, at: string, notes?: string) => {
        auditSeq += 1
        auditLog.push({
          id: `aud-${String(auditSeq).padStart(4, '0')}`,
          invoiceId,
          action,
          oldStatus: oldS,
          newStatus: newS,
          actorId: 'admin-national',
          actorName: 'Admin Nasional',
          notes,
          createdAt: at,
        })
      }
      pushAudit('created', undefined, 'draft', createdAt, `Invoice ${type} dibuat`)
      if (sent) pushAudit('sent', 'draft', 'sent', `${dueDate}T03:15:00Z`, 'Dikirim ke Paper.id')
      if (paid) pushAudit('paid', 'sent', 'paid', invoice.paidAt!, 'Pembayaran diterima via Paper.id')
      if (status === 'overdue') pushAudit('overdue', 'sent', 'overdue', addDays(dueDate, 32) + 'T00:05:00Z', 'Jatuh tempo terlewati')
      if (status === 'cancelled') pushAudit('cancelled', 'draft', 'cancelled', invoice.cancelledAt!, invoice.cancelReason)

      // Payment record for paid invoices
      if (paid) {
        paymentSeq += 1
        payments.push({
          id: `pay-${String(paymentSeq).padStart(4, '0')}`,
          invoiceId,
          amount,
          paidAt: invoice.paidAt!,
          paymentMethod: ['virtual_account', 'bank_transfer', 'qris'][invoiceSeq % 3],
          paperIdPaymentId: `PAY-${invoiceSeq}${dueDate.slice(2, 4)}`,
          paperIdStatus: 'success',
          createdAt: invoice.paidAt!,
        })
      }

      // Next renewal period begins the day after the current one ends.
      periodStart = addDays(periodEnd, 1)
    })
  })

  return { members, invoices, payments, auditLog }
}

export { NOW }
