/**
 * Domain models for the BNI Finance System.
 *
 * These mirror the Postgres schema in `db/init.sql`. The presentation layer
 * depends only on these types and the repository interfaces — never on a
 * concrete data source — so the mock backend can be swapped for the real
 * Go/Paper.id/BNI-VM APIs without touching any UI code.
 */

// ---------------------------------------------------------------------------
// Enums (mirror the Postgres ENUM types)
// ---------------------------------------------------------------------------

export type InvoiceType = 'registration' | 'renewal'

export type InvoiceStatus = 'draft' | 'sent' | 'paid' | 'overdue' | 'cancelled' | 'terminated'

/**
 * `visitor` adalah tamu yang datang ke pertemuan chapter tapi BELUM mendaftar.
 *
 * Dibedakan supaya terlihat siapa yang perlu diikuti sampai menjadi anggota —
 * dan supaya ia tidak ikut tertagih. Penjaganya ada di backend
 * (domain.MemberStatus.BolehDitagih); di sini ia hanya perlu terlihat.
 */
export type MemberStatus = 'active' | 'inactive' | 'pending' | 'visitor' | 'new_member'

// ---------------------------------------------------------------------------
// Synced entities (read-only mirrors from BNI Visitor Management)
// ---------------------------------------------------------------------------

export interface Chapter {
  id: string
  name: string
  displayName: string
  areaName?: string
  cityName?: string
  syncedAt: string
}

export interface Member {
  id: string
  chapterId: string
  name: string
  email?: string
  phone?: string
  company?: string
  businessField?: string
  status: MemberStatus
  joinedDate: string | null
  renewalDate: string | null
  syncedAt: string
}

/** Member enriched with its chapter — convenience for list/detail views. */
export interface MemberWithChapter extends Member {
  chapter: Chapter | null
}

// ---------------------------------------------------------------------------
// Fee configuration
// ---------------------------------------------------------------------------

/** Satu komponen harga dalam Rupiah, satu baris di invoice. */
export interface BarisBiaya {
  /** `registration`, `membership`, `vat`, atau `training`. */
  kode: string
  nama: string
  jumlah: number
}

/**
 * Harga dipecah menjadi komponennya. `aktif: false` berarti harga dihitung
 * dari satu harga Dollar per tipe, dan komponen di sini belum berlaku.
 */
export interface RincianBiaya {
  aktif: boolean
  /** Registration Fee, hanya untuk pendaftaran. */
  pendaftaranUsd: number
  /** Membership Fee, untuk pendaftaran dan renewal. */
  keanggotaanUsd: number
  ppnPersen: number
  /** Online Training dalam Rupiah, hanya untuk pendaftaran, tidak kena VAT. */
  training: number
}

export interface FeeSettings {
  id: string
  /** Rupiah yang ditagihkan. Diturunkan server dari USD × kurs. */
  registrationFee: number
  renewalFee: number
  currency: string
  /** Harga dasar dalam Dollar seperti yang ditetapkan BNI, dan kurs manualnya. */
  registrationFeeUsd: number
  renewalFeeUsd: number
  usdRate: number
  /** Selama `rincian.aktif`, keempat harga di atas diturunkan darinya. */
  rincian: RincianBiaya
  notes?: string
  updatedBy?: string
  updatedAt: string
  createdAt: string
}

// ---------------------------------------------------------------------------
// Invoice
// ---------------------------------------------------------------------------

export interface Invoice {
  id: string
  /** Human-friendly number, e.g. INV-2026-001 */
  number: string

  memberId: string
  chapterId: string

  type: InvoiceType
  amount: number
  currency: string

  dueDate: string // ISO date — = tanggal invoice diterbitkan
  periodStart: string // awal masa berlaku keanggotaan
  periodEnd: string // period_start + 1 tahun

  status: InvoiceStatus

  // Paper.id integration fields
  paperIdInvoiceId?: string
  paperIdInvoiceUrl?: string
  paperIdPaymentUrl?: string
  paperIdSentAt?: string

  /** Berapa kali invoice ini dikirim ulang sebagai pengingat lewat Paper.id. */
  paperIdReminderCount?: number

