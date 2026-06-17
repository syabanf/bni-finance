// Build the UTF-8 BOM at runtime — a literal U+FEFF gets stripped by the bundler.
const BOM = String.fromCharCode(0xfeff)

/** Trigger a client-side CSV download from headers + rows. */
export function downloadCsv(filename: string, headers: string[], rows: (string | number)[][]) {
  const escape = (v: string | number) => `"${String(v).replace(/"/g, '""')}"`
  const lines = [headers, ...rows].map((r) => r.map(escape).join(','))
  // Prepend a BOM so Excel reads UTF-8 (Rp, accented names) correctly.
  const blob = new Blob([BOM + lines.join('\n')], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
