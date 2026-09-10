import type { InvoiceWithRelations } from '@/types'
import { formatCurrency, formatDate } from '@/lib/format'
import { INVOICE_STATUS_LABEL } from '@/lib/status'

/**
 * Invoice, dibuat menyerupai dokumen yang diterbitkan Paper.id.
 *
 * KENAPA MENIRU, BUKAN MEMAKAI TATA LETAK SENDIRI. Member menerima invoice dari
 * Paper.id lewat email, lalu sebagian meminta salinannya ke pengurus. Dua
 * dokumen dengan tata letak berbeda untuk tagihan yang sama menimbulkan
 * pertanyaan yang tidak perlu — "ini tagihan yang mana, yang mana yang harus
 * saya bayar?" — dan pertanyaan itu paling sering muncul justru saat orang
 * sedang ragu membayar.
 *
 * Versi sebelumnya memakai kop merah BNI dengan badge status. Bagus dipandang,
 * tapi tidak menyerupai apa pun yang pernah diterima member.
 *
 * Seluruh gaya ditulis inline supaya markup yang SAMA PERSIS tampil identik di
 * dua tempat: pratinjau di layar (lewat dangerouslySetInnerHTML) dan jendela
 * cetak untuk "Simpan sebagai PDF", yang tidak memuat Tailwind.
 */

const INK = '#2c3e50'
const MUTED = '#4a5b6b'
const LINE = '#d8dee5'
const HEAD = '#34495e'

/** Escape nilai dari basis data sebelum masuk markup (anti-XSS). */
function esc(s: unknown): string {
  return String(s ?? '').replace(/[&<>"']/g, (c) =>
    ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c] as string,
  )
}

/**
 * Warna watermark mengikuti ARTI statusnya, bukan sekadar berbeda-beda.
 *
 * Hijau untuk yang selesai, merah untuk yang perlu tindakan, abu untuk yang
 * tidak berlaku lagi. Orang membaca warnanya lebih dulu daripada tulisannya.
 */
const WATERMARK: Record<string, string> = {
  paid: '16,145,96',
  overdue: '200,16,46',
  sent: '91,141,184',
  cancelled: '120,120,128',
  terminated: '120,120,128',
  draft: '150,150,158',
}

function barisMeta(label: string, nilai: string): string {
  return `
    <tr>
      <td style="text-align:right;font-weight:700;color:${INK};padding:2px 14px 2px 0;white-space:nowrap;">${esc(label)}</td>
      <td style="text-align:right;white-space:nowrap;padding:2px 0;">${esc(nilai)}</td>
    </tr>`
}

function barisTotal(label: string, nilai: string, tebal = false): string {
  const ukuran = tebal ? '14px' : '12.5px'
  return `
    <tr>
      <td style="padding:11px 0;border-bottom:1px solid #eceff2;font-weight:700;font-size:${ukuran};">${esc(label)}</td>
      <td style="padding:11px 0 11px 30px;border-bottom:1px solid #eceff2;text-align:right;font-weight:700;font-size:${ukuran};white-space:nowrap;">${esc(nilai)}</td>
    </tr>`
}

