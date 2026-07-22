import { downloadCsv } from './csv'
import { downloadXlsx } from './xlsx'
import {
  printTableReport,
  type ReportColumn,
  type ReportSection,
  type SummaryStat,
} from './pdfReport'
import { formatDateTime } from './format'

export interface PageExport {
  /** Base filename, no extension. */
  filename: string
  /** Document title (also the Excel sheet name). */
  title: string
  subtitle?: string
  /** PDF header meta lines. Defaults to row count + generated-at. */
  meta?: string[]
  columns: ReportColumn[]
  /** Rows for CSV/Excel — keep numbers as numbers so Excel can sum them. */
  rows: (string | number)[][]
  /** Display rows for the PDF (formatted currency/dates). Defaults to `rows`. */
  pdfRows?: (string | number)[][]
  totals?: (string | number)[]
  summary?: SummaryStat[]
  extraSections?: ReportSection[]
  /** Called when the PDF print window is blocked by a popup blocker. */
  onPopupBlocked?: () => void
}

/**
 * Build the Excel/CSV/PDF handlers for an <ExportMenu>. Callers pass the
 * CURRENTLY FILTERED rows, so every export reflects the active filters.
 */
export function makeExportHandlers(cfg: PageExport) {
  const headers = cfg.columns.map((c) => c.label)
  return {
    onExcel: () => downloadXlsx(cfg.filename, cfg.title, headers, cfg.rows),
    onCsv: () => downloadCsv(`${cfg.filename}.csv`, headers, cfg.rows),
    onPdf: () => {
      const ok = printTableReport({
        title: cfg.title,
        subtitle: cfg.subtitle,
        meta: cfg.meta ?? [`${cfg.rows.length} baris`, `Dibuat ${formatDateTime(new Date())}`],
        columns: cfg.columns,
        rows: cfg.pdfRows ?? cfg.rows,
        totals: cfg.totals,
        summary: cfg.summary,
        extraSections: cfg.extraSections,
        documentTitle: `${cfg.title} — BNI Finance`,
      })
      if (!ok) cfg.onPopupBlocked?.()
    },
  }
}
