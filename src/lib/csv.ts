// Build the UTF-8 BOM at runtime — a literal U+FEFF gets stripped by the bundler.
const BOM = String.fromCharCode(0xfeff)

/**
 * Escape one CSV cell. Numbers are emitted unquoted; strings are quoted, and
 * any string starting with = + - @ (or tab/CR) is prefixed with `'` so Excel /
 * Sheets don't evaluate it as a formula (CSV-injection guard — member names are
 * externally sourced from BNI VM).
 */
function escapeCell(v: string | number): string {
  if (typeof v === 'number') return Number.isFinite(v) ? String(v) : ''
  let s = String(v)
  if (/^[=+\-@\t\r]/.test(s)) s = "'" + s
  return `"${s.replace(/"/g, '""')}"`
}

/** Trigger a client-side CSV download from headers + rows. */
export function downloadCsv(filename: string, headers: string[], rows: (string | number)[][]) {
  const lines = [headers, ...rows].map((r) => r.map(escapeCell).join(','))
  // Prepend a BOM so Excel reads UTF-8 (Rp, accented names) correctly.
  const blob = new Blob([BOM + lines.join('\n')], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
