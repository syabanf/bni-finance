/**
 * Generic "list as PDF" generator. Builds a self-contained, inline-styled HTML
 * document and opens it in a fresh window so the user can Save-as-PDF / print —
 * same approach as the single-invoice document (no PDF dependency).
 */

const BRAND = '#E11900'
const INK = '#0f172a'
const MUTED = '#64748b'
const LINE = '#e2e8f0'

type Align = 'left' | 'right' | 'center'

function esc(s: unknown): string {
  return String(s ?? '').replace(
    /[&<>"']/g,
    (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c] as string,
  )
}

export interface ReportColumn {
  label: string
  align?: Align
}

export interface TableReportOptions {
  title: string
  subtitle?: string
  /** Extra lines shown top-right (e.g. row count, generated-at). */
  meta?: string[]
  columns: ReportColumn[]
  rows: (string | number)[][]
  /** Optional bold totals row appended to the table. */
  totals?: (string | number)[]
  /** Browser tab / document title. Defaults to `title`. */
  documentTitle?: string
}

export function buildTableReportDocument(o: TableReportOptions): string {
  const align = (i: number) => o.columns[i]?.align ?? 'left'

  const th = o.columns
    .map(
      (c) =>
        `<th style="text-align:${c.align ?? 'left'};padding:10px 12px;font-size:11px;font-weight:700;letter-spacing:.5px;color:#fff;">${esc(c.label)}</th>`,
    )
    .join('')

  const body = o.rows
    .map(
      (r) =>
        `<tr>${r
          .map(
            (cell, i) =>
              `<td style="text-align:${align(i)};padding:9px 12px;font-size:12px;color:${INK};border-bottom:1px solid ${LINE};">${esc(cell)}</td>`,
          )
          .join('')}</tr>`,
    )
    .join('')

  const totals = o.totals
    ? `<tr>${o.totals
        .map(
          (cell, i) =>
            `<td style="text-align:${align(i)};padding:11px 12px;font-size:12px;font-weight:700;color:${INK};border-top:2px solid ${INK};background:#f8fafc;">${esc(cell)}</td>`,
        )
        .join('')}</tr>`
    : ''

  const meta = (o.meta ?? [])
    .map((m) => `<div style="font-size:12px;color:${MUTED};">${esc(m)}</div>`)
    .join('')

  return `<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${esc(o.documentTitle ?? o.title)}</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{background:#fff;color:${INK};padding:28px;-webkit-print-color-adjust:exact;print-color-adjust:exact;
         font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif}
    table{width:100%;border-collapse:collapse}
    thead tr{background:${INK}}
    tbody tr:nth-child(even){background:#fafafa}
    @media print{ body{padding:0} @page{margin:10mm;size:A4 landscape} }
  </style>
</head>
<body>
  <div style="display:flex;justify-content:space-between;align-items:flex-start;gap:24px;border-bottom:3px solid ${BRAND};padding-bottom:14px;margin-bottom:16px;">
    <div>
      <div style="display:inline-block;background:${BRAND};color:#fff;font-weight:900;font-size:20px;letter-spacing:-1px;padding:4px 10px;border-radius:6px;">BNI</div>
      <div style="font-size:20px;font-weight:800;margin-top:10px;">${esc(o.title)}</div>
      ${o.subtitle ? `<div style="font-size:13px;color:${MUTED};margin-top:2px;">${esc(o.subtitle)}</div>` : ''}
    </div>
    <div style="text-align:right;">${meta}</div>
  </div>
  <table>
    <thead><tr>${th}</tr></thead>
    <tbody>${body}${totals}</tbody>
  </table>
  <div style="margin-top:20px;text-align:center;font-size:11px;color:#94a3b8;border-top:1px solid ${LINE};padding-top:12px;">
    BNI Finance Hub — dokumen dibuat otomatis oleh sistem.
  </div>
</body>
</html>`
}

/** Open the report in a fresh window and trigger the browser print/save dialog. */
export function printTableReport(o: TableReportOptions): void {
  const win = window.open('', '_blank', 'width=1000,height=820')
  if (!win) return
  win.document.open()
  win.document.write(buildTableReportDocument(o))
  win.document.close()
  win.focus()
  const trigger = () => win.print()
  if (win.document.readyState === 'complete') setTimeout(trigger, 350)
  else win.onload = () => setTimeout(trigger, 250)
}
