import type { RenewalRepository } from '@/services/types'
import type { RenewalAnswer, RenewalRequest, RenewalStage } from '@/types'
import { delay, nextId, nowISO, store } from './store'
import { mockUserRepository } from './userRepository'
// Kedua modul saling mengimpor. Aman karena tidak ada yang memanggil fungsi
// modul lain saat dimuat, hanya saat repository dipakai.
import { periodeTerdaftar } from './periodRepository'
import { buktiMenunggu } from './proofRepository'
import { mockAuthRepository } from './authRepository'

/**
 * Konfirmasi renewal versi contoh.
 *
 * Menegakkan aturan yang SAMA dengan backend, bukan versi longgarnya:
 *
 *   - satu baris per member per periode (menekan dua kali tidak menggandakan)
 *   - `pending` bukan jawaban yang sah, itu keadaan awal
 *
 * Mock yang lebih longgar membuat orang menyusun sesuatu di demo yang ditolak
 * sistem sebenarnya — dan itu baru terlihat setelah mereka memercayainya.
 */
const requests: RenewalRequest[] = []

/**
 * Invoice renewal untuk konfirmasi ini, dengan aturan yang sama dengan server:
 * invoice renewal member itu yang dibuat setelah konfirmasinya diminta dan
 * tidak dibatalkan, yang terbaru.
 */
function invoiceKonfirmasi(r: RenewalRequest) {
  return store.invoices
    .filter(
      (i) =>
        i.memberId === r.memberId &&
        i.type === 'renewal' &&
        i.status !== 'cancelled' &&
        i.status !== 'terminated' &&
        i.createdAt >= r.requestedAt,
    )
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt))[0]
}

/** Urutan tahapnya sama dengan CASE di renewal/repository.go. */
function tahap(r: RenewalRequest): RenewalStage {
  const inv = invoiceKonfirmasi(r)
  if (inv?.status === 'paid') return inv.renewalSubmittedAt ? 'submitted' : 'paid'
  if (inv && buktiMenunggu(inv.id)) return 'proof_pending'
  if (inv?.status === 'sent' || inv?.status === 'overdue') return 'invoiced'
  if (inv?.status === 'draft') return 'invoice_draft'
  if (r.answer === 'will_renew' && r.invoiceRequestedAt) return 'invoice_requested'
  return r.answer
}

function isiRelasi(r: RenewalRequest): RenewalRequest {
  const m = store.members.find((x) => x.id === r.memberId)
  const c = store.chapters.find((x) => x.id === (m?.chapterId ?? r.chapterId))
  const inv = invoiceKonfirmasi(r)
  return {
    ...r,
    memberName: m?.name ?? null,
    chapterName: c?.displayName ?? null,
    renewalDate: m?.renewalDate ?? null,
    stage: tahap(r),
    invoiceId: inv?.id ?? null,
    invoiceNumber: inv?.number ?? null,
    invoiceStatus: inv?.status ?? null,
  }
}

/** Permintaan konfirmasi TERAKHIR member ini; null bila belum pernah diminta. */
export function konfirmasiTerakhir(memberId: string): RenewalRequest | null {
  const milik = requests.filter((r) => r.memberId === memberId)
  if (milik.length === 0) return null
  return milik.reduce((a, b) => (b.requestedAt > a.requestedAt ? b : a))
}

/** Jumlah permintaan di satu periode. Penjaga hapus periode memakainya. */
export function jumlahPermintaanPeriode(period: string): number {
  return requests.filter((r) => r.period === period).length
}

/** Dipakai hapus member: permintaannya ikut terhapus, seperti di server. */
export function hapusPermintaanMember(memberId: string) {
  for (let i = requests.length - 1; i >= 0; i--) {
    if (requests[i].memberId === memberId) requests.splice(i, 1)
  }
}