  /** Faktur pajak, dilampirkan setelah invoice lunas. */
  taxInvoiceUrl?: string
  taxInvoiceAt?: string
  /** Kapan finance mengajukan renewal member ini ke BNI Global, dan oleh siapa. */
  renewalSubmittedAt?: string
  renewalSubmittedBy?: string
  /**
   * Komponen harga yang berlaku saat invoice dibuat; jumlahnya sama dengan
   * `amount`. Tidak ada bila invoice satu baris.
   */
  rincian?: BarisBiaya[]

  /**
   * Selalu 'paper_id' sejak pembayaran mandiri dihapus.
   *
   * Kolom xendit_* masih ada di database — sengaja, karena membuangnya berarti
   * migrasi destruktif atas data yang sudah tersimpan — tetapi tidak ada lagi
   * kode yang membacanya, jadi tipenya tidak lagi memuatnya. Field mati di tipe
   * hanya mengundang orang memakainya kembali.
   */
  paymentProvider?: string

  // Payment
  paidAt?: string
  paidAmount?: number

  // Metadata
  notes?: string
  createdBy?: string
  cancelledBy?: string
  cancelledAt?: string
  cancelReason?: string

  createdAt: string
  updatedAt: string
}

/** Invoice joined with member & chapter — used by tables and detail pages. */
export interface InvoiceWithRelations extends Invoice {
  member: Member | null
  chapter: Chapter | null
}

// ---------------------------------------------------------------------------
// Payment (recorded from a Paper.id webhook)
// ---------------------------------------------------------------------------

export interface Payment {
  id: string
  invoiceId: string
  amount: number
  paidAt: string
  paymentMethod?: string
  paperIdPaymentId?: string
  paperIdStatus?: string
  /** Manual payments: uploaded proof URL + free-text note. */
  proofUrl?: string
  note?: string
  createdAt: string
}

export interface PaymentWithInvoice extends Payment {
  invoice: Invoice | null
  member: Member | null
}

// ---------------------------------------------------------------------------
// Audit log
// ---------------------------------------------------------------------------

export type AuditAction =
  | 'created'
  | 'sent'
  | 'paid'
  | 'cancelled'
  | 'overdue'
  | 'updated'
  | 'reminded'
  | 'terminated'
  | 'tax_invoice'
  | 'renewal_submitted'

export interface AuditLogEntry {
  id: string
  invoiceId: string
  action: AuditAction
  oldStatus?: InvoiceStatus
  newStatus?: InvoiceStatus
  actorId?: string
  actorName?: string
  notes?: string
  createdAt: string
}

// ---------------------------------------------------------------------------
// Dashboard aggregates
// ---------------------------------------------------------------------------

export interface ChapterStat {
  chapterId: string
  chapterName: string
  total: number
  paid: number
  outstanding: number
  overdue: number
  totalAmount: number
  /** Per tipe, tanpa yang dibatalkan. Harganya jauh berbeda, jadi nominalnya ikut dibawa. */
  renewal: number
  renewalAmount: number
  registration: number
  registrationAmount: number
}

export interface DashboardSummary {
  /** All invoices (excluding cancelled) — count + total value. */
  total: { count: number; amount: number; trend: number }
  /** All paid invoices — count + collected revenue. */
  paid: { count: number; amount: number; trend: number }
  /** Issued but unpaid (sent + overdue). */
  outstanding: { count: number; amount: number; trend: number }
  overdue: { count: number; amount: number; trend: number }
  renewalDue: { count: number; trend: number }
  /** Breakdown for the payment-status donut chart. */
  statusBreakdown: { status: InvoiceStatus; count: number }[]
  /** Last 6 months of issued vs. paid totals. */
  monthly: { month: string; issued: number; paid: number }[]
  /** Per-chapter breakdown. */
  chapterStats: ChapterStat[]
}

// ---------------------------------------------------------------------------
// Shared query / pagination shapes
// ---------------------------------------------------------------------------

