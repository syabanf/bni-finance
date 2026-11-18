import { ApiError, api, fileUrl, query, type ListResponse } from '@/lib/apiClient'
import { todayISO } from '@/lib/date'
import type { CreateInvoiceInput, InvoiceRepository, ManualPaymentInput } from '@/services/types'
import type {
  AuditLogEntry,
  Chapter,
  Invoice,
  InvoiceFilters,
  InvoiceSummary,
  InvoiceWithRelations,
  MemberWithChapter,
  RenewalDueMember,
} from '@/types'
import { isNotFound } from './chapterRepository'
import { MAKS_PER_HALAMAN, ambilPerId, tarikSemua, tarikUntuk } from './kumpulan'

/**
 * Daftar chapter disimpan sebentar.
 *
 * Chapter sedikit dan jarang berubah, sementara tiap halaman invoice
 * membutuhkannya untuk menampilkan nama chapter. Umurnya pendek supaya chapter
 * yang baru dibuat muncul tanpa memuat ulang halaman.
 */
const UMUR_CHAPTER = 60_000
let chapterTersimpan: { pada: number; isi: Promise<Map<string, Chapter>> } | null = null

function daftarChapter(): Promise<Map<string, Chapter>> {
  const sekarang = Date.now()
  // Menyimpan PROMISE-nya, bukan hasilnya: tanpa itu, dua panggilan yang
  // berbarengan sama-sama melihat cache kosong dan sama-sama menembak jaringan.
  if (chapterTersimpan && sekarang - chapterTersimpan.pada < UMUR_CHAPTER) return chapterTersimpan.isi

  const isi = tarikSemua<Chapter>('/chapters').then((c) => new Map(c.map((x) => [x.id, x])))
  // Kegagalan tidak boleh mengendap di cache dan membuat halaman rusak selama
  // semenit penuh; permintaan berikutnya harus boleh mencoba lagi.
  isi.catch(() => {
    if (chapterTersimpan?.isi === isi) chapterTersimpan = null
  })
  chapterTersimpan = { pada: sekarang, isi }
  return isi
}

/**
 * Menempelkan member dan chapter ke invoice.
 *
 * API mengembalikan invoice tanpa relasinya. Member diminta hanya untuk
 * invoice yang sedang dipegang: satu halaman berisi 25 invoice berarti satu
 * permintaan berisi paling banyak 25 id. Sebelumnya SELURUH member ditarik
 * untuk keperluan ini, pada setiap pemuatan halaman invoice.
 */
async function lampirkan(invoices: Invoice[]): Promise<InvoiceWithRelations[]> {
  if (invoices.length === 0) return []
  const [members, chapters] = await Promise.all([
    ambilPerId<MemberWithChapter>(
      '/members',
      invoices.map((i) => i.memberId),
    ),
    daftarChapter(),
  ])
  return invoices.map((invoice) => ({
    ...invoice,
    // Path /uploads/... milik backend; diberi alamatnya supaya tautan faktur
    // tetap benar saat backend berada di origin lain (sama seperti bukti bayar).
    taxInvoiceUrl: fileUrl(invoice.taxInvoiceUrl),
    member: members.get(invoice.memberId) ?? null,
    chapter: chapters.get(invoice.chapterId) ?? null,
  }))
}

/** Invoice terkirim yang sudah lewat jatuh tempo, tetapi statusnya belum overdue. */
function yangTelat(invoices: Invoice[]): Invoice[] {
  const today = todayISO()
  return invoices.filter((i) => i.status === 'sent' && i.dueDate < today)
}

/**
 * Menampilkan invoice telat sebagai overdue TANPA menulisnya ke server.
 *
 * Untuk daftar lengkap. Di sana syncOverdue berarti satu PATCH per invoice
 * telat, sekaligus: diukur pada 2.143 invoice telat, peramban menolak ribuan
 * permintaan dengan ERR_INSUFFICIENT_RESOURCES. Statusnya tetap ditulis saat
 * invoice itu tampil di daftar berhalaman, paling banyak satu halaman sekali.
 */
