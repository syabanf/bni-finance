/**
 * Service container — the composition root.
 *
 * The UI imports repositories from HERE, never from a concrete module. Today
 * everything resolves to the in-memory mock implementations. When the real
 * backend is ready, gate on `VITE_USE_MOCK` and return Supabase/HTTP-backed
 * implementations instead — no page or hook needs to change.
 */

import type {
  AuthRepository,
  ChapterRepository,
  DashboardRepository,
  InvoiceRepository,
  MemberRepository,
  PaymentRepository,
  SettingsRepository,
} from './types'

import { mockAuthRepository } from './mock/authRepository'
import { mockChapterRepository } from './mock/chapterRepository'
import { mockDashboardRepository } from './mock/dashboardRepository'
import { mockInvoiceRepository } from './mock/invoiceRepository'
import { mockMemberRepository } from './mock/memberRepository'
import { mockPaymentRepository } from './mock/paymentRepository'
import { mockSettingsRepository } from './mock/settingsRepository'

import { supabaseAuthRepository } from './supabase/authRepository'
import { supabaseChapterRepository } from './supabase/chapterRepository'
import { supabaseDashboardRepository } from './supabase/dashboardRepository'
import { supabaseInvoiceRepository } from './supabase/invoiceRepository'
import { supabaseMemberRepository } from './supabase/memberRepository'
import { supabasePaymentRepository } from './supabase/paymentRepository'
import { supabaseSettingsRepository } from './supabase/settingsRepository'

const useMock = import.meta.env.VITE_USE_MOCK !== 'false'

interface Services {
  auth: AuthRepository
  chapters: ChapterRepository
  members: MemberRepository
  invoices: InvoiceRepository
  settings: SettingsRepository
  payments: PaymentRepository
  dashboard: DashboardRepository
}

const mockServices: Services = {
  auth: mockAuthRepository,
  chapters: mockChapterRepository,
  members: mockMemberRepository,
  invoices: mockInvoiceRepository,
  settings: mockSettingsRepository,
  payments: mockPaymentRepository,
  dashboard: mockDashboardRepository,
}

const supabaseServices: Services = {
  auth: supabaseAuthRepository,
  chapters: supabaseChapterRepository,
  members: supabaseMemberRepository,
  invoices: supabaseInvoiceRepository,
  settings: supabaseSettingsRepository,
  payments: supabasePaymentRepository,
  dashboard: supabaseDashboardRepository,
}

export const services: Services = useMock ? mockServices : supabaseServices

// Convenience named exports
export const {
  auth: authService,
  chapters: chapterService,
  members: memberService,
  invoices: invoiceService,
  settings: settingsService,
  payments: paymentService,
  dashboard: dashboardService,
} = services
