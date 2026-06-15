/**
 * In-memory store backing the mock repositories. A single instance holds all
 * collections for the session; repository methods read and mutate it. Swapping
 * to a real backend means replacing the repositories, not this file.
 */

import type {
  AuditLogEntry,
  Chapter,
  FeeSettings,
  Invoice,
  Member,
  Payment,
} from '@/types'
import { buildSeedData, seedChapters, seedFeeSettings } from './seed'

export interface MockStore {
  chapters: Chapter[]
  members: Member[]
  invoices: Invoice[]
  payments: Payment[]
  auditLog: AuditLogEntry[]
  feeSettings: FeeSettings
}

function createStore(): MockStore {
  const { members, invoices, payments, auditLog } = buildSeedData()
  return {
    chapters: [...seedChapters],
    members,
    invoices,
    payments,
    auditLog,
    feeSettings: { ...seedFeeSettings },
  }
}

/** Shared singleton — every repository talks to the same instance. */
export const store: MockStore = createStore()

// --- small helpers shared by the mock repositories ---------------------------

let idCounter = 1000

/** Monotonic id generator (no Math.random — keeps things predictable). */
export function nextId(prefix: string): string {
  idCounter += 1
  return `${prefix}-${idCounter}`
}

/** Simulate network latency so loading states are visible. */
export function delay<T>(value: T, ms = 280): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), ms))
}

export function nowISO(): string {
  return new Date().toISOString()
}
