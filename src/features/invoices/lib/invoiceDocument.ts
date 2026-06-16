import type { InvoiceWithRelations } from '@/types'
import { formatCurrency, formatDate, formatDateLong } from '@/lib/format'

/**
 * Self-contained invoice renderer.
 *
 * Everything is inline-styled so the exact same markup renders identically:
 *  - on-screen inside the preview modal (via dangerouslySetInnerHTML), and
 *  - inside a fresh print window for "Save as PDF" (no Tailwind dependency).
 */

const BRAND = '#E11900'
const INK = '#0f172a'
const MUTED = '#64748b'
const LINE = '#e2e8f0'

const STATUS_STYLE: Record<string, { label: string; bg: string; fg: string }> = {
  draft: { label: 'DRAFT', bg: '#f1f5f9', fg: '#475569' },
  sent: { label: 'OUTSTANDING', bg: '#fef3c7', fg: '#b45309' },
  overdue: { label: 'OVERDUE', bg: '#fee2e2', fg: '#b91c1c' },
  paid: { label: 'LUNAS', bg: '#dcfce7', fg: '#15803d' },
  cancelled: { label: 'DIBATALKAN', bg: '#f1f5f9', fg: '#64748b' },
}

/** Escape DB/user-controlled values before inlining into invoice HTML (anti-XSS). */
function esc(s: unknown): string {
  return String(s ?? '').replace(/[&<>"']/g, (c) =>
    ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c] as string,
  )
}

function metaRow(label: string, value: string): string {
  return `
    <div style="display:flex;justify-content:space-between;gap:24px;padding:7px 0;border-bottom:1px solid ${LINE};">
      <span style="color:${MUTED};font-size:13px;">${esc(label)}</span>
      <span style="color:${INK};font-size:13px;font-weight:600;text-align:right;">${esc(value)}</span>
    </div>`
}

