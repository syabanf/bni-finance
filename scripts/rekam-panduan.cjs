#!/usr/bin/env node
/**
 * Merekam video halaman Panduan ke public/video-panduan (.webm dan poster .jpg).
 *
 * Playwright sengaja bukan dependensi proyek; pasang di luar repo:
 *
 *   npm i --prefix /tmp/pw playwright && npx --prefix /tmp/pw playwright install chromium
 *   NODE_PATH=/tmp/pw/node_modules node scripts/rekam-panduan.cjs [id-alur...]
 *
 * Yang harus sudah berjalan:
 *   - `npm run dev` di http://localhost:5173
 *   - untuk alur impor-member: backend di port 8123, memakai basis data SEMENTARA
 *     dan kunci kosong, karena alur itu menulis chapter dan member:
 *     DATABASE_URL=…/bni_finance_test RESEND_API_KEY= MAIL_FROM= PAPER_ID_CLIENT_ID=
 *     PAPER_ID_CLIENT_SECRET= PAPER_ID_COMPANY_ID= AUTH_QUICK_LOGIN=<email admin> go run ./cmd/api
 *     `unset` tidak cukup: backend memuat ulang variabel yang hilang dari .env.
 *
 * Video ini publik. Alur lain berjalan di Data Contoh, dan rutenya mengganti email
 * serta nomor telepon di seed dengan nilai fiktif sebelum sampai ke halaman.
 */
const { chromium } = require('playwright')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')

const BASE = process.env.PANDUAN_URL ?? 'http://localhost:5173'
const KELUAR = path.join(__dirname, '..', 'public', 'video-panduan')
const UKURAN = { width: 1280, height: 720 }

/** Dipasang di setiap muatan halaman: kursor, keterangan, dan kartu pembuka. */
const TATA_PANGGUNG = () => {
  const pasang = () => {
    if (document.getElementById('panggung')) return
    const el = document.createElement('div')
    el.id = 'panggung'
    el.innerHTML = `
      <div id="kartu" style="position:fixed;inset:0;z-index:99999;pointer-events:none;background:#fff;display:none;align-items:center;justify-content:center;text-align:center;font:700 34px/1.3 system-ui,sans-serif;color:#101112;padding:60px"></div>
      <div id="teks" style="position:fixed;left:50%;top:12px;transform:translateX(-50%);z-index:99998;pointer-events:none;max-width:980px;background:rgba(16,17,18,.9);color:#fff;font:500 19px/1.4 system-ui,sans-serif;padding:12px 22px;border-radius:12px;text-align:center;display:none"></div>
      <div id="kursor" style="position:fixed;left:0;top:0;z-index:100000;width:22px;height:22px;margin:-11px 0 0 -11px;border-radius:50%;background:rgba(237,28,36,.45);border:2px solid #ed1c24;pointer-events:none;transition:transform .12s"></div>`
    document.documentElement.appendChild(el)
    const k = el.querySelector('#kursor')
    addEventListener('mousemove', (e) => { k.style.left = e.clientX + 'px'; k.style.top = e.clientY + 'px' }, true)
    addEventListener('mousedown', () => { k.style.transform = 'scale(.6)' }, true)
    addEventListener('mouseup', () => { k.style.transform = '' }, true)
  }
  if (document.documentElement) pasang()
  addEventListener('DOMContentLoaded', pasang)
}

