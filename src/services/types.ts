/**
 * Repository contracts — the boundary between the presentation layer and the
 * data source. Pages and hooks depend ONLY on these interfaces. Concrete
 * implementations live under `services/mock` today and can be replaced with
 * HTTP implementations later (see `services/index.ts`).
 *
 * Everything is async so swapping to a network-backed source is a no-op for
 * callers.
 */

import type {
  ImportHasil,
  ManagedUser,
  RenewalAnswer,
  RenewalRequest,
  UserRole,
  AuditLogEntry,
  AuthUser,
  Chapter,
  ChapterCounts,
  DashboardSummary,
  FeeSettings,
  Invoice,
  InvoiceFilters,
  InvoiceType,
  InvoicePage,
  InvoiceWithRelations,
  MemberWithChapter,
  PaymentWithInvoice,
  RenewalDueMember,
} from '@/types'

export interface AuthRepository {
  login(email: string, password: string): Promise<AuthUser>
  logout(): Promise<void>
  getCurrentUser(): AuthUser | null
  /** Update the signed-in user's display name. */
  updateProfile(input: { name: string }): Promise<AuthUser>
  /** Set a new password for the signed-in user. */
  updatePassword(newPassword: string): Promise<void>
}

/**
 * Pengelolaan akun — dipakai halaman Pengguna.
 *
 * Ada karena tanpanya peran ST dan MC tidak bisa dibuat sama sekali dari
 * aplikasi: batas chapter sudah ditegakkan backend, tapi akun yang memakainya
 * hanya bisa dibuat lewat curl.
 */
export interface UserRepository {
  list(): Promise<ManagedUser[]>
  create(input: CreateUserInput): Promise<ManagedUser>
  /** Ganti peran (dan chapter-nya bila peran itu berlingkup chapter). */
  changeRole(id: string, role: UserRole, chapterId: string | null): Promise<ManagedUser>
  remove(id: string): Promise<void>
}

export interface CreateUserInput {
  email: string
  name: string
  password: string
  role: UserRole
  /** Wajib untuk `st` dan `mc`, harus kosong untuk `admin` dan `user`. */
  chapterId: string | null
}

/**
 * Konfirmasi renewal — ST menanyakan, MC menjawab.
 *
 * ST TIDAK boleh menjawab, dan itu inti alurnya: ia yang bertanya, jadi
 * membiarkannya menjawab sendiri membuat konfirmasinya tidak berarti apa-apa.
 * Aturan itu ditegakkan server; di sini hanya tombolnya yang disembunyikan.
 */
/**
 * Impor chapter dan member dari berkas.
 *
 * SELALU dua langkah: `preview` tidak pernah menulis. Impor yang langsung
 * menulis menyembunyikan chapter yang salah ketik, kolom yang tergeser, dan id
 * ganda di balik satu kalimat "berhasil" — dan pada data keanggotaan, itu baru
 * ketahuan saat tagihannya salah kirim.
 */
export interface ImportRepository {
  preview(jenis: 'chapters' | 'members', file: File): Promise<ImportHasil>
  apply(jenis: 'chapters' | 'members', file: File): Promise<ImportHasil>
}

export interface RenewalRepository {
  list(params?: { answer?: RenewalAnswer; period?: string }): Promise<RenewalRequest[]>
  /**
   * ST meminta konfirmasi untuk sekumpulan member.
   *
   * `assignedMc` boleh null — permintaan tetap terlihat oleh SELURUH MC di
   * chapter itu. Menuntutnya terisi akan menghentikan ST yang chapternya belum
   * punya akun MC sama sekali.
   */
  request(
    memberIds: string[],
    period: string,
    assignedMc?: string | null,
  ): Promise<{ dibuat: number; dilewati: number; total: number }>
  /** MC menjawab satu permintaan. */
  answer(id: string, answer: RenewalAnswer, note?: string): Promise<RenewalRequest>
}

