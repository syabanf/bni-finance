import type { RenewalPeriod } from '@/types'

/**
 * Hitungan tanggal untuk periode konfirmasi renewal. Tanggal ditulis sebagai
 * teks "YYYY-MM-DD" dan dibandingkan sebagai teks, jadi zona waktu browser
 * tidak menggeser periode ke hari sebelumnya.
 */

const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des']

const iso = (tahun: number, bulan: number, hari: number) =>
  `${tahun}-${String(bulan).padStart(2, '0')}-${String(hari).padStart(2, '0')}`

/** Tanggal hari ini menurut jam pengguna, sebagai "YYYY-MM-DD". */
export function hariIni(d = new Date()): string {
  return iso(d.getFullYear(), d.getMonth() + 1, d.getDate())
}

/** Hari sesudah tanggal t. */
function besoknya(t: string): string {
  const [y, m, d] = t.split('-').map(Number)
  const b = new Date(Date.UTC(y, m - 1, d + 1))
  return iso(b.getUTCFullYear(), b.getUTCMonth() + 1, b.getUTCDate())
}

/** Hari terakhir bulan ke-`bulan` (1 sampai 12). */
const akhirBulan = (tahun: number, bulan: number) => new Date(Date.UTC(tahun, bulan, 0)).getUTCDate()

const tanggalPendek = (t: string) => {
  const [y, m, d] = t.split('-').map(Number)
  return { y, label: `${d} ${BULAN[m - 1]}` }
}

/**
 * Nama yang disarankan untuk sebuah rentang: "2029 · Jul–Des" untuk semester,
 * "2029" untuk setahun penuh, dan kedua tanggalnya untuk rentang lain.
 */
export function namaDariRentang(mulai: string, selesai: string): string {
  if (!mulai || !selesai) return ''
  const tahun = mulai.slice(0, 4)
  if (tahun === selesai.slice(0, 4)) {
    const awal = mulai.slice(5)
    const akhir = selesai.slice(5)
    if (awal === '01-01' && akhir === '06-30') return `${tahun} · Jan–Jun`
    if (awal === '07-01' && akhir === '12-31') return `${tahun} · Jul–Des`
    if (awal === '01-01' && akhir === '12-31') return tahun
  }
  const a = tanggalPendek(mulai)
  const b = tanggalPendek(selesai)
  return a.y === b.y ? `${a.label} – ${b.label} ${b.y}` : `${a.label} ${a.y} – ${b.label} ${b.y}`
}

/**
 * Rentang untuk periode baru: sambungan periode terakhir dengan panjang yang
 * sama dalam bulan, atau semester berjalan bila belum ada periode sama sekali.
 *
 * Panjangnya dihitung dalam bulan, bukan hari: semester Juli sampai Desember
 * lebih panjang tiga hari daripada Januari sampai Juni, dan menyambung dengan
 * hari akan menggeser semester berikutnya dari tanggal 1.
 */
export function rentangBerikutnya(periode: RenewalPeriod[], hari = hariIni()): { mulai: string; selesai: string } {
  const terakhir = [...periode].sort((a, b) => a.endDate.localeCompare(b.endDate)).at(-1)
  if (!terakhir) {
    const [y, m] = hari.split('-').map(Number)
    return m <= 6 ? { mulai: iso(y, 1, 1), selesai: iso(y, 6, 30) } : { mulai: iso(y, 7, 1), selesai: iso(y, 12, 31) }
  }
  const [ya, ma] = terakhir.startDate.split('-').map(Number)
  const [yb, mb] = terakhir.endDate.split('-').map(Number)
  const bulan = Math.max(1, (yb - ya) * 12 + (mb - ma) + 1)
  const mulai = besoknya(terakhir.endDate)
  const [y, m] = mulai.split('-').map(Number)
  const ujung = m - 1 + bulan - 1
  const ys = y + Math.floor(ujung / 12)
  const ms = (ujung % 12) + 1
  return { mulai, selesai: iso(ys, ms, akhirBulan(ys, ms)) }
}

/**
 * Periode yang dibuka pertama kali: yang memuat hari ini, lalu yang terdekat
 * sesudahnya, lalu yang paling akhir.
 */
export function periodeAwal(periode: RenewalPeriod[], hari = hariIni()): string | null {
  return (
    periode.find((p) => p.startDate <= hari && hari <= p.endDate)?.id ??
    periode.find((p) => p.startDate > hari)?.id ??
    periode.at(-1)?.id ??
    null
  )
}

/** Periode lain yang beririsan dengan rentang ini, bila ada. */
export function periodeBeririsan(
  periode: RenewalPeriod[],
  mulai: string,
  selesai: string,
  kecuali?: string,
): RenewalPeriod | undefined {
  return periode.find((p) => p.id !== kecuali && p.startDate <= selesai && p.endDate >= mulai)
}

/**
 * Id yang diberikan server untuk rentang baru. Mock memakainya supaya id di
 * demo sama dengan id di produksi.
 */
export function kodePeriode(mulai: string, selesai: string): string[] {
  const tanggal = `${mulai.replaceAll('-', '')}-${selesai.replaceAll('-', '')}`
  const tahun = mulai.slice(0, 4)
  if (tahun !== selesai.slice(0, 4)) return [tanggal]
  const nama = namaDariRentang(mulai, selesai)
  if (nama.endsWith('Jan–Jun')) return [`${tahun}-S1`, tanggal]
  if (nama.endsWith('Jul–Des')) return [`${tahun}-S2`, tanggal]
  if (nama === tahun) return [tahun, tanggal]
  return [tanggal]
}
