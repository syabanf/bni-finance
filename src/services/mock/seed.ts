/**
 * Deterministic seed data for the mock backend.
 *
 * Dates are anchored around "today" = 2026-06-15 (the date in the project
 * plan) so renewal-due / overdue logic produces a realistic spread. The data
 * is generated once and held in memory by the store; the UI can mutate it
 * during a session (create / send / pay / cancel invoices).
 */

import type {
  AuditLogEntry,
  Chapter,
  FeeSettings,
  Invoice,
  InvoiceStatus,
  InvoiceType,
  Member,
  Payment,
} from '@/types'
import { addDays, addYear } from '@/lib/date'

const NOW = '2026-06-15'

/** Tahun penomoran invoice, mengikuti nomor literal di db/init.sql. */
const SEED_YEAR = '2026'
const SYNCED_AT = '2026-06-15T07:30:00Z'

// ---------------------------------------------------------------------------
// Chapters
// ---------------------------------------------------------------------------

export const seedChapters: Chapter[] = [
  { id: 'ch-garuda', name: 'Garuda', displayName: 'BNI Garuda', areaName: 'Jakarta Pusat', cityName: 'Jakarta', syncedAt: SYNCED_AT },
  { id: 'ch-nusantara', name: 'Nusantara', displayName: 'BNI Nusantara', areaName: 'Jakarta Selatan', cityName: 'Jakarta', syncedAt: SYNCED_AT },
  { id: 'ch-merdeka', name: 'Merdeka', displayName: 'BNI Merdeka', areaName: 'Bandung Kota', cityName: 'Bandung', syncedAt: SYNCED_AT },
  { id: 'ch-samudra', name: 'Samudra', displayName: 'BNI Samudra', areaName: 'Surabaya Timur', cityName: 'Surabaya', syncedAt: SYNCED_AT },
  // Seed 2026-08-24 — dua kota yang sebelumnya belum terwakili.
  { id: 'ch-cakrawala', name: 'Cakrawala', displayName: 'BNI Cakrawala', areaName: 'Denpasar Selatan', cityName: 'Denpasar', syncedAt: SYNCED_AT },
  { id: 'ch-mahakam', name: 'Mahakam', displayName: 'BNI Mahakam', areaName: 'Samarinda Kota', cityName: 'Samarinda', syncedAt: SYNCED_AT },
]

// ---------------------------------------------------------------------------
// Members — name, chapter, joined date drive the generated invoices below.
// ---------------------------------------------------------------------------

interface MemberSeed {
  name: string
  chapterId: string
  joined: string
  status?: Member['status']
  /**
   * Invoice yang dimiliki member ini, berurutan.
   *
   * Tipe ditulis eksplisit, tidak lagi disimpulkan dari posisi. Aturan lama
   * "entri pertama pasti registration" tidak bisa mencerminkan data nyata, di
   * mana sebagian besar member hanya punya satu invoice renewal — dan aturan
   * yang menebak akan selalu meleset di situ.
   */
  history: { status: InvoiceStatus; type: InvoiceType }[]
}

const memberSeeds: MemberSeed[] = [
  // Cerminan persis db/init.sql — nama, chapter, urutan, tipe, dan status
  // invoice-nya. Beralih antara Data Contoh dan Backend API tidak boleh
  // mengubah dunia yang dilihat pengguna; kalau berbeda, demo menjanjikan
  // sesuatu yang tidak akan mereka temukan di sistem sebenarnya.
  //
  // Enam invoice, mencakup kelima status, sama seperti data nyata.
  { name: 'Budi Santoso', chapterId: 'ch-garuda', joined: '2025-09-15',
    history: [{ status: 'sent', type: 'renewal' }] },
  { name: 'Siti Rahayu', chapterId: 'ch-garuda', joined: '2025-10-19',
    history: [{ status: 'overdue', type: 'renewal' }] },
  { name: 'Andi Wijaya', chapterId: 'ch-nusantara', joined: '2025-08-31',
    history: [{ status: 'paid', type: 'renewal' }] },
  { name: 'Dewi Lestari', chapterId: 'ch-nusantara', joined: '2026-04-17',
    history: [{ status: 'draft', type: 'registration' }] },
  { name: 'Rudi Hartono', chapterId: 'ch-merdeka', joined: '2026-02-02',
    history: [{ status: 'paid', type: 'renewal' }] },
  { name: 'Maya Puspita', chapterId: 'ch-merdeka', joined: '2026-06-01',
    status: 'pending', history: [] },
  { name: 'Hendra Gunawan', chapterId: 'ch-samudra', joined: '2026-06-21',
    history: [{ status: 'cancelled', type: 'registration' }] },
  { name: 'Rina Kartika', chapterId: 'ch-samudra', joined: '2025-05-12',
    status: 'inactive', history: [] },

  // --- Seed 2026-08-24: member yang BELUM punya invoice ---------------------
  // Cerminan db/seeds/2026-08-24-member-tanpa-invoice.sql. Semuanya
  // history: [] — itu seluruh maksudnya. Halaman "Buat Invoice" hanya
  // menampilkan member yang belum punya invoice pendaftaran aktif; tanpa
  // tambahan ini, daftarnya habis begitu member lama ditagih dan alur
  // pengiriman tidak bisa dicoba lagi.
  { name: 'Gita Anindya', chapterId: 'ch-cakrawala', joined: '2026-08-24',
    status: 'pending', history: [] },
  { name: 'Bayu Prakoso', chapterId: 'ch-cakrawala', joined: '2026-08-24',
    status: 'pending', history: [] },
  { name: 'Nadia Rahmawati', chapterId: 'ch-mahakam', joined: '2026-08-24',
    status: 'pending', history: [] },
  { name: 'Fajar Nugroho', chapterId: 'ch-mahakam', joined: '2026-08-24',
    status: 'pending', history: [] },
  { name: 'Laras Wulandari', chapterId: 'ch-garuda', joined: '2025-08-31',
    history: [] },
  { name: 'Reza Firmansyah', chapterId: 'ch-nusantara', joined: '2025-09-07',
    history: [] },
  { name: 'Ayu Kusuma', chapterId: 'ch-merdeka', joined: '2025-09-14',
    history: [] },
  { name: 'Yoga Pratama', chapterId: 'ch-samudra', joined: '2026-07-25',
    history: [] },
]

