/**
 * Formatting helpers — currency, dates, and number shorthands.
 * Locale defaults to Indonesian (id-ID) to match the product audience.
 */

const ID_MONTHS_SHORT = [
  'Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun',
  'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des',
]

/** Rp 500.000 */
export function formatCurrency(value: number, currency = 'IDR'): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

/** Compact form for KPI cards: Rp 12,5 jt / Rp 1,2 M */
export function formatCurrencyCompact(value: number): string {
  if (value >= 1_000_000_000) return `Rp ${(value / 1_000_000_000).toFixed(1).replace('.', ',')} M`
  if (value >= 1_000_000) return `Rp ${(value / 1_000_000).toFixed(1).replace('.', ',')} jt`
  if (value >= 1_000) return `Rp ${(value / 1_000).toFixed(0)} rb`
  return formatCurrency(value)
}

/** 15 Jun 2024 */
export function formatDate(iso: string | Date | null | undefined): string {
  if (iso == null) return '—'
  const d = typeof iso === 'string' ? new Date(iso) : iso
  if (Number.isNaN(d.getTime())) return '—'
  return `${d.getDate()} ${ID_MONTHS_SHORT[d.getMonth()]} ${d.getFullYear()}`
}

/** 15 Jun 2024, 10:30 */
export function formatDateTime(iso: string | Date | null | undefined): string {
  if (iso == null) return '—'
  const d = typeof iso === 'string' ? new Date(iso) : iso
  if (Number.isNaN(d.getTime())) return '—'
  const time = d.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
  return `${formatDate(d)}, ${time}`
}

/** First-letter avatar fallback. */
export function initials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length === 1) return parts[0].slice(0, 1).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}
