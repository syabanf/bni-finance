import type {
  Chapter,
  InvoiceType,
  InvoiceWithRelations,
  PaymentWithInvoice,
  ReportChapterRow,
  ReportDetailRow,
  ReportFilters,
  ReportSummary,
} from '@/types'

const sum = (ns: number[]) => ns.reduce((a, n) => a + n, 0)
const persen = (diterima: number, ditagih: number) =>
  ditagih > 0 ? Math.round((diterima / ditagih) * 100) : 0

/**
 * Menghitung laporan dari invoice dan pembayaran yang sudah ada di tangan.
 *
 * Inilah hitungan yang dulu berjalan di halaman Laporan. Sekarang server yang
 * menghitungnya dari view basis data; fungsi ini tinggal dipakai mode Data
 * Contoh, dan menjadi pembanding saat angka server diperiksa. Aturannya harus
 * tetap sama dengan v_laporan_invoice dan v_laporan_pembayaran:
 *
 *   ditagih   invoice yang TERBIT pada periode, kecuali draft dan batal.
 *             Draft belum pernah dikirim ke member, jadi belum ditagih.
 *   diterima  pembayaran yang MASUK pada periode. Saringan tipe berlaku lewat
 *             tipe invoice yang dibayar.
 */
export function hitungLaporan(
  invoices: InvoiceWithRelations[],
  payments: PaymentWithInvoice[],
  chapters: Chapter[],
  f: ReportFilters = {},
): { summary: ReportSummary; detail: ReportDetailRow[] } {
  const dalamPeriode = (iso?: string | null) => {
    if (!iso) return false
    const hari = iso.slice(0, 10)
    if (f.from && hari < f.from) return false
    if (f.to && hari > f.to) return false
    return true
  }
  const cocokTipe = (t: InvoiceType | undefined) => !f.type || f.type === 'all' || t === f.type

  const invs = invoices.filter(
    (i) => i.status !== 'cancelled' && i.status !== 'draft' && dalamPeriode(i.createdAt) && cocokTipe(i.type),
  )
  const pays = payments.filter((p) => dalamPeriode(p.paidAt) && cocokTipe(p.invoice?.type))

  const ditagih = sum(invs.map((i) => i.amount))
  const diterima = sum(pays.map((p) => p.amount))

  // SETIAP chapter disiapkan lebih dulu, termasuk yang tanpa invoice. Chapter
  // dengan nol tagihan adalah chapter yang belum disentuh, dan menghilangkannya
  // dari tabel membuatnya terbaca tidak ada.
  const baris = new Map<string, ReportChapterRow>()
  const namaChapter = (id: string) => chapters.find((c) => c.id === id)?.displayName ?? '—'
  const pastikan = (id: string, name: string): ReportChapterRow => {
    let row = baris.get(id)
    if (!row) {
      row = { id, name, count: 0, ditagih: 0, renewal: 0, pendaftaran: 0, diterima: 0, outstanding: 0, rate: 0 }
      baris.set(id, row)
    }
    return row
  }
  for (const c of chapters) pastikan(c.id, c.displayName)

  for (const i of invs) {
    const row = pastikan(i.chapterId, i.chapter?.displayName ?? namaChapter(i.chapterId))
    row.count += 1
    row.ditagih += i.amount
    if (i.type === 'renewal') row.renewal += i.amount
    else row.pendaftaran += i.amount
    if (i.status === 'sent' || i.status === 'overdue') row.outstanding += i.amount
  }
  for (const p of pays) {
    const chId = p.invoice?.chapterId
    if (chId) pastikan(chId, namaChapter(chId)).diterima += p.amount
  }

  const bulan = new Map<string, { issued: number; paid: number }>()
  const tambah = (key: string, field: 'issued' | 'paid', amt: number) => {
    const m = bulan.get(key) ?? { issued: 0, paid: 0 }
    m[field] += amt
    bulan.set(key, m)
  }
  for (const i of invs) tambah(i.createdAt.slice(0, 7), 'issued', i.amount)
  for (const p of pays) tambah(p.paidAt.slice(0, 7), 'paid', p.amount)

  const metode = new Map<string, { count: number; amount: number }>()
  for (const p of pays) {
    const key = p.paymentMethod || 'other'
    const m = metode.get(key) ?? { count: 0, amount: 0 }
    m.count += 1
    m.amount += p.amount
    metode.set(key, m)
  }

  // Dasar yang sama dengan tabel chapter: invoice yang terbit pada periode, dan
  // penerimaan yang masuk pada periode.
  const diterimaPerInvoice = new Map<string, number>()
  for (const p of pays) diterimaPerInvoice.set(p.invoiceId, (diterimaPerInvoice.get(p.invoiceId) ?? 0) + p.amount)
  const detail: ReportDetailRow[] = invs
    .map((i) => ({
      id: i.id,
      number: i.number,
      member: i.member?.name ?? null,
      chapter: i.chapter?.displayName ?? null,
      type: i.type,
      terbit: i.createdAt,
      jatuhTempo: i.dueDate,
      status: i.status,
      ditagih: i.amount,
      diterima: diterimaPerInvoice.get(i.id) ?? 0,
    }))
    .sort((a, b) => b.terbit.localeCompare(a.terbit) || a.number.localeCompare(b.number))

  return {
    summary: {
      count: invs.length,
      ditagih,
      diterima,
      outstanding: sum(invs.filter((i) => i.status === 'sent' || i.status === 'overdue').map((i) => i.amount)),
      rate: persen(diterima, ditagih),
      chapters: [...baris.values()].map((r) => ({ ...r, rate: persen(r.diterima, r.ditagih) })),
      types: {
        registration: invs.filter((i) => i.type === 'registration').length,
        renewal: invs.filter((i) => i.type === 'renewal').length,
      },
      monthly: [...bulan.entries()]
        .sort((a, b) => a[0].localeCompare(b[0]))
        .slice(-12)
        .map(([month, v]) => ({ month, ...v })),
      methods: [...metode.entries()].map(([method, v]) => ({ method, ...v })).sort((a, b) => b.amount - a.amount),
      detailDiterima: sum(detail.map((d) => d.diterima)),
    },
    detail,
  }
}
