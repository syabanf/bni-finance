/**
 * Domain models for the BNI Finance System.
 *
 * These mirror the Supabase schema described in the technical plan
 * (section 4). The presentation layer depends only on these types and the
 * repository interfaces — never on a concrete data source — so the mock
 * backend can be swapped for the real Supabase/Paper.id/BNI-VM APIs without
 * touching any UI code.
 */

// ---------------------------------------------------------------------------
// Enums (mirror the Postgres ENUM types)
// ---------------------------------------------------------------------------

export type InvoiceType = 'registration' | 'renewal'

export type InvoiceStatus = 'draft' | 'sent' | 'paid' | 'overdue' | 'cancelled'

export type MemberStatus = 'active' | 'inactive' | 'pending'

// ---------------------------------------------------------------------------
// Synced entities (read-only mirrors from BNI Visitor Management)
// ---------------------------------------------------------------------------

export interface Chapter {
  id: string
  name: string
  displayName: string
  areaName?: string
  cityName?: string
  syncedAt: string
}

export interface Member {
  id: string
  chapterId: string
  name: string
  email?: string
  phone?: string
  status: MemberStatus
  joinedDate: string // ISO date
  syncedAt: string
}

/** Member enriched with its chapter — convenience for list/detail views. */
export interface MemberWithChapter extends Member {
  chapter: Chapter | null
}

// ---------------------------------------------------------------------------
// Fee configuration
// ---------------------------------------------------------------------------

export interface FeeSettings {
  id: string
  registrationFee: number
  renewalFee: number
  currency: string
  notes?: string
  updatedBy?: string
  updatedAt: string
  createdAt: string
}

// ---------------------------------------------------------------------------
// Invoice
// ---------------------------------------------------------------------------

export interface Invoice {
  id: string
  /** Human-friendly number, e.g. INV-2026-001 */
  number: string

  memberId: string
  chapterId: string

  type: InvoiceType
  amount: number
  currency: string

  dueDate: string // ISO date — = tanggal invoice diterbitkan
  periodStart: string // awal masa berlaku keanggotaan
  periodEnd: string // period_start + 1 tahun

  status: InvoiceStatus

  // Paper.id integration fields
  paperIdInvoiceId?: string
  paperIdInvoiceUrl?: string
  paperIdPaymentUrl?: string
  paperIdSentAt?: string

  // Payment
  paidAt?: string
  paidAmount?: number

  // Metadata
  notes?: string
  createdBy?: string
  cancelledBy?: string
  cancelledAt?: string
  cancelReason?: string

  createdAt: string
  updatedAt: string
}

/** Invoice joined with member & chapter — used by tables and detail pages. */
export interface InvoiceWithRelations extends Invoice {
  member: Member | null
  chapter: Chapter | null
}

// ---------------------------------------------------------------------------
// Payment (recorded from a Paper.id webhook)
// ---------------------------------------------------------------------------

export interface Payment {
  id: string
  invoiceId: string
  amount: number
  paidAt: string
  paymentMethod?: string
  paperIdPaymentId?: string
  paperIdStatus?: string
  createdAt: string
}

export interface PaymentWithInvoice extends Payment {
  invoice: Invoice | null
  member: Member | null
}

// ---------------------------------------------------------------------------
// Audit log
// ---------------------------------------------------------------------------

export type AuditAction =
  | 'created'
  | 'sent'
  | 'paid'
  | 'cancelled'
  | 'overdue'
  | 'updated'

export interface AuditLogEntry {
  id: string
  invoiceId: string
  action: AuditAction
  oldStatus?: InvoiceStatus
  newStatus?: InvoiceStatus
  actorId?: string
  actorName?: string
  notes?: string
  createdAt: string
}

// ---------------------------------------------------------------------------
// Dashboard aggregates
// ---------------------------------------------------------------------------

export interface ChapterStat {
  chapterId: string
  chapterName: string
  total: number
  paid: number
  outstanding: number
  overdue: number
  totalAmount: number
}

export interface DashboardSummary {
  /** All invoices (excluding cancelled) — count + total value. */
  total: { count: number; amount: number; trend: number }
  /** All paid invoices — count + collected revenue. */
  paid: { count: number; amount: number; trend: number }
  /** Issued but unpaid (sent + overdue). */
  outstanding: { count: number; amount: number; trend: number }
  overdue: { count: number; amount: number; trend: number }
  renewalDue: { count: number; trend: number }
  /** Breakdown for the payment-status donut chart. */
  statusBreakdown: { status: InvoiceStatus; count: number }[]
  /** Last 6 months of issued vs. paid totals. */
  monthly: { month: string; issued: number; paid: number }[]
  /** Per-chapter breakdown. */
  chapterStats: ChapterStat[]
}

// ---------------------------------------------------------------------------
// Shared query / pagination shapes
// ---------------------------------------------------------------------------

export interface InvoiceFilters {
  status?: InvoiceStatus | 'all'
  type?: InvoiceType | 'all'
  chapterId?: string | 'all'
  search?: string
}

export interface RenewalDueMember extends MemberWithChapter {
  /** The most recent invoice whose period is ending. */
  lastInvoice: Invoice
  daysUntilDue: number
}

export interface AuthUser {
  id: string
  name: string
  email: string
  role: 'national_admin'
}
