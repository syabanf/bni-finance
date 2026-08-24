/**
 * Service container — the composition root.
 *
 * The UI imports repositories from HERE, never from a concrete module.
 * `dataSource` picks between the in-memory mock and the HTTP implementations
 * that talk to the Go backend — no page or hook changes either way. The choice
 * is a runtime setting (a button in Pengaturan), not a build-time env var, so a
 * demo can switch without restarting the dev server.
 */

import type {
  AuthRepository,
  ChapterRepository,
  DashboardRepository,
  InvoiceRepository,
  MemberRepository,
  PaymentRepository,
  SettingsRepository,
  ImportRepository,
  RenewalRepository,
  UserRepository,
} from './types'

import { isMockMode } from './dataSource'

import { mockAuthRepository } from './mock/authRepository'
import { mockChapterRepository } from './mock/chapterRepository'
import { mockDashboardRepository } from './mock/dashboardRepository'
import { mockInvoiceRepository } from './mock/invoiceRepository'
import { mockMemberRepository } from './mock/memberRepository'
import { mockPaymentRepository } from './mock/paymentRepository'
import { mockSettingsRepository } from './mock/settingsRepository'
import { mockImportRepository } from './mock/importRepository'
import { mockRenewalRepository } from './mock/renewalRepository'
import { mockUserRepository } from './mock/userRepository'

import { apiAuthRepository } from './api/authRepository'
import { apiChapterRepository } from './api/chapterRepository'
import { apiDashboardRepository } from './api/dashboardRepository'
import { apiInvoiceRepository } from './api/invoiceRepository'
import { apiMemberRepository } from './api/memberRepository'
import { apiPaymentRepository } from './api/paymentRepository'
import { apiSettingsRepository } from './api/settingsRepository'
import { apiImportRepository } from './api/importRepository'
import { apiRenewalRepository } from './api/renewalRepository'
import { apiUserRepository } from './api/userRepository'

const useMock = isMockMode()

interface Services {
  auth: AuthRepository
  chapters: ChapterRepository
  members: MemberRepository
  invoices: InvoiceRepository
  settings: SettingsRepository
  payments: PaymentRepository
  dashboard: DashboardRepository
  users: UserRepository
  renewals: RenewalRepository
  imports: ImportRepository
}

const mockServices: Services = {
  auth: mockAuthRepository,
  chapters: mockChapterRepository,
  members: mockMemberRepository,
  invoices: mockInvoiceRepository,
  settings: mockSettingsRepository,
  payments: mockPaymentRepository,
  dashboard: mockDashboardRepository,
  users: mockUserRepository,
  renewals: mockRenewalRepository,
  imports: mockImportRepository,
}

const apiServices: Services = {
  auth: apiAuthRepository,
  chapters: apiChapterRepository,
  members: apiMemberRepository,
  invoices: apiInvoiceRepository,
  settings: apiSettingsRepository,
  payments: apiPaymentRepository,
  dashboard: apiDashboardRepository,
  users: apiUserRepository,
  renewals: apiRenewalRepository,
  imports: apiImportRepository,
}

export const services: Services = useMock ? mockServices : apiServices

// Convenience named exports
export const {
  auth: authService,
  chapters: chapterService,
  members: memberService,
  invoices: invoiceService,
  settings: settingsService,
  payments: paymentService,
  dashboard: dashboardService,
  users: userService,
  renewals: renewalService,
  imports: importService,
} = services