export interface InvoiceFilters {
  /** `outstanding` adalah jalan pintas server untuk sent + overdue. */
  status?: InvoiceStatus | 'all' | 'outstanding'
  type?: InvoiceType | 'all'
  chapterId?: string | 'all'
  search?: string
  /** Umur tunggakan; dihitung server terhadap tanggal basis data. */
  aging?: '1-30' | '31-60' | '60+' | 'all'
  /** Invoice renewal lunas yang belum (`pending`) atau sudah (`done`) diajukan ke BNI Global. */
  submission?: 'pending' | 'done'
  dueFrom?: string
  dueTo?: string
  issuedFrom?: string
  issuedTo?: string
  /** Tanggal dikirim ke Paper.id. Mengisinya membuang invoice yang belum dikirim. */
  sentFrom?: string
  sentTo?: string
  /** `sent` mengurutkan menurut tanggal kirim, terbaru dulu. */
  sort?: 'sent'
  /** Hanya invoice milik member-member ini. */
  memberIds?: string[]
  limit?: number
  offset?: number
  /**
   * Minta agregat untuk seluruh hasil filter. Bawaannya mati.
   *
   * Satu query agregat tambahan di server, jadi pemanggil yang hanya butuh
   * barisnya — ekspor, misalnya — tidak membayarnya.
   */
  summary?: boolean
}

/**
 * Hitungan dan jumlah untuk SELURUH hasil filter, bukan halaman yang diterima.
 *
 * Dihitung server. Begitu paginasi pindah ke server, klien hanya memegang satu
 * halaman — menghitung kartu ringkasan darinya akan menampilkan angka jauh
 * lebih kecil daripada kenyataannya, tanpa satu pun tanda bahwa itu keliru.
 */
export interface InvoiceBucket {
  count: number
  amount: number
}

export interface InvoiceSummary {
  byStatus: Record<string, InvoiceBucket>
  /**
   * Rincian pendaftaran vs renewal.
   *
   * Berbeda dari byStatus, ini MENGIKUTI tab status yang sedang dipilih. Tab
   * bertanya "ada berapa di tiap status" — jadi menyaringnya lebih dulu membuat
   * tab lain selalu nol. Kartu bertanya "dari yang sedang saya lihat, berapa
   * pendaftaran dan berapa renewal".
   */
  byType: Record<string, InvoiceBucket>
  /** Tidak termasuk cancelled dan terminated — yang batal bukan yang ditagih. */
  total: InvoiceBucket
}

/** Satu halaman invoice, beserta angka untuk keseluruhan hasil filter. */
export interface InvoicePage {
  rows: InvoiceWithRelations[]
  total: number
  summary?: InvoiceSummary
}

/** Saringan daftar member. Seluruhnya dijalankan server. */
export interface MemberFilters {
  search?: string
  chapterId?: string | 'all'
  status?: MemberStatus | 'all'
  /** `visitor` mencakup tamu dan yang pendaftarannya masih diproses. */
  kelompok?: 'all' | 'member' | 'visitor'
  kontak?: 'all' | 'no-phone' | 'no-email' | 'no-contact'
  /** Hanya member yang tanggal jatuh temponya terisi. */
  punyaJatuhTempo?: boolean
  dueFrom?: string
  dueTo?: string
  limit?: number
  offset?: number
  /** Minta hitungan per status untuk seluruh hasil filter. */
  summary?: boolean
}

/** Satu halaman member, beserta hitungan untuk keseluruhan hasil filter. */
export interface MemberPage {
  rows: MemberWithChapter[]
  total: number
  /** Dihitung TANPA saringan status, supaya kartu status lain tidak nol. */
  summary?: { byStatus: Partial<Record<MemberStatus, number>> }
}

/** Saringan daftar pembayaran. Seluruhnya dijalankan server. */
export interface PaymentFilters {
  /** Nomor invoice atau nama member. */
  search?: string
  method?: string | 'all'
  chapterId?: string | 'all'
  /** Tipe invoice yang dibayar. */
  type?: InvoiceType | 'all'
  paidFrom?: string
  paidTo?: string
  limit?: number
  offset?: number
  summary?: boolean
}

export interface PaymentSummary {
  count: number
  amount: number
  byType: Partial<Record<InvoiceType, number>>
  /** Metode yang ada di hasil filter tanpa saringan metodenya. */
  methods: string[]
}

