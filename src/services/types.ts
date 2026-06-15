/**
 * Repository contracts — the boundary between the presentation layer and the
 * data source. Pages and hooks depend ONLY on these interfaces. Concrete
 * implementations live under `services/mock` today and can be replaced with
 * Supabase/HTTP implementations later (see `services/index.ts`).
 *
 * Everything is async so swapping to a network-backed source is a no-op for
 * callers.
 */

import type {
  AuditLogEntry,
  AuthUser,
  Chapter,
  DashboardSummary,
  FeeSettings,
  Invoice,
  InvoiceFilters,
  InvoiceType,
  InvoiceWithRelations,
  MemberWithChapter,
  PaymentWithInvoice,
  RenewalDueMember,
} from '@/types'

export interface AuthRepository {
  login(email: string, password: string): Promise<AuthUser>
  logout(): Promise<void>
  getCurrentUser(): AuthUser | null
}

export interface ChapterRepository {
  list(): Promise<Chapter[]>
  getById(id: string): Promise<Chapter | null>
  /** Pull fresh data from BNI VM and refresh the local mirror. */
  sync(): Promise<{ count: number; syncedAt: string }>
}

export interface MemberRepository {
  list(params?: { chapterId?: string; search?: string }): Promise<MemberWithChapter[]>
  getById(id: string): Promise<MemberWithChapter | null>
  /** Members eligible for a new registration invoice (no active one yet). */
  eligibleForRegistration(): Promise<MemberWithChapter[]>
  sync(): Promise<{ count: number; syncedAt: string }>
}

export interface InvoiceRepository {
  list(filters?: InvoiceFilters): Promise<InvoiceWithRelations[]>
  getById(id: string): Promise<InvoiceWithRelations | null>
  listByMember(memberId: string): Promise<Invoice[]>
  create(input: CreateInvoiceInput): Promise<Invoice>
  /** Push to Paper.id and move the invoice to `sent`. */
  send(id: string): Promise<Invoice>
  /** Re-send an already-sent/overdue invoice to Paper.id (refreshes payment link). */
  resend(id: string): Promise<Invoice>
  cancel(id: string, reason: string): Promise<Invoice>
  /** Simulate a Paper.id "payment.success" webhook for a sent invoice. */
  markPaid(id: string): Promise<Invoice>
  getAuditLog(invoiceId: string): Promise<AuditLogEntry[]>
  /** Members at/near the end of their membership period. */
  renewalDue(withinDays?: number): Promise<RenewalDueMember[]>
}

export interface CreateInvoiceInput {
  memberId: string
  type: InvoiceType
  amount: number
  dueDate: string
  periodStart: string
  periodEnd: string
  notes?: string
}

export interface SettingsRepository {
  getFees(): Promise<FeeSettings>
  updateFees(input: Pick<FeeSettings, 'registrationFee' | 'renewalFee' | 'notes'>): Promise<FeeSettings>
}

export interface PaymentRepository {
  list(): Promise<PaymentWithInvoice[]>
  listByInvoice(invoiceId: string): Promise<PaymentWithInvoice[]>
}

export interface UrgentCount {
  overdue: number
  renewalDue: number
  total: number
}

export interface DashboardRepository {
  summary(): Promise<DashboardSummary>
}