/** Markup bagian dalam invoice (tanpa pembungkus <html>/<body>). */
export function renderInvoiceBody(inv: InvoiceWithRelations): string {
  const m = inv.member
  const produk =
    inv.type === 'registration' ? 'Pendaftaran Keanggotaan BNI' : 'Perpanjangan Keanggotaan BNI'
  const deskripsi =
    inv.type === 'registration'
      ? 'Pendaftaran anggota baru'
      : 'Perpanjangan keanggotaan tahunan'

  const terbayar = inv.status === 'paid' ? inv.amount : 0
  const sisa = Math.max(0, inv.amount - terbayar)
  const rgb = WATERMARK[inv.status] ?? '150,150,158'
  const label = (INVOICE_STATUS_LABEL[inv.status] ?? 'Draft').toUpperCase()

  // Watermark ada di SETIAP status, termasuk yang masih berjalan.
  //
  // Salinan cetak beredar lebih lama daripada keadaannya: invoice lunas masih
  // tersimpan di map orang, yang dibatalkan masih terbawa ke rapat. Tanpa
  // penanda, dokumen yang sama terbaca sebagai tagihan yang masih hidup.
  //
  // print-color-adjust dipaksa karena peramban membuang latar berwarna saat
  // mencetak — watermark yang hilang justru pada cetakan adalah watermark yang
  // gagal tepat di tempat ia paling dibutuhkan.
  const watermark = `
    <div style="position:absolute;top:44%;left:50%;
                transform:translate(-50%,-50%) rotate(-24deg);
                font-size:88px;font-weight:800;letter-spacing:6px;white-space:nowrap;
                color:rgba(${rgb},0.13);z-index:0;pointer-events:none;
                -webkit-print-color-adjust:exact;print-color-adjust:exact;">${esc(label)}</div>`

  return `
  <div style="position:relative;max-width:820px;margin:0 auto;background:#fff;color:${INK};
              padding:46px 52px;border-radius:14px;
              font:12.5px/1.5 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;
              box-shadow:0 10px 40px rgba(15,23,42,.08);">
    ${watermark}
    <div style="position:relative;z-index:1;">

      <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:30px;">
        <div>
          <div style="font-size:21px;font-weight:800;letter-spacing:-.4px;">BNI Finance Hub</div>
          <div style="font-size:10.5px;color:#7f8c9a;margin-top:2px;">Invoice &amp; pembayaran keanggotaan BNI</div>
        </div>
        <div>
          <div style="font-size:30px;font-weight:400;color:#5b8db8;line-height:1;margin-bottom:12px;text-align:right;">Invoice</div>
          <table style="border-collapse:collapse;margin-left:auto;font-size:12px;">
            ${barisMeta('Referensi', inv.number)}
            ${barisMeta('Tgl. Invoice', formatDate(inv.createdAt ?? inv.dueDate))}
            ${barisMeta('Tgl. Jatuh Tempo', formatDate(inv.dueDate))}
            ${barisMeta('NPWP', '-')}
          </table>
        </div>
      </div>

      <div style="display:flex;gap:40px;margin-bottom:26px;">
        <div style="flex:1;min-width:0;">
          <h2 style="font-size:14px;font-weight:700;margin:0 0 8px;padding-bottom:7px;border-bottom:1px solid ${LINE};">Info Perusahaan</h2>
          <div style="font-size:15px;font-weight:700;margin-bottom:5px;">BNI Indonesia</div>
          <p style="margin:1px 0;color:${MUTED};">Chapter : ${esc(inv.chapter?.displayName ?? '-')}</p>
          <p style="margin:1px 0;color:${MUTED};">Email : no-reply@reddie.id</p>
        </div>
        <div style="flex:1;min-width:0;">
          <h2 style="font-size:14px;font-weight:700;margin:0 0 8px;padding-bottom:7px;border-bottom:1px solid ${LINE};">Tagihan Untuk</h2>
          <div style="font-size:15px;font-weight:700;margin-bottom:5px;">${esc(m?.name ?? '-')}</div>
          <p style="margin:1px 0;color:${MUTED};">Telp : ${esc(m?.phone || '-')}</p>
          <p style="margin:1px 0;color:${MUTED};">Email : ${esc(m?.email || '-')}</p>
        </div>
      </div>

      <table style="width:100%;border-collapse:collapse;margin-bottom:22px;">
        <thead>
          <tr>
            <th style="background:${HEAD};color:#fff;font-size:11px;font-weight:600;padding:9px 10px;text-align:left;-webkit-print-color-adjust:exact;print-color-adjust:exact;">Produk</th>
            <th style="background:${HEAD};color:#fff;font-size:11px;font-weight:600;padding:9px 10px;text-align:left;-webkit-print-color-adjust:exact;print-color-adjust:exact;">Deskripsi</th>
            <th style="background:${HEAD};color:#fff;font-size:11px;font-weight:600;padding:9px 10px;text-align:center;-webkit-print-color-adjust:exact;print-color-adjust:exact;">Kuantitas</th>
            <th style="background:${HEAD};color:#fff;font-size:11px;font-weight:600;padding:9px 10px;text-align:right;-webkit-print-color-adjust:exact;print-color-adjust:exact;">Harga</th>
            <th style="background:${HEAD};color:#fff;font-size:11px;font-weight:600;padding:9px 10px;text-align:center;-webkit-print-color-adjust:exact;print-color-adjust:exact;">Diskon</th>
            <th style="background:${HEAD};color:#fff;font-size:11px;font-weight:600;padding:9px 10px;text-align:center;-webkit-print-color-adjust:exact;print-color-adjust:exact;">Pajak</th>
            <th style="background:${HEAD};color:#fff;font-size:11px;font-weight:600;padding:9px 10px;text-align:right;-webkit-print-color-adjust:exact;print-color-adjust:exact;">Jumlah</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td style="padding:13px 10px;border-bottom:1px solid #eceff2;vertical-align:top;">${esc(produk)}</td>
            <td style="padding:13px 10px;border-bottom:1px solid #eceff2;vertical-align:top;">${esc(deskripsi)}</td>
            <td style="padding:13px 10px;border-bottom:1px solid #eceff2;text-align:center;">1</td>
            <td style="padding:13px 10px;border-bottom:1px solid #eceff2;text-align:right;white-space:nowrap;">${esc(formatCurrency(inv.amount))}</td>
            <td style="padding:13px 10px;border-bottom:1px solid #eceff2;text-align:center;">0%</td>
            <td style="padding:13px 10px;border-bottom:1px solid #eceff2;text-align:center;">-</td>
            <td style="padding:13px 10px;border-bottom:1px solid #eceff2;text-align:right;white-space:nowrap;">${esc(formatCurrency(inv.amount))}</td>
          </tr>
        </tbody>
      </table>

      <div style="display:flex;justify-content:flex-end;">
        <table style="border-collapse:collapse;min-width:300px;">
          ${barisTotal('Subtotal', formatCurrency(inv.amount))}
          ${barisTotal('Diskon Total', formatCurrency(0))}
          ${barisTotal('Pajak', '-')}
          ${barisTotal('Total', formatCurrency(inv.amount), true)}
          ${barisTotal('Total Terbayar', formatCurrency(terbayar))}
          ${barisTotal('Sisa Tagihan', formatCurrency(sisa), true)}
        </table>
      </div>

      <div style="margin-top:34px;">
        <h2 style="font-size:14px;font-weight:700;margin:0 0 8px;padding-bottom:7px;border-bottom:1px solid ${LINE};">Keterangan</h2>
        <p style="margin:0 0 20px;color:${MUTED};">
          Periode keanggotaan ${esc(formatDate(inv.periodStart))} – ${esc(formatDate(inv.periodEnd))}.
        </p>
        <h2 style="font-size:14px;font-weight:700;margin:0 0 8px;padding-bottom:7px;border-bottom:1px solid ${LINE};">Syarat dan Ketentuan</h2>
        <p style="margin:0;color:${MUTED};">
          Pembayaran dilakukan melalui tautan Paper.id yang dikirim ke email member.
          Invoice ini diterbitkan otomatis oleh sistem dan sah tanpa tanda tangan.
        </p>
      </div>
    </div>
  </div>`
}

/** Dokumen HTML utuh untuk jendela cetak / Simpan sebagai PDF. */
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
      @page{margin:14mm}
    }
  </style>
</head>
<body>${renderInvoiceBody(inv)}</body>
</html>`
}

/** Membuka invoice di jendela baru lalu memanggil dialog cetak. */
export function downloadInvoice(inv: InvoiceWithRelations): boolean {
  const win = window.open('', '_blank', 'width=900,height=1000')
  if (!win) return false
  win.document.open()
  win.document.write(buildInvoiceDocument(inv))
  win.document.close()
  win.focus()
  const trigger = () => win.print()
  if (win.document.readyState === 'complete') setTimeout(trigger, 400)
  else win.onload = () => setTimeout(trigger, 300)
  return true
}
