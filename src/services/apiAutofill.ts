/**
 * Fills a console request with values that actually exist.
 *
 * Typing a UUID by hand is the slowest part of using the console, and a wrong
 * one only ever produces a 404 — which teaches you nothing about the endpoint.
 * So autofill looks the values up through the SAME transport the request will
 * use: mock mode fills from the mock store, API mode from the live database.
 *
 * It also respects each endpoint's precondition. `POST /invoices/{id}/send`
 * needs a DRAFT invoice and `POST /paperid/test-callback` needs one already
 * sent; handing either the newest invoice would usually fail with a 409.
 */

import { sendConsoleRequest, type ConsoleOperation } from './apiConsole'

export interface AutofillResult {
  pathValues: Record<string, string>
  queryValues: Record<string, string>
  /** Replacement body JSON, or undefined to leave the editor alone. */
  body?: string
  /** Human-readable account of what was filled and what could not be. */
  notes: string[]
}

type Row = Record<string, unknown>

/** GETs a list endpoint and returns its first row, or null. */
async function first(path: string, queryValues: Record<string, string> = {}): Promise<Row | null> {
  try {
    const res = await sendConsoleRequest({ method: 'GET', path, queryValues: { limit: '1', ...queryValues } })
    if (!res.ok) return null
    const data = (res.body as { data?: Row[] } | null)?.data
    return Array.isArray(data) && data.length > 0 ? data[0] : null
  } catch {
    return null
  }
}

/**
 * Where each `{param}` comes from. First match wins, so the endpoint-specific
 * rows must precede the generic ones.
 */
const PATH_SOURCES: {
  match: RegExp
  param: string
  list: string
  query?: Record<string, string>
  field?: string
  label: string
}[] = [
  // Preconditions first — these need a particular STATE, not just any row.
  {
    match: /\/invoices\/\{id\}\/send$/,
    param: 'id',
    list: '/api/v1/invoices',
    query: { status: 'draft' },
    label: 'invoice berstatus draft',
  },
  {
    match: /\/invoices\/\{id\}\/audit$/,
    param: 'id',
    list: '/api/v1/invoices',
    label: 'invoice mana pun',
  },
  { match: /\/app-settings\/\{key\}$/, param: 'key', list: '/api/v1/app-settings', field: 'key', label: 'kunci pengaturan' },
  { match: /\/public\/invoices\/\{id\}/, param: 'id', list: '/api/v1/invoices', query: { status: 'sent' }, label: 'invoice terkirim' },
  // Generic resource lookups.
  { match: /\/invoices\/\{id\}$/, param: 'id', list: '/api/v1/invoices', label: 'invoice' },
  { match: /\/payments\/\{id\}$/, param: 'id', list: '/api/v1/payments', label: 'pembayaran' },
  { match: /\/members\/\{id\}$/, param: 'id', list: '/api/v1/members', label: 'member' },
  { match: /\/chapters\/\{id\}$/, param: 'id', list: '/api/v1/chapters', label: 'chapter' },
  { match: /\/users\/\{id\}/, param: 'id', list: '/api/v1/users', label: 'pengguna' },
]

function isoDate(offsetDays = 0): string {
  const d = new Date()
  d.setDate(d.getDate() + offsetDays)
  return d.toISOString().slice(0, 10)
}

/**
 * Body fields that must point at something real. Anything not listed keeps the
 * sample value from the spec — autofill supplies identity and dates, not
 * business decisions like amounts or notes.
 */
async function fillBody(op: ConsoleOperation, notes: string[]): Promise<string | undefined> {
  if (op.body === undefined || typeof op.body !== 'object' || op.body === null) return undefined
  const body = { ...(op.body as Row) }
  const keys = new Set(Object.keys(body))

  // A member and its chapter are resolved together: pairing a member with some
  // other chapter's id would create a record that contradicts itself.
  if (keys.has('memberId') || keys.has('chapterId')) {
    const member = await first('/api/v1/members', { status: 'active' })
    if (member) {
      if (keys.has('memberId')) body.memberId = member.id
      if (keys.has('chapterId')) body.chapterId = member.chapterId
      notes.push(`member "${String(member.name)}" beserta chapternya`)
    } else if (keys.has('chapterId')) {
      const chapter = await first('/api/v1/chapters')
      if (chapter) {
        body.chapterId = chapter.id
        notes.push(`chapter "${String(chapter.displayName ?? chapter.name)}"`)
      }
    }
  }

  if (keys.has('invoiceId')) {
    // A payment against a cancelled invoice is rejected, so ask for one that
    // is still outstanding.
    const invoice = (await first('/api/v1/invoices', { status: 'outstanding' })) ?? (await first('/api/v1/invoices'))
    if (invoice) {
      body.invoiceId = invoice.id
      // Paying the exact amount is the case worth testing by default.
      if (keys.has('amount') && typeof invoice.amount === 'number') body.amount = invoice.amount
      notes.push(`invoice ${String(invoice.number)}`)
    } else {
      notes.push('tidak ada invoice untuk dirujuk')
    }
  }

  if (keys.has('dueDate')) body.dueDate = isoDate(30)
  if (keys.has('periodStart')) body.periodStart = isoDate(0)
  if (keys.has('periodEnd')) body.periodEnd = isoDate(365)
  if (keys.has('paidAt')) body.paidAt = new Date().toISOString()

  return JSON.stringify(body, null, 2)
}

export async function autofill(op: ConsoleOperation): Promise<AutofillResult> {
  const notes: string[] = []
  const pathValues: Record<string, string> = {}
  const queryValues: Record<string, string> = {}

  for (const param of op.params) {
    if (param.in !== 'path') continue

    const source = PATH_SOURCES.find((s) => s.match.test(op.path) && s.param === param.name)
    if (!source) {
      notes.push(`${param.name}: tidak ada sumber otomatis`)
      continue
    }

    const row = await first(source.list, source.query)
    const value = row?.[source.field ?? 'id']
    if (value === undefined || value === null) {
      notes.push(`${param.name}: tidak ditemukan ${source.label}`)
      continue
    }
    pathValues[param.name] = String(value)
    notes.push(`${param.name} ← ${source.label}`)
  }

  // Only required query params are filled; the rest are optional filters that
  // the caller is better off choosing.
  for (const param of op.params) {
    if (param.in !== 'query' || !param.required) continue
    if (param.default !== undefined) queryValues[param.name] = String(param.default)
  }

  const body = await fillBody(op, notes)

  return { pathValues, queryValues, body, notes }
}