// ---------------------------------------------------------------------------
// Fee settings
// ---------------------------------------------------------------------------

export const seedFeeSettings: FeeSettings = {
  id: 'fee-default',
  registrationFee: 1_500_000,
  renewalFee: 1_500_000,
  currency: 'IDR',
  notes: 'Biaya pendaftaran berlaku untuk visitor yang resmi bergabung. Renewal dibayar tahunan.',
  updatedBy: 'admin-national',
  updatedAt: '2026-01-05T03:00:00Z',
  createdAt: '2026-01-05T03:00:00Z',
}

// ---------------------------------------------------------------------------
// Generation: members → invoices → payments → audit log
// ---------------------------------------------------------------------------

/**
 * Dua alamat milik tim, dipakai berselang-seling untuk seluruh member seed.
 *
 * Alasannya sama dengan [SEED_PHONE]: menerbitkan invoice dengan kanal email
 * menyala membuat Paper.id benar-benar mengirim ke alamat yang tertulis di
 * member. Alamat karangan bisa saja milik orang lain; dengan alamat milik tim,
 * uji coba hanya sampai ke kita.
 *
 * Dua dan bukan satu supaya kedua kotak masuk benar-benar teruji — pengantaran
 * bisa berhasil ke satu domain dan tersaring di domain lain, dan itu hanya
 * ketahuan bila keduanya dipakai.
 *
 * Ganti bila kotak masuk ujinya berganti — jangan dikembalikan menjadi
 * per-nama.
 */
const SEED_EMAILS = ['muhfahmifm@gmail.com', 'fahmi@wit.id'] as const

/**
 * Satu nomor untuk seluruh member seed, dan itu disengaja.
 *
 * Menerbitkan invoice dengan kanal WhatsApp menyala membuat Paper.id benar-benar
 * mengirim pesan ke nomor yang tertulis di member. Nomor karangan yang berbeda-
 * beda berarti pesan uji coba mendarat di ponsel orang lain yang kebetulan
 * memiliki nomor itu. Dengan satu nomor milik tim, uji coba hanya sampai ke
 * kita sendiri.
 *
 * Ganti bila ponsel ujinya berganti — jangan dikembalikan menjadi acak.
 * Alamat email mengikuti alasan yang sama; lihat SEED_EMAILS.
 */
const SEED_PHONE = '082240274833'

interface BuiltData {
  members: Member[]
  invoices: Invoice[]
  payments: Payment[]
  auditLog: AuditLogEntry[]
}

