import { ApiError, api, query, type ListResponse } from '@/lib/apiClient'
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

/**
 * The API returns bare invoices; the UI wants member and chapter attached.
 * Rather than a join per invoice, the reference tables are fetched once per
 * call and indexed — they are small (hundreds of rows) and change rarely.
 */
/**
 * Menarik SELURUH baris sebuah daftar, bukan hanya halaman pertama.
 *
 * Versi sebelumnya meminta satu halaman berisi 200 baris dan memperlakukannya
 * sebagai keseluruhan. Untuk member, akibatnya tidak terlihat sebagai galat:
 * invoice milik member ke-201 ke atas tetap tampil, hanya saja kolom namanya
 * jadi "—" — dan itu terbaca sebagai data member yang rusak, bukan sebagai
 * daftar yang terpotong.
 */
async function tarikSemua<T>(jalur: string, batasAman = 5000): Promise<T[]> {
  const kumpulan: T[] = []
  for (;;) {
    const res = await api.get<ListResponse<T>>(`${jalur}${query({ limit: 200, offset: kumpulan.length })}`)
    if (res.data.length === 0) break
    kumpulan.push(...res.data)
    if (kumpulan.length >= res.meta.total || kumpulan.length >= batasAman) break
  }
  return kumpulan
}

interface Direktori {
  members: Map<string, MemberWithChapter>
  chapters: Map<string, Chapter>
}

/**
 * Tabel rujukan disimpan sebentar.
 *
 * Sebelumnya member dan chapter ditarik ulang pada SETIAP panggilan invoice.
 * Itu masih tertutupi ketika halaman hanya memuat sekali; dengan paginasi di
 * server, tiap pindah halaman dan tiap potongan ekspor membayar dua permintaan
 * tambahan untuk data yang praktis tidak berubah.
 *
 * Umurnya pendek dengan sengaja — member yang baru dibuat harus muncul tanpa
 * perlu memuat ulang halaman.
 */
const UMUR_DIREKTORI = 60_000
let direktori: { pada: number; isi: Promise<Direktori> } | null = null

async function loadDirectory(): Promise<Direktori> {
  const sekarang = Date.now()
  // Menyimpan PROMISE-nya, bukan hasilnya: tanpa itu, dua panggilan yang
  // berbarengan sama-sama melihat cache kosong dan sama-sama menembak jaringan.
  if (direktori && sekarang - direktori.pada < UMUR_DIREKTORI) return direktori.isi

  const isi = (async () => {
    const [members, chapters] = await Promise.all([
      tarikSemua<MemberWithChapter>('/members'),
      tarikSemua<Chapter>('/chapters'),
    ])
    return {
      members: new Map(members.map((m) => [m.id, m])),
      chapters: new Map(chapters.map((c) => [c.id, c])),
    }
  })()

  // Kegagalan tidak boleh mengendap di cache dan membuat halaman rusak selama
  // semenit penuh; permintaan berikutnya harus boleh mencoba lagi.
  isi.catch(() => {
    if (direktori?.isi === isi) direktori = null
  })

  direktori = { pada: sekarang, isi }
  return isi
}

function attach(
  invoice: Invoice,
  members: Map<string, MemberWithChapter>,
  chapters: Map<string, Chapter>,
): InvoiceWithRelations {
  return {
    ...invoice,
    member: members.get(invoice.memberId) ?? null,
    chapter: chapters.get(invoice.chapterId) ?? null,
  }
}

/** Marks sent invoices past their due date as overdue, matching the UI's view. */
async function syncOverdue(invoices: Invoice[]): Promise<void> {
  const today = todayISO()
  const stale = invoices.filter((i) => i.status === 'sent' && i.dueDate < today)
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

/**
 * Batas per permintaan yang diterima server.
 *
 * Bukan angka pilihan klien: server menolak limit di atas 200, jadi menaikkannya
 * di sini hanya menghasilkan permintaan yang ditolak diam-diam.
 */
const MAKS_PER_HALAMAN = 200

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

    const { members, chapters } = await loadDirectory()
    return {
      rows: res.data.map((i) => attach(i, members, chapters)),
      total: res.meta.total,
      summary: res.meta.summary,
    }
  },

  async list(filters) {
    const res = await api.get<ListResponse<Invoice>>(
      `/invoices${query({ ...paramFilter(filters), limit: MAKS_PER_HALAMAN })}`,
    )
    await syncOverdue(res.data)

    const { members, chapters } = await loadDirectory()
    const rows = res.data.map((i) => attach(i, members, chapters))

    return rows
  },

  async getById(id) {
    let invoice: Invoice
    try {
      invoice = await api.get<Invoice>(`/invoices/${encodeURIComponent(id)}`)
    } catch (err) {
      if (isNotFound(err)) return null
      throw err
    }
    const { members, chapters } = await loadDirectory()
    return attach(invoice, members, chapters)
  },

  async listByMember(memberId) {
    const res = await api.get<ListResponse<Invoice>>(
      `/invoices${query({ memberId, limit: 200 })}`,
    )
    return res.data
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

  async getAuditLog(invoiceId) {
    const res = await api.get<ListResponse<AuditLogEntry>>(
      `/invoices/${encodeURIComponent(invoiceId)}/audit${query({ limit: 200 })}`,
    )
    return res.data
  },

  async renewalDue(withinDays = 30) {
    const [due, invoices] = await Promise.all([
      api.get<ListResponse<MemberWithChapter & { daysUntilDue: number }>>(
        `/members/renewal-due${query({ days: withinDays, limit: 200 })}`,
      ),
      api.get<ListResponse<Invoice>>(`/invoices${query({ limit: 200 })}`),
    ])

    // Newest invoice per member — the one whose period is ending.
    const latest = new Map<string, Invoice>()
    for (const invoice of invoices.data) {
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