function tampilkanOverdue(invoices: Invoice[]): void {
  for (const invoice of yangTelat(invoices)) invoice.status = 'overdue'
}

/** Marks sent invoices past their due date as overdue, matching the UI's view. */
async function syncOverdue(invoices: Invoice[]): Promise<void> {
  const stale = yangTelat(invoices)
  await Promise.all(
    stale.map((i) =>
      api.patch<Invoice>(`/invoices/${i.id}`, { status: 'overdue' }).catch(() => {
        // A failed sweep must not break the list the user asked for; the next
        // load tries again.
      }),
    ),
  )
  for (const invoice of stale) invoice.status = 'overdue'
}

/**
 * Menentukan apakah sebuah operasi yang GAGAL DI SISI KLIEN sebenarnya berhasil
 * di server.
 *
 * Panggilan ke Paper.id terukur 5 sampai 36 detik untuk operasi yang sama
 * persis. Kalau koneksi putus atau timeout di tengahnya, klien melihat
 * kegagalan sementara server menyelesaikannya dengan tenang — dan itu bukan
 * hipotesis: pada uji kirim, curl melaporkan gagal untuk dua dari tiga tagihan
 * yang ternyata SEMUANYA sampai ke penerima.
 *
 * Melaporkan gagal padahal berhasil adalah kesalahan yang mahal. Operator akan
 * mengirim ulang; Paper.id membakar nomor invoice secara permanen, jadi nomor
 * kedua ikut terbakar, dan member menerima tagihan yang sama dua kali.
 *
 * Karena itu kegagalan JARINGAN — ApiError berstatus 0, bukan penolakan server
 * yang punya kode HTTP jelas — tidak langsung dipercaya. Invoice dibaca ulang
 * beberapa kali untuk melihat apakah operasinya benar-benar mendarat.
 *
 * Jeda totalnya sekitar dua puluh detik, dan itu disengaja: menunggu selama itu
 * jauh lebih murah daripada satu nomor terbakar plus satu pesan ganda ke member.
 */
async function reconcileNetworkFailure(
  err: unknown,
  id: string,
  landed: (after: Invoice) => boolean,
): Promise<Invoice> {
  // Penolakan server punya kode HTTP dan bisa dipercaya apa adanya — hanya
  // kegagalan jaringan yang ambigu.
  if (!(err instanceof ApiError) || err.status !== 0) throw err

  for (const jeda of [2000, 4000, 6000, 8000]) {
    await new Promise((r) => setTimeout(r, jeda))
    try {
      const after = await api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
      if (landed(after)) return after
    } catch {
      // Server masih tidak terjangkau; coba lagi pada jeda berikutnya.
    }
  }
  throw err
}

/** Menyusun query dari filter UI; "all" berarti tidak menyaring. */
function paramFilter(f: InvoiceFilters | undefined) {
  const nilai = (v?: string) => (v && v !== 'all' ? v : undefined)
  return {
    status: nilai(f?.status),
    type: nilai(f?.type),
    chapterId: nilai(f?.chapterId),
    aging: nilai(f?.aging),
    q: f?.search?.trim() || undefined,
    dueFrom: f?.dueFrom || undefined,
    dueTo: f?.dueTo || undefined,
    issuedFrom: f?.issuedFrom || undefined,
    issuedTo: f?.issuedTo || undefined,
  }
}