class Rekaman {
  constructor(page) { this.page = page }
  jeda(ms = 900) { return this.page.waitForTimeout(ms) }
  async kartu(teks, ms = 0) {
    await this.page.evaluate((t) => {
      const k = document.getElementById('kartu')
      k.textContent = t ?? ''
      k.style.display = t ? 'flex' : 'none'
    }, teks)
    if (ms) await this.jeda(ms)
  }
  /** Keterangan tampil lebih lama untuk kalimat panjang, supaya sempat terbaca. */
  async teks(t) {
    await this.page.evaluate((x) => {
      const e = document.getElementById('teks')
      e.textContent = x
      e.style.display = 'block'
    }, t)
    await this.jeda(Math.max(1600, t.length * 42))
  }
  async tunjuk(loc) {
    await loc.waitFor({ state: 'visible' })
    await loc.scrollIntoViewIfNeeded()
    const b = await loc.boundingBox()
    await this.page.mouse.move(b.x + b.width / 2, b.y + b.height / 2, { steps: 22 })
    await this.jeda(350)
  }
  async klik(loc) {
    await this.tunjuk(loc)
    await loc.click()
    await this.jeda(700)
  }
  async ketik(loc, isi) {
    await this.klik(loc)
    await loc.fill('')
    await loc.pressSequentially(isi, { delay: 45 })
    await this.jeda(300)
  }
  tombol(nama) { return this.page.getByRole('button', { name: nama }) }
  menu(nama) { return this.page.getByRole('link', { name: nama, exact: true }).first() }
}

async function masuk(r, tombolAkun) {
  await r.page.waitForLoadState('networkidle')
  await r.page.getByRole('button', { name: tombolAkun }).first().click()
  await r.page.waitForURL('**/dashboard')
  await r.page.waitForLoadState('networkidle')
}

/** Laporan fiktif berformat SpreadsheetML, sama seperti ekspor BNI Connect. */
function laporanFiktif() {
  const orang = [
    ['Adi Pratama', 'Konstruksi', 'PT Contoh Karya', '081200000011'],
    ['Bela Kurnia', 'Akuntansi', 'KAP Contoh', '081200000012'],
    ['Candra Wijaya', 'Restoran', 'Warung Contoh', '081200000013'],
    ['Dina Maharani', 'Percetakan', 'CV Cetak Contoh', '081200000014'],
    ['Eko Saputra', 'Konsultan IT', 'PT Contoh Teknologi', '081200000015'],
  ]
  const sel = (v) => `<Cell><Data ss:Type="String">${v}</Data></Cell>`
  const baris = (...v) => `<Row>${v.map(sel).join('')}</Row>`
  const buku = (judul, kolom, isi) =>
    `<?xml version="1.0"?><Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet" xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"><Worksheet ss:Name="Report"><Table>` +
    baris(judul) + baris('Running User', 'Country', 'Region', 'Chapter') +
    baris('Contoh', 'Indonesia', 'Jakarta Barat', 'Nusa Jaya') + baris('Chapter:', 'Nusa Jaya') +
    baris(...kolom) + isi.join('') + `</Table></Worksheet></Workbook>`
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'panduan-'))
  const roster = path.join(dir, 'Chapter_Roster_Report.xls')
  const dues = path.join(dir, 'Chapter_Membership_Dues_Report.xls')
  fs.writeFileSync(roster, buku('Chapter Roster Report', ['Member Name', 'Classification', 'Company Name', 'Phone'],
    orang.map((o) => baris(o[0], o[1], o[2], o[3]))))
  fs.writeFileSync(dues, buku('Membership Dues Report', ['Member Name', 'Industry', 'Type', 'Membership Status', 'Due Date'],
    orang.map((o, i) => baris(o[0], o[1], 'Member', 'Active', `2027-0${i + 1}-01T00:00:00`))))
  return { roster, dues }
}

