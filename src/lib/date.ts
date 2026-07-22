/** Date math used by the invoice business logic (membership periods). */

const pad2 = (n: number) => String(n).padStart(2, '0')

/**
 * Today as an ISO date string (yyyy-mm-dd) in LOCAL time.
 * Using toISOString() here would return the UTC date, which is a day behind
 * for the whole 00:00–07:00 window in WIB (UTC+7) — throwing off "this month"
 * and the report period presets. daysUntil() also compares in local time.
 */
export function todayISO(): string {
  const d = new Date()
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`
}

/** Current year-month (yyyy-mm) in local time. */
export function monthNowKey(): string {
  const d = new Date()
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}`
}

/** Add a number of days to an ISO date, returning a new ISO date string. */
export function addDays(iso: string, days: number): string {
  const d = new Date(iso)
  d.setDate(d.getDate() + days)
  return d.toISOString().slice(0, 10)
}

/** Membership runs one year (365 days) from the start. */
export function addYear(iso: string): string {
  return addDays(iso, 365)
}

/** Whole days from today until `iso` (negative = in the past). */
export function daysUntil(iso: string): number {
  const target = new Date(iso).setHours(0, 0, 0, 0)
  const now = new Date().setHours(0, 0, 0, 0)
  return Math.round((target - now) / 86_400_000)
}

/** yyyy-mm key for grouping by month. */
export function monthKey(iso: string): string {
  return iso.slice(0, 7)
}