/** Inner invoice markup (no <html>/<body> wrapper). */
export function renderInvoiceBody(inv: InvoiceWithRelations): string {
  const m = inv.member
  const ch = inv.chapter
  const st = STATUS_STYLE[inv.status] ?? STATUS_STYLE.draft
  const itemTitle =
    inv.type === 'registration' ? 'Biaya Pendaftaran Member BNI' : 'Biaya Renewal Keanggotaan BNI'
  const itemSub = `Periode ${formatDate(inv.periodStart)} – ${formatDate(inv.periodEnd)}`

  return `
  <div style="max-width:760px;margin:0 auto;background:#fff;color:${INK};
              font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;
              box-shadow:0 10px 40px rgba(15,23,42,.08);border-radius:16px;overflow:hidden;">

    <!-- Header band -->
    <div style="background:linear-gradient(135deg,${BRAND} 0%,#9d1200 100%);padding:36px 48px;color:#fff;
                display:flex;justify-content:space-between;align-items:flex-start;">
      <div>
        <div style="font-size:22px;font-weight:800;letter-spacing:-.4px;">BNI Indonesia</div>
        <div style="font-size:12px;opacity:.85;margin-top:2px;letter-spacing:.4px;">PAYMENT PLATFORM</div>
      </div>
      <div style="text-align:right;">
        <div style="font-size:30px;font-weight:800;letter-spacing:1px;line-height:1;">INVOICE</div>
        <div style="font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:14px;opacity:.9;margin-top:8px;">
          ${esc(inv.number)}
        </div>
        <div style="display:inline-block;margin-top:10px;background:rgba(255,255,255,.18);
                    padding:4px 12px;border-radius:999px;font-size:11px;font-weight:700;letter-spacing:.6px;">
          ${st.label}
        </div>
      </div>
    </div>

    <!-- Body -->
    <div style="padding:40px 48px;">

      <!-- Parties -->
      <div style="display:flex;justify-content:space-between;gap:40px;margin-bottom:34px;">
        <div style="flex:1;">
          <div style="font-size:11px;font-weight:700;letter-spacing:.8px;color:${MUTED};margin-bottom:10px;">
            DITAGIHKAN KEPADA
          </div>
          <div style="font-size:17px;font-weight:700;color:${INK};">${esc(m?.name ?? '—')}</div>
          ${m?.email ? `<div style="font-size:13px;color:${MUTED};margin-top:5px;">${esc(m.email)}</div>` : ''}
          ${m?.phone ? `<div style="font-size:13px;color:${MUTED};margin-top:2px;">${esc(m.phone)}</div>` : ''}
          ${ch ? `<div style="display:inline-block;margin-top:10px;background:#f8fafc;border:1px solid ${LINE};
                  padding:4px 11px;border-radius:8px;font-size:12px;color:${INK};font-weight:600;">${esc(ch.displayName)}</div>` : ''}
        </div>
        <div style="width:280px;">
          ${metaRow('Tipe', inv.type === 'registration' ? 'Pendaftaran' : 'Renewal')}
          ${metaRow('Tanggal Terbit', formatDate(inv.createdAt))}
          ${inv.dueDate ? metaRow('Jatuh Tempo', formatDate(inv.dueDate)) : ''}
          ${metaRow('Mata Uang', inv.currency)}
        </div>
      </div>

      <!-- Items -->
      <table style="width:100%;border-collapse:collapse;margin-bottom:8px;">
        <thead>
          <tr style="background:${INK};">
            <th style="text-align:left;padding:12px 18px;font-size:11px;font-weight:700;letter-spacing:.6px;color:#fff;border-radius:10px 0 0 10px;">
              DESKRIPSI
            </th>
            <th style="text-align:right;padding:12px 18px;font-size:11px;font-weight:700;letter-spacing:.6px;color:#fff;border-radius:0 10px 10px 0;">
              JUMLAH
            </th>
          </tr>
        </thead>
        <tbody>
          <tr style="border-bottom:1px solid ${LINE};">
            <td style="padding:18px;vertical-align:top;">
              <div style="font-size:15px;font-weight:600;color:${INK};">${itemTitle}</div>
              <div style="font-size:12px;color:${MUTED};margin-top:4px;">${itemSub}</div>
              ${inv.notes ? `<div style="font-size:12px;color:#94a3b8;margin-top:4px;font-style:italic;">${esc(inv.notes)}</div>` : ''}
            </td>
            <td style="padding:18px;text-align:right;font-size:15px;font-weight:600;color:${INK};white-space:nowrap;vertical-align:top;">
              ${formatCurrency(inv.amount)}
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Totals -->
      <div style="display:flex;justify-content:flex-end;margin-top:18px;">
        <div style="width:300px;">
          <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;color:${MUTED};">
            <span>Subtotal</span><span style="color:${INK};font-weight:600;">${formatCurrency(inv.amount)}</span>
          </div>
          <div style="display:flex;justify-content:space-between;align-items:center;margin-top:8px;
                      background:#fff5f4;border:1px solid #fcd9d3;border-radius:12px;padding:14px 18px;">
            <span style="font-size:13px;font-weight:700;color:${INK};">TOTAL TAGIHAN</span>
            <span style="font-size:21px;font-weight:800;color:${BRAND};">${formatCurrency(inv.amount)}</span>
          </div>
        </div>
      </div>

      ${
        inv.dueDate && inv.status !== 'paid' && inv.status !== 'cancelled'
          ? `<div style="margin-top:30px;background:#fffbeb;border:1px solid #fde68a;border-left:4px solid #f59e0b;
                  border-radius:10px;padding:14px 18px;font-size:13px;color:#92400e;">
              Mohon lakukan pembayaran sebelum <strong>${formatDateLong(inv.dueDate)}</strong>
              untuk menghindari keterlambatan.
            </div>`
          : ''
      }
      ${
        inv.status === 'paid' && inv.paidAt
          ? `<div style="margin-top:30px;background:#f0fdf4;border:1px solid #bbf7d0;border-left:4px solid #22c55e;
                  border-radius:10px;padding:14px 18px;font-size:13px;color:#166534;">
              Invoice ini telah <strong>LUNAS</strong> pada ${formatDateLong(inv.paidAt)}. Terima kasih.
            </div>`
          : ''
      }
    </div>

    <!-- Footer -->
    <div style="border-top:1px solid ${LINE};padding:24px 48px;text-align:center;">
      <div style="font-size:12px;color:${MUTED};">BNI Indonesia — dokumen ini diterbitkan secara otomatis oleh sistem.</div>
      <div style="font-size:12px;color:#94a3b8;margin-top:3px;">Terima kasih atas kepercayaan Anda.</div>
    </div>
  </div>`
}

/** Full standalone HTML document for the print / Save-as-PDF window. */
export function buildInvoiceDocument(inv: InvoiceWithRelations): string {
  return `<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${esc(inv.number)}</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{background:#f1f5f9;padding:32px 16px;-webkit-print-color-adjust:exact;print-color-adjust:exact}
    @media print{
      body{background:#fff;padding:0}
      @page{margin:12mm}
    }
  </style>
</head>
<body>${renderInvoiceBody(inv)}</body>
</html>`
}

/** Open a fresh window with the invoice and trigger the browser print/save dialog. */
export function downloadInvoice(inv: InvoiceWithRelations): void {
  const win = window.open('', '_blank', 'width=900,height=820')
  if (!win) return
  win.document.open()
  win.document.write(buildInvoiceDocument(inv))
  win.document.close()
  win.focus()
  // give the new document a tick to lay out before printing
  const trigger = () => win.print()
  if (win.document.readyState === 'complete') setTimeout(trigger, 350)
  else win.onload = () => setTimeout(trigger, 250)
}