/** Satu halaman pembayaran, beserta jumlah untuk keseluruhan hasil filter. */
export interface PaymentPage {
  rows: PaymentWithInvoice[]
  total: number
  summary?: PaymentSummary
}

/** Periode dan tipe laporan. `from` dan `to` inklusif, berformat YYYY-MM-DD. */
export interface ReportFilters {
  from?: string
  to?: string
  type?: InvoiceType | 'all'
  limit?: number
  offset?: number
}

export interface ReportChapterRow {
  id: string
  name: string
  count: number
  ditagih: number
  /** Ditagih, dipecah per tipe. renewal + pendaftaran === ditagih. */
  renewal: number
  pendaftaran: number
  diterima: number
  outstanding: number
  rate: number
}

/** Seluruh angka halaman Laporan selain tabel detail. */
export interface ReportSummary {
  count: number
  ditagih: number
  diterima: number
  outstanding: number
  rate: number
  /** Setiap chapter, termasuk yang tanpa invoice. Urutannya tidak dijamin. */
  chapters: ReportChapterRow[]
  types: Record<InvoiceType, number>
  monthly: { month: string; issued: number; paid: number }[]
  methods: { method: string; count: number; amount: number }[]
  /** Jumlah kolom diterima di SELURUH baris detail. */
  detailDiterima: number
}

/** Satu baris tabel detail: satu invoice yang terbit pada periode. */
export interface ReportDetailRow {
  id: string
  number: string
  member: string | null
  chapter: string | null
  type: InvoiceType
  terbit: string
  jatuhTempo: string
  status: InvoiceStatus
  ditagih: number
  /** Pembayaran yang DITERIMA pada periode untuk invoice ini. */
  diterima: number
}

export interface RenewalDueMember extends MemberWithChapter {
  /** The most recent invoice whose period is ending. */
  lastInvoice: Invoice
  daysUntilDue: number
}

/** Peran pengguna. Admin = kontrol penuh, User = lihat & ekspor saja. */
/**
 * `st` = Secretary/Treasurer, `mc` = Membership Committee — keduanya pengurus
 * SATU chapter, bukan nasional. Lingkupnya ada di `chapterId` pada user.
 */
export type UserRole = 'admin' | 'st' | 'mc' | 'user'

/**
 * Akun sebagaimana dilihat halaman Pengguna.
 *
 * `chapterId` null berarti nasional — itulah yang dipakai `admin` dan `user`.
 * Untuk `st` dan `mc` ia WAJIB terisi: backend menolak peran berlingkup chapter
 * tanpa chapter, karena lingkup yang kosong bisa terbaca sebagai "tanpa batas".
 */
/** Apa yang akan terjadi pada satu baris berkas impor. */
export type ImportTindakan = 'baru' | 'diperbarui' | 'sama' | 'ditolak'

export interface ImportBaris {
  /** Nomor baris SEPERTI DI EXCEL, supaya bisa langsung dicari di berkasnya. */
  nomor: number
  id: string
  nama: string
  tindakan: ImportTindakan
  alasan?: string
  perubahan?: string[]
}

/**
 * Hasil pemeriksaan berkas impor.
 *
 * `diterapkan` membedakan pratinjau dari penulisan — dan itu yang membuat
 * pratinjau bisa dipercaya: angkanya dihitung kode yang sama dengan yang
 * menulis, jadi tidak mungkin berbeda.
 */
export interface ImportHasil {
  format: 'csv' | 'xlsx'
  jenis: 'chapters' | 'members'
  diterapkan: boolean
  total: number
  baru: number
  diperbarui: number
  sama: number
  ditolak: number
  baris: ImportBaris[]
  peringatan?: string[]
  /** Hanya untuk laporan BNI Connect; baru = chapter dibuat dari laporannya. */
  chapter?: { id: string; nama: string; baru: boolean }
}

/** Jawaban MC atas permintaan konfirmasi renewal. */
export type RenewalAnswer = 'pending' | 'will_renew' | 'will_not' | 'unsure'

