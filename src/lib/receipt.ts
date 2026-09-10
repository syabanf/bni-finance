import { formatCurrency, formatDateTime } from './format'

/**
 * Nota pembayaran — dokumen satu halaman untuk dicetak atau disimpan PDF.
 *
 * Dipisahkan dari pdfReport.ts karena bentuknya memang berbeda: laporan adalah
 * tabel yang panjangnya tak tentu, nota adalah dokumen tunggal dengan nominal
 * yang menonjol. Memaksa keduanya memakai satu pembangun akan membuat salah
 * satunya penuh cabang "kalau ini nota, jangan…".
 */
export interface Nota {
  nomorNota: string
  dibayarPada: string
  member: string
  nomorInvoice: string
  tipe: string
  metode: string
  nominal: number
  catatan?: string
}

/** Meloloskan teks agar tidak bisa menyuntik markup ke dokumen nota. */
function aman(v: string): string {
  return v
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export function buildReceiptDocument(n: Nota): string {
  const baris = (label: string, nilai: string) => `
    <tr>
      <td class="label">${aman(label)}</td>
      <td class="nilai">${aman(nilai)}</td>
    </tr>`

  return `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<title>Nota Pembayaran ${aman(n.nomorInvoice)}</title>
<style>
  @page { size: A5; margin: 14mm; }
  * { box-sizing: border-box; }
  body {
    margin: 0; padding: 0;
    font: 13px/1.55 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    color: #1a1a1a;
  }
  .kop { display: flex; justify-content: space-between; align-items: flex-start;
         border-bottom: 2px solid #1a1a1a; padding-bottom: 12px; margin-bottom: 18px; }
  .merek { font-size: 20px; font-weight: 800; letter-spacing: -0.4px; }
  .sub { font-size: 11px; color: #6b6b6b; margin-top: 2px; }
  .judul { font-size: 13px; font-weight: 700; text-transform: uppercase;
           letter-spacing: 0.6px; text-align: right; }
  .nomor { font-size: 10px; color: #6b6b6b; font-family: ui-monospace, monospace;
           margin-top: 3px; word-break: break-all; max-width: 180px; }
  table { width: 100%; border-collapse: collapse; }
  td { padding: 7px 0; vertical-align: top; }
  .label { color: #6b6b6b; width: 42%; }
  .nilai { font-weight: 600; text-align: right; }
  .jumlah { margin-top: 18px; border-top: 2px solid #1a1a1a; padding-top: 12px;
            display: flex; justify-content: space-between; align-items: baseline; }
  .jumlah .teks { font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; font-size: 12px; }
  .jumlah .angka { font-size: 22px; font-weight: 800; }
  .catatan { margin-top: 16px; padding: 10px 12px; background: #f6f6f6;
             border-radius: 6px; font-size: 12px; color: #444; }
  .kaki { margin-top: 26px; padding-top: 10px; border-top: 1px solid #e2e2e2;
          font-size: 10px; color: #8a8a8a; text-align: center; line-height: 1.6; }
</style>
</head>
<body>
  <div class="kop">
    <div>
      <div class="merek">BNI Finance Hub</div>
      <div class="sub">Invoice &amp; pembayaran keanggotaan BNI</div>
    </div>
    <div>
      <div class="judul">Nota Pembayaran</div>
      <div class="nomor">${aman(n.nomorNota)}</div>
    </div>
  </div>

  <table>
    ${baris('Member', n.member)}
    ${baris('No. Invoice', n.nomorInvoice)}
    ${baris('Tipe', n.tipe)}
    ${baris('Metode pembayaran', n.metode)}
    ${baris('Waktu bayar', formatDateTime(n.dibayarPada))}
  </table>

  <div class="jumlah">
    <span class="teks">Jumlah diterima</span>
    <span class="angka">${aman(formatCurrency(n.nominal))}</span>
  </div>

  ${n.catatan ? `<div class="catatan">${aman(n.catatan)}</div>` : ''}

  <div class="kaki">
    Nota ini dibuat otomatis oleh sistem dan sah tanpa tanda tangan.<br>
    Dicetak ${aman(formatDateTime(new Date().toISOString()))}
  </div>
</body>
</html>`
}

/**
 * Membuka nota di jendela baru lalu memanggil dialog cetak.
 *
 * Mengembalikan false bila jendelanya diblokir, supaya pemanggilnya bisa
 * memberi tahu — bukan gagal tanpa suara, yang tampak seperti tombol rusak.
 */
export function printReceipt(n: Nota): boolean {
  const win = window.open('', '_blank', 'width=760,height=900')
  if (!win) return false
  win.document.open()
  win.document.write(buildReceiptDocument(n))
  win.document.close()
  win.focus()
  const cetak = () => win.print()
  if (win.document.readyState === 'complete') setTimeout(cetak, 350)
  else win.onload = () => setTimeout(cetak, 250)
  return true
}