export const mockRenewalRepository: RenewalRepository = {
  async list(params) {
    // Server mengisi nama MC lewat join ke tabel users; mock menirunya.
    const akun = new Map((await mockUserRepository.list()).map((u) => [u.id, u.name]))
    let out = requests
      .map(isiRelasi)
      .map((r) => ({ ...r, assignedMcName: r.assignedMc ? (akun.get(r.assignedMc) ?? null) : null }))
    if (params?.answer) out = out.filter((r) => r.answer === params.answer)
    if (params?.period) out = out.filter((r) => r.period === params.period)
    // Belum dijawab lebih dulu, lalu yang jatuh temponya terdekat — sama
    // seperti urutan di server.
    out.sort((a, b) => {
      if ((a.answer === 'pending') !== (b.answer === 'pending')) return a.answer === 'pending' ? -1 : 1
      return (a.renewalDate ?? '9999').localeCompare(b.renewalDate ?? '9999')
    })
    return delay(out, 220)
  },

  async request(memberIds, period, assignedMc) {
    // Server menolak periode yang tidak ada di master data.
    if (!periodeTerdaftar(period)) {
      throw new Error(`periode "${period}" belum terdaftar, tambahkan dulu di Kelola Periode`)
    }
    let dibuat = 0
    let dilewati = 0
    let visitor = 0
    for (const memberId of memberIds) {
      const m = store.members.find((x) => x.id === memberId)
      if (!m) throw new Error(`Member ${memberId} tidak ditemukan.`)
      // Visitor dilewati — meniru penjaga di backend.
      //
      // Mock yang lebih longgar daripada server adalah mock yang berbohong:
      // demo memperlihatkan permintaan renewal terkirim ke tamu, lalu produksi
      // menolaknya. Yang salah kemudian dikira produksinya.
      if (m.status === 'visitor') {
        visitor++
        continue
      }
      // Kunci (member, periode) — persis indeks unik di basis data.
      if (requests.some((r) => r.memberId === memberId && r.period === period)) {
        dilewati++
        continue
      }
      requests.push({
        id: nextId('rnw'),
        memberId,
        chapterId: m.chapterId,
        period,
        requestedBy: 'admin-national',
        requestedAt: nowISO(),
        assignedMc: assignedMc || null,
        answer: 'pending',
        answeredBy: null,
        answeredAt: null,
        note: null,
        memberName: null,
        chapterName: null,
        renewalDate: null,
        stage: 'pending',
        createdAt: nowISO(),
        updatedAt: nowISO(),
      })
      dibuat++
    }
    await delay(null, 300)
    return { dibuat, dilewati, visitor, total: dibuat + dilewati + visitor }
  },

  async cancel(id) {
    const i = requests.findIndex((x) => x.id === id)
    if (i < 0) throw new Error('Permintaan tidak ditemukan.')
    if (requests[i].answer !== 'pending') {
      throw new Error('permintaan yang sudah dijawab MC tidak bisa dibatalkan')
    }
    requests.splice(i, 1)
    return delay(undefined, 200)
  },

  async requestInvoice(ids) {
    let diminta = 0
    for (const id of new Set(ids)) {
      const r = requests.find((x) => x.id === id)
      if (!r || r.answer !== 'will_renew' || r.invoiceRequestedAt || invoiceKonfirmasi(r)) continue
      r.invoiceRequestedAt = nowISO()
      r.invoiceRequestedBy = mockAuthRepository.getCurrentUser()?.name ?? 'Pengguna Demo'
      r.updatedAt = r.invoiceRequestedAt
      diminta++
    }
    return delay({ diminta, dilewati: new Set(ids).size - diminta }, 250)
  },

  async cancelInvoiceRequest(id) {
    const r = requests.find((x) => x.id === id)
    if (!r) throw new Error('Permintaan tidak ditemukan.')
    if (!r.invoiceRequestedAt) throw new Error('invoice member ini belum diminta')
    if (invoiceKonfirmasi(r)) throw new Error('finance sudah membuat invoicenya; permintaan tidak bisa ditarik')
    r.invoiceRequestedAt = null
    r.invoiceRequestedBy = null
    r.updatedAt = nowISO()
    return delay(isiRelasi(r), 200)
  },

  async answer(id, answer: RenewalAnswer, note) {
    if (answer === 'pending') {
      throw new Error("Jawaban tidak boleh 'pending' — itu keadaan awal, bukan jawaban.")
    }
    const r = requests.find((x) => x.id === id)
    if (!r) throw new Error('Permintaan tidak ditemukan.')
    r.answer = answer
    r.note = note?.trim() || null
    r.answeredBy = 'mc-demo'
    r.answeredAt = nowISO()
    r.updatedAt = nowISO()
    return delay(isiRelasi(r), 250)
  },
}