/**
 * Periode konfirmasi renewal dari master data. Admin yang menambah, mengubah,
 * dan menghapusnya; ST dan MC memilihnya di halaman Konfirmasi Renewal.
 */
export interface RenewalPeriod {
  /** Teks yang tersimpan di RenewalRequest.period, misalnya "2029-S2". */
  id: string
  name: string
  startDate: string
  endDate: string
  /** Permintaan konfirmasi di periode ini, dalam lingkup chapter pengguna. */
  requestCount: number
}

export interface SimpanPeriodeInput {
  name: string
  startDate: string
  endDate: string
}

/**
 * Satu permintaan konfirmasi renewal.
 *
 * ST menanyakan, MC menjawab. `period` adalah id periode di master data,
 * bukan tanggal ST menanyakannya, karena itulah yang membedakan satu
 * permintaan dari permintaan periode berikutnya.
 */
export interface RenewalRequest {
  id: string
  memberId: string
  chapterId: string
  period: string
  requestedBy: string | null
  requestedAt: string
  assignedMc: string | null
  /** Nama akun MC yang ditugaskan, diisi server. */
  assignedMcName?: string | null
  answer: RenewalAnswer
  answeredBy: string | null
  answeredAt: string | null
  note: string | null
  memberName: string | null
  chapterName: string | null
  renewalDate: string | null
  createdAt: string
  updatedAt: string
  /** Kapan dan oleh siapa ST meminta invoice ke finance. */
  invoiceRequestedAt?: string | null
  invoiceRequestedBy?: string | null
  /** Tahap member ini dalam alur renewal, dihitung server. */
  stage: RenewalStage
  /** Invoice renewal yang terbit untuk konfirmasi ini. */
  invoiceId?: string | null
  invoiceNumber?: string | null
  invoiceStatus?: InvoiceStatus | null
}

/**
 * Tahap dalam alur renewal klien. Empat tahap awal sama dengan jawaban MC,
 * sisanya menyusul setelah MC menjawab Diterima.
 */
export type RenewalStage =
  | RenewalAnswer
  | 'invoice_requested'
  | 'invoice_draft'
  | 'invoiced'
  | 'proof_pending'
  | 'paid'
  | 'submitted'

export type ProofStatus = 'pending' | 'approved' | 'rejected'

/** Bukti bayar yang dikirim ST dan diperiksa finance. */
export interface PaymentProof {
  id: string
  invoiceId: string
  chapterId: string
  amount: number
  paidAt: string
  paymentMethod?: string | null
  proofUrl: string
  note?: string | null
  status: ProofStatus
  submittedBy?: string | null
  submittedAt: string
  reviewedBy?: string | null
  reviewedAt?: string | null
  reviewNote?: string | null
  paymentId?: string | null
  invoiceNumber: string
  invoiceAmount: number
  invoiceStatus: InvoiceStatus
  memberName?: string | null
  chapterName?: string | null
}

export interface ManagedUser {
  id: string
  email: string
  name: string
  role: UserRole
  chapterId: string | null
  createdAt: string
  updatedAt: string
}

export interface AuthUser {
  id: string
  name: string
  email: string
  role: UserRole
  /**
   * Chapter pengguna; null berarti nasional (admin dan user).
   *
   * Backend sudah mengirimkannya sejak peran ST/MC ditambahkan, tapi tipe ini
   * belum membawanya — sehingga UI sama sekali tidak tahu chapter pengguna yang
   * sedang masuk, dan tidak bisa menyaring apa pun berdasarkan itu.
   */
  chapterId?: string | null
}

/**
 * Angka ringkas per chapter, dihitung server.
 *
 * Halaman daftar chapter dulu menghitungnya sendiri dengan menarik seluruh
 * member dan seluruh invoice — dua daftar yang sama-sama dipaku pada 200 baris.
 * Dampaknya bukan sekadar lambat: jumlah member berhenti bertambah di 200, dan
 * tunggakannya hanya menjumlahkan 200 invoice pertama.
 */
export interface ChapterCounts {
  chapterId: string
  memberCount: number
  /** NOMINAL invoice sent + overdue, bukan cacahnya. */
  outstanding: number
}