export interface ChapterRepository {
  list(): Promise<Chapter[]>
  /** Jumlah member dan nominal tunggakan tiap chapter, dihitung server. */
  counts(): Promise<ChapterCounts[]>
  getById(id: string): Promise<Chapter | null>
  /** Pull fresh data from BNI VM and refresh the local mirror. */
  sync(): Promise<{ count: number; syncedAt: string }>
}

export interface MemberRepository {
  list(params?: { chapterId?: string; search?: string }): Promise<MemberWithChapter[]>
  getById(id: string): Promise<MemberWithChapter | null>
  /** Members eligible for a new registration invoice (no active one yet). */
  eligibleForRegistration(): Promise<MemberWithChapter[]>
  sync(): Promise<{ count: number; syncedAt: string }>
}

export interface InvoiceRepository {
  list(filters?: InvoiceFilters): Promise<InvoiceWithRelations[]>
  /**
   * Satu halaman, disaring dan dihitung SERVER.
   *
   * Terpisah dari list() dengan sengaja. list() menarik apa adanya dan dipakai
   * pemanggil yang memang butuh seluruh baris; ia juga masih terpaku pada batas
   * 200 baris — potongan diam yang belum diperbaiki untuk Dasbor, daftar
   * Chapter, dan Notifikasi.
   */
  listPaged(filters?: InvoiceFilters): Promise<InvoicePage>
  getById(id: string): Promise<InvoiceWithRelations | null>
  listByMember(memberId: string): Promise<Invoice[]>
  create(input: CreateInvoiceInput): Promise<Invoice>
  /** Push to Paper.id and move the invoice to `sent`. */
  send(id: string): Promise<Invoice>
  /** Re-send an already-sent/overdue invoice to Paper.id (refreshes payment link). */
  resend(id: string): Promise<Invoice>
  cancel(id: string, reason: string): Promise<Invoice>
  /**
   * Memutus keanggotaan: invoicenya gugur karena hubungannya berakhir.
   *
   * Berbeda dari `cancel`, dan bedanya bukan kosmetik — `cancel` adalah
   * pembatalan biasa (salah terbit, member menunda), sedangkan ini menandai
   * tagihan yang gugur karena keanggotaannya diputus. Menyatukan keduanya
   * membuat laporan tidak bisa lagi membedakan tagihan yang batal dari
   * hubungan yang berakhir.
   */
  terminate(id: string, reason: string): Promise<Invoice>
  /** Simulate a Paper.id "payment.success" webhook for a sent invoice. */
  markPaid(id: string): Promise<Invoice>
  /** Manually record an offline payment (e.g. bank transfer) with optional proof. */
  recordManualPayment(id: string, input: ManualPaymentInput): Promise<Invoice>
  getAuditLog(invoiceId: string): Promise<AuditLogEntry[]>
  /** Members at/near the end of their membership period. */
  renewalDue(withinDays?: number): Promise<RenewalDueMember[]>
}

export interface CreateInvoiceInput {
  memberId: string
  type: InvoiceType
  amount: number
  dueDate: string
  periodStart: string
  periodEnd: string
  notes?: string
}

export interface ManualPaymentInput {
  amount: number
  paidAt: string
  method: string
  note?: string
  /** URL of the uploaded payment proof (from PaymentRepository.uploadProof). */
  proofUrl?: string
}

export interface SettingsRepository {
  getFees(): Promise<FeeSettings>
  updateFees(input: Pick<FeeSettings, 'registrationFee' | 'renewalFee' | 'notes'>): Promise<FeeSettings>
}

export interface PaymentRepository {
  list(): Promise<PaymentWithInvoice[]>
  listByInvoice(invoiceId: string): Promise<PaymentWithInvoice[]>
  /** Upload a payment-proof file, returning a URL to store with the payment. */
  uploadProof(file: File): Promise<string>
}

export interface UrgentCount {
  overdue: number
  renewalDue: number
  total: number
}

export interface DashboardRepository {
  summary(): Promise<DashboardSummary>
}