export function buildSeedData(): BuiltData {
  const members: Member[] = []
  const invoices: Invoice[] = []
  const payments: Payment[] = []
  const auditLog: AuditLogEntry[] = []

  let invoiceSeq = 0
  let paymentSeq = 0
  let auditSeq = 0

  memberSeeds.forEach((seed, idx) => {
    const memberId = `m${String(idx + 1).padStart(3, '0')}`
    const member: Member = {
      id: memberId,
      chapterId: seed.chapterId,
      name: seed.name,
      email: SEED_EMAILS[idx % SEED_EMAILS.length],
      phone: SEED_PHONE,
      status: seed.status ?? (seed.history.some((h) => h.status === 'overdue') ? 'pending' : 'active'),
      joinedDate: seed.joined,
      renewalDate: null,
      syncedAt: SYNCED_AT,
    }
    members.push(member)

    // Walk the member's invoice history. First entry = registration, the rest = renewals.
    let periodStart = seed.joined
    seed.history.forEach(({ status, type }, hIdx) => {
      invoiceSeq += 1
      // Nominal registrasi 2 juta mengikuti invoice contoh di db/init.sql,
      // bukan seedFeeSettings.registrationFee. Keduanya memang berbeda di data
      // nyata, dan mencerminkannya apa adanya lebih jujur daripada menyamakan
      // diam-diam lalu membuat demo dan sistem sebenarnya menampilkan angka
      // yang tidak sama.
      const amount = type === 'registration' ? 2_000_000 : seedFeeSettings.renewalFee
      const periodEnd = addYear(periodStart)
      // Issued shortly before the period starts (registration on join day; renewal ~ when prior ends).
      const dueDate = hIdx === 0 ? periodStart : addDays(periodStart, -2)
      const createdAt = `${dueDate}T02:00:00Z`
      // Tahunnya tetap, bukan diturunkan dari jatuh tempo. Nomor di
      // db/init.sql ditulis literal sebagai INV-2026-001 sampai 006, dan
      // menurunkannya dari tanggal membuat tiga nomor pertama jatuh ke 2025 —
      // beda dengan data nyata tanpa alasan yang bisa dijelaskan ke pengguna.
      const number = `INV-${SEED_YEAR}-${String(invoiceSeq).padStart(3, '0')}`

      const invoiceId = `inv-${String(invoiceSeq).padStart(4, '0')}`
      const sent = status === 'sent' || status === 'paid' || status === 'overdue'
      const paid = status === 'paid'

      const invoice: Invoice = {
        id: invoiceId,
        number,
        memberId,
        chapterId: seed.chapterId,
        type,
        amount,
        currency: 'IDR',
        dueDate,
        periodStart,
        periodEnd,
        status,
        paperIdInvoiceId: sent ? `PPR-${invoiceSeq}${dueDate.slice(2, 4)}` : undefined,
        paperIdInvoiceUrl: sent ? `https://app.paper.id/invoice/PPR-${invoiceSeq}` : undefined,
        paperIdPaymentUrl: sent ? `https://pay.paper.id/PPR-${invoiceSeq}` : undefined,
        paperIdSentAt: sent ? `${addDays(dueDate, 0)}T03:15:00Z` : undefined,
        paidAt: paid ? `${addDays(dueDate, 4)}T08:20:00Z` : undefined,
        paidAmount: paid ? amount : undefined,
        notes: undefined,
        createdBy: 'admin-national',
        cancelledBy: status === 'cancelled' ? 'admin-national' : undefined,
        cancelledAt: status === 'cancelled' ? `${addDays(dueDate, 3)}T05:00:00Z` : undefined,
        cancelReason: status === 'cancelled' ? 'Member mengundurkan diri sebelum pembayaran.' : undefined,
        createdAt,
        updatedAt: paid ? `${addDays(dueDate, 4)}T08:20:00Z` : createdAt,
      }
      invoices.push(invoice)

      // Audit trail
      const pushAudit = (action: AuditLogEntry['action'], oldS: InvoiceStatus | undefined, newS: InvoiceStatus, at: string, notes?: string) => {
        auditSeq += 1
        auditLog.push({
          id: `aud-${String(auditSeq).padStart(4, '0')}`,
          invoiceId,
          action,
          oldStatus: oldS,
          newStatus: newS,
          actorId: 'admin-national',
          actorName: 'Admin Nasional',
          notes,
          createdAt: at,
        })
      }
      pushAudit('created', undefined, 'draft', createdAt, `Invoice ${type} dibuat`)
      if (sent) pushAudit('sent', 'draft', 'sent', `${dueDate}T03:15:00Z`, 'Dikirim ke Paper.id')
      if (paid) pushAudit('paid', 'sent', 'paid', invoice.paidAt!, 'Pembayaran diterima via Paper.id')
      if (status === 'overdue') pushAudit('overdue', 'sent', 'overdue', addDays(dueDate, 32) + 'T00:05:00Z', 'Jatuh tempo terlewati')
      if (status === 'cancelled') pushAudit('cancelled', 'draft', 'cancelled', invoice.cancelledAt!, invoice.cancelReason)

      // Payment record for paid invoices
      if (paid) {
        paymentSeq += 1
        payments.push({
          id: `pay-${String(paymentSeq).padStart(4, '0')}`,
          invoiceId,
          amount,
          paidAt: invoice.paidAt!,
          paymentMethod: ['virtual_account', 'bank_transfer', 'qris'][invoiceSeq % 3],
          paperIdPaymentId: `PAY-${invoiceSeq}${dueDate.slice(2, 4)}`,
          paperIdStatus: 'success',
          createdAt: invoice.paidAt!,
        })
      }

      // The membership lapses when its latest covered period ends. The backend
      // filters renewal-due on `renewal_date` alone, so leaving it null here
      // would make that endpoint permanently empty on mock data.
      if (status !== 'cancelled') member.renewalDate = periodEnd

      // Next renewal period begins the day after the current one ends.
      periodStart = addDays(periodEnd, 1)
    })
  })

  return { members, invoices, payments, auditLog }
}

export { NOW }