export const apiInvoiceRepository: InvoiceRepository = {
  async listPaged(filters) {
    const res = await api.get<ListResponse<Invoice> & { meta: { summary?: InvoiceSummary } }>(
      `/invoices${query({
        ...paramFilter(filters),
        summary: filters?.summary ? 'true' : undefined,
        limit: Math.min(filters?.limit ?? 25, MAKS_PER_HALAMAN),
        offset: filters?.offset ?? 0,
      })}`,
    )
    await syncOverdue(res.data)

    return {
      rows: await lampirkan(res.data),
      total: res.meta.total,
      summary: res.meta.summary,
    }
  },

  async list(filters) {
    // SELURUH hasil filter, bukan 200 baris pertama: pemanggilnya (laporan,
    // ekspor) menjumlahkan uang dari daftar ini.
    const semua = await tarikSemua<Invoice>('/invoices', paramFilter(filters))
    tampilkanOverdue(semua)
    return lampirkan(semua)
  },

  async getById(id) {
    let invoice: Invoice
    try {
      invoice = await api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
    } catch (err) {
      if (isNotFound(err)) return null
      throw err
    }
    const [lengkap] = await lampirkan([invoice])
    return lengkap
  },

  async listByMember(memberId) {
    return tarikSemua<Invoice>('/invoices', { memberId })
  },

  async create(input: CreateInvoiceInput) {
    // chapterId isn't in the input — it comes from the member's chapter.
    const member = await api.get<MemberWithChapter>(`/members/${encodeURIComponent(input.memberId)}`)
    return api.post<Invoice>('/invoices', {
      memberId: input.memberId,
      chapterId: member.chapterId,
      type: input.type,
      amount: input.amount,
      dueDate: input.dueDate,
      periodStart: input.periodStart,
      periodEnd: input.periodEnd,
      notes: input.notes,
    })
  },

  async send(id) {
    const invoice = await api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
    if (invoice.status !== 'draft') throw new Error('Hanya invoice draft yang bisa dikirim')

    // Dulu di sini ada percabangan Self Payment Mode: bila menyala, invoice
    // hanya ditandai terkirim tanpa mendorong apa pun ke Paper.id. Jalur itu
    // hilang bersama pembayaran mandiri, tetapi pembacaan setelannya sempat
    // tertinggal — memanggil app_settings untuk kunci yang sudah tidak ada,
    // lalu selalu jatuh ke cabang yang sama. Bekerja, tetapi menipu pembaca.
    // The Paper.id credentials live on the server; this endpoint does the push,
    // stores the returned payment link + PDF, and records the audit entry.
    //
    // The body is deliberately empty: which channels deliver the invoice is an
    // operational policy read from app_settings by the server, not a per-click
    // choice. Passing flags from here would mean this path, the bulk actions in
    // the invoice list, and "create + send" each had to remember them — and the
    // one that forgot would quietly stop reaching members while still
    // reporting success. Set the channels in Pengaturan.
    try {
      return await api.post<Invoice>(`/invoices/${encodeURIComponent(id)}/send`, {})
    } catch (err) {
      // Berhasil bila invoice sudah tidak lagi draft — server memindahkannya
      // ke sent dalam satu transaksi bersama pencatatan hasil Paper.id.
      return reconcileNetworkFailure(err, id, (after) => after.status !== 'draft')
    }
  },

  /**
   * Pengingat: minta Paper.id mengantar tagihan ini lagi ke member.
   *
   * Dulu fungsi ini tidak melakukan apa pun — ia mengambil invoice, memeriksa
   * statusnya, lalu mengembalikannya utuh dengan alasan "tautannya masih
   * berlaku, tidak ada yang perlu dibuat ulang". Benar soal tautannya, tetapi
   * itu bukan yang diminta: menekan Kirim Ulang tidak pernah membuat satu pesan
   * pun sampai ke member, sambil melaporkan sukses.
   *
   * Sekarang benar-benar mengirim. Backend menerbitkan dokumen baru di Paper.id
   * dengan nomor turunan (-R1, -R2) karena Paper.id membakar nomor secara
   * permanen dan tidak punya endpoint pengingat. Status invoice tidak berubah.
   */
  async resend(id) {
    // Penghitung dibaca lebih dulu supaya kenaikannya bisa dipakai sebagai
    // bukti bahwa pengingat benar-benar terkirim, seandainya koneksi putus.
    let sebelum = 0
    try {
      sebelum = (await api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)).paperIdReminderCount ?? 0
    } catch {
      // Gagal membaca keadaan awal bukan alasan membatalkan pengiriman.
    }
    try {
      return await api.post<Invoice>(`/invoices/${encodeURIComponent(id)}/remind`, {})
    } catch (err) {
      return reconcileNetworkFailure(err, id, (after) => (after.paperIdReminderCount ?? 0) > sebelum)
    }
  },

  async updateDraft(id, input) {
    return api.patch<Invoice>(`/invoices/${encodeURIComponent(id)}`, input)
  },

  async remove(id) {
    await api.delete(`/invoices/${encodeURIComponent(id)}`)
  },

  async cancel(id, reason) {
    return api.patch<Invoice>(`/invoices/${encodeURIComponent(id)}`, {
      status: 'cancelled',
      cancelReason: reason,
    })
  },

  async terminate(id, reason) {
    // Memakai kolom cancelReason yang sama: alasannya satu jenis informasi,
    // dan yang membedakan keduanya adalah STATUS-nya, bukan tempat alasannya
    // disimpan.
    return api.patch<Invoice>(`/invoices/${encodeURIComponent(id)}`, {
      status: 'terminated',
      cancelReason: reason,
    })
  },

  async markPaid(id) {
    const invoice = await api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
    // Recording the payment settles the invoice in the same transaction, so
    // there is no separate status update to make.
    await api.post('/payments', {
      invoiceId: id,
      amount: invoice.amount,
      paymentMethod: 'paper_id',
    })
    return api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
  },

  async recordManualPayment(id, input: ManualPaymentInput) {
    await api.post('/payments', {
      invoiceId: id,
      amount: input.amount,
      paidAt: new Date(input.paidAt).toISOString(),
      paymentMethod: input.method,
      proofUrl: input.proofUrl,
      note: input.note,
    })
    return api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
  },

  async attachTaxInvoice(id, file) {
    // Dua langkah: berkasnya disimpan dulu, lalu path-nya ditempel ke invoice.
    // Server menolak path yang bukan hasil /uploads dan invoice yang belum lunas.
    const { url } = await api.upload('/uploads', file)
    const inv = await api.put<Invoice>(`/invoices/${encodeURIComponent(id)}/tax-invoice`, { url })
    return { ...inv, taxInvoiceUrl: fileUrl(inv.taxInvoiceUrl) }
  },

  async getAuditLog(invoiceId) {
    const res = await api.get<ListResponse<AuditLogEntry>>(
      `/invoices/${encodeURIComponent(invoiceId)}/audit${query({ limit: 200 })}`,
    )
    return res.data
  },

  async renewalDue(withinDays = 30) {
    // 500 adalah batas atas endpoint ini di server.
    const due = await api.get<ListResponse<MemberWithChapter & { daysUntilDue: number }>>(
      `/members/renewal-due${query({ days: withinDays, limit: 500 })}`,
    )
    // Invoice diminta hanya untuk member yang jatuh tempo. Sebelumnya 200
    // invoice terbaru dianggap seluruhnya, jadi member yang invoice
    // terakhirnya di luar 200 itu hilang dari daftar ini.
    const invoices = await tarikUntuk<Invoice>(
      '/invoices',
      'memberIds',
      due.data.map((m) => m.id),
    )

    // Newest invoice per member — the one whose period is ending.
    const latest = new Map<string, Invoice>()
    for (const invoice of invoices) {
      const existing = latest.get(invoice.memberId)
      if (!existing || invoice.periodEnd > existing.periodEnd) latest.set(invoice.memberId, invoice)
    }

    return due.data
      .map((member): RenewalDueMember | null => {
        const lastInvoice = latest.get(member.id)
        // The type requires an invoice; a member who has never been billed has
        // nothing to renew yet.
        if (!lastInvoice) return null
        return { ...member, lastInvoice, daysUntilDue: member.daysUntilDue }
      })
      .filter((m): m is RenewalDueMember => m !== null)
  },
}