const ALUR = {
  'impor-member': {
    judul: 'Impor member dari BNI Connect',
    api: true,
    async jalan(r) {
      const { roster, dues } = laporanFiktif()
      const unggah = async (f) => {
        await r.tunjuk(r.tombol(/Pilih berkas/))
        await r.page.locator('input[type=file]').setInputFiles(f)
        await r.jeda()
      }
      await r.klik(r.page.getByRole('button', { name: 'Pengaturan' }))
      await r.klik(r.menu('Impor Data'))
      await r.teks('Biarkan Chapter tujuan pada pilihan "Dari laporan BNI Connect".')
      await r.teks('Langkah 1: unggah Chapter Roster Report, lalu tekan Tinjau.')
      await unggah(roster)
      await r.klik(r.tombol('Tinjau'))
      await r.teks('Pratinjau menyebut chapter dari laporan. Belum ada yang ditulis.')
      await r.klik(r.tombol(/^Terapkan \d+ perubahan/))
      await r.teks('Chapter dan member tersimpan.')
      await r.klik(r.tombol('Lanjut ke langkah 2'))
      await r.teks('Langkah 2: unggah Membership Dues Report, atau template jatuh tempo.')
      await unggah(dues)
      await r.klik(r.tombol('Tinjau'))
      await r.teks('Kolom Keterangan menyebut data yang berubah, seperti tanggal jatuh tempo.')
      await r.klik(r.tombol(/^Terapkan \d+ perubahan/))
      await r.klik(r.tombol('Selesai'))
      await r.teks('Impor selesai. Member chapter sudah bisa dilihat.')
    },
  },
  'member-baru': {
    judul: 'Tambah member dan tagih pendaftaran',
    async jalan(r) {
      const p = r.page
      await r.klik(r.menu('Member'))
      await r.teks('Tekan Tambah Member untuk member baru.')
      await r.klik(r.tombol('Tambah Member'))
      const d = p.getByRole('dialog')
      await r.ketik(d.getByPlaceholder('Nama lengkap'), 'Rina Wulandari')
      await d.locator('select').selectOption({ label: 'BNI Garuda' })
      await r.teks('Isi nama dan chapter. Member masuk dengan status New Member.')
      await r.klik(d.getByRole('button', { name: 'Tambah' }))
      await r.ketik(p.getByPlaceholder('Cari nama, ID, atau email…'), 'Rina')
      await r.klik(p.getByRole('cell', { name: /Rina Wulandari/ }).first())
      await r.teks('Dari detail member, tekan Buat Invoice.')
      await r.klik(r.tombol('Buat Invoice'))
      await r.teks('Tipe Pendaftaran dan nama member sudah terpilih.')
      await r.teks('Periksa nominal dan masa berlaku di ringkasan.')
      await r.klik(r.tombol('Simpan sebagai Draft'))
      await r.teks('Invoice tersimpan sebagai draft, siap dikirim.')
    },
  },
  'ubah-member': {
    judul: 'Ubah kontak atau status member',
    async jalan(r) {
      const p = r.page
      await r.klik(r.menu('Member'))
      await r.teks('Pilih nama member untuk membuka detailnya.')
      await r.klik(p.locator('tbody tr').first())
      await r.teks('Tekan Ubah Data.')
      await r.klik(r.tombol('Ubah Data'))
      const d = p.getByRole('dialog')
      await r.ketik(d.locator('input').nth(1), '081200000099')
      await r.teks('Perbaiki email atau no. HP di sini.')
      await r.tunjuk(d.locator('select'))
      await d.locator('select').selectOption('inactive')
      await r.teks('Untuk member drop, ubah Status keanggotaan ke Inactive.')
      await r.klik(d.getByRole('button', { name: 'Simpan' }))
      await r.teks('Tersimpan. Invoice yang sudah terbit tidak ikut berubah.')
    },
  },
  'konfirmasi-renewal': {
    judul: 'Konfirmasi renewal sampai invoice',
    async jalan(r) {
      const p = r.page
      await r.klik(r.menu('Konfirmasi Renewal'))
      await r.teks('ST memilih periode, lalu menekan Minta Konfirmasi.')
      await r.klik(r.tombol('Minta Konfirmasi'))
      const d = p.getByRole('dialog')
      await r.teks('Sistem mencentang member yang jatuh tempo di periode ini.')
      await r.klik(d.getByRole('button', { name: 'Kirim permintaan' }))
      await r.teks('MC menerima email, lalu menjawab di halaman ini.')
      await r.klik(p.getByRole('button', { name: 'Diterima' }).first())
      await r.klik(p.getByRole('button', { name: 'Diterima' }).first())
      await r.teks('Member yang dijawab Diterima siap ditagih sekaligus.')
      await r.klik(r.tombol(/^Buat Invoice \(\d+\)/))
      await r.teks('Semua member itu sudah terpilih di formulir invoice.')
      await r.klik(r.tombol(/^Simpan( \d+)? (Draft|sebagai Draft)/))
      await p.getByText(/invoice draft dibuat/i).first().waitFor()
      await r.teks('Invoice terbuat sebagai draft, siap dikirim bersamaan.')
    },
  },
  'kirim-bayar': {
    judul: 'Kirim invoice dan catat pembayaran',
    async jalan(r) {
      const p = r.page
      await r.klik(r.menu('Semua Invoice'))
      await r.klik(p.getByRole('button', { name: /^Draft/ }).first())
      await r.teks('Centang draft yang akan dikirim.')
      const centang = p.locator('tbody input[type=checkbox]')
      await r.klik(centang.first())
      await r.teks('Tekan Kirim ke Paper.id. Tautan pembayaran dan jatuh tempo terisi otomatis.')
      await r.klik(r.tombol(/^Kirim \d+ ke Paper\.id/))
      await r.jeda(1500)
      await r.teks('Pembayaran lewat Paper.id tercatat otomatis.')
      await r.klik(p.getByRole('button', { name: /^Belum Dibayar/ }).first())
      await r.klik(p.getByRole('button', { name: 'Lihat detail' }).first())
      await r.teks('Untuk transfer bank atau tunai, tekan Catat Pembayaran Manual.')
      await r.klik(r.tombol('Catat Pembayaran Manual'))
      const d = p.getByRole('dialog')
      await r.teks('Isi nominal, tanggal, dan metode pembayaran.')
      await r.klik(d.getByRole('button', { name: 'Catat Pembayaran' }))
      await p.getByText('Memuat invoice').waitFor({ state: 'hidden' })
      await r.jeda()
      await r.teks('Invoice lunas, dan due date member otomatis maju.')
    },
  },
}

async function rekam(browser, id) {
  const alur = ALUR[id]
  const context = await browser.newContext({
    viewport: UKURAN,
    locale: 'id-ID',
    recordVideo: { dir: os.tmpdir(), size: UKURAN },
  })
  await context.addInitScript(TATA_PANGGUNG)
  await context.addInitScript((sumber) => localStorage.setItem('bni.dataSource', sumber), alur.api ? 'api' : 'mock')
  // Seed Data Contoh memuat kontak nyata pengembang; video ini publik.
  await context.route('**/src/services/mock/seed.ts*', async (route) => {
    const res = await route.fetch()
    const isi = (await res.text())
      .replace(/[\w.+-]+@(gmail\.com|wit\.id)/g, 'member@contoh.id')
      .replace(/\b(\+?62|0)8\d{8,11}\b/g, '081200000000')
    await route.fulfill({ response: res, body: isi })
  })
  const page = await context.newPage()
  const r = new Rekaman(page)
  await page.goto(`${BASE}/login`)
  await r.kartu(alur.judul)
  await masuk(r, /^Admin/)
  await r.kartu(alur.judul, 1800)
  await r.kartu(null)
  await r.jeda(500)
  await alur.jalan(r)
  await r.jeda(1200)
  await page.evaluate((t) => {
    document.getElementById('teks').style.display = 'none'
    const k = document.getElementById('kartu')
    k.textContent = t
    k.style.display = 'flex'
  }, alur.judul)
  await page.screenshot({ path: path.join(KELUAR, `${id}.jpg`), type: 'jpeg', quality: 80 })
  await page.close()
  const video = page.video()
  await context.close()
  await video.saveAs(path.join(KELUAR, `${id}.webm`))
  await video.delete()
}

async function main() {
  const ids = process.argv.slice(2)
  fs.mkdirSync(KELUAR, { recursive: true })
  const browser = await chromium.launch()
  let gagal = 0
  for (const id of ids.length ? ids : Object.keys(ALUR)) {
    try {
      await rekam(browser, id)
      const mb = fs.statSync(path.join(KELUAR, `${id}.webm`)).size / 1048576
      console.log(`${id}: ${mb.toFixed(1)} MB`)
    } catch (err) {
      gagal++
      console.error(`${id} GAGAL: ${err.message.split('\n').slice(0, 4).join(' | ')}`)
    }
  }
  await browser.close()
  process.exit(gagal ? 1 : 0)
}

main()
