import type { RenewalRepository } from '@/services/types'
import type { RenewalAnswer, RenewalRequest } from '@/types'
import { delay, nextId, nowISO, store } from './store'

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

function isiRelasi(r: RenewalRequest): RenewalRequest {
  const m = store.members.find((x) => x.id === r.memberId)
  const c = store.chapters.find((x) => x.id === (m?.chapterId ?? r.chapterId))
  return {
    ...r,
    memberName: m?.name ?? null,
    chapterName: c?.displayName ?? null,
    renewalDate: m?.renewalDate ?? null,
  }
}

export const mockRenewalRepository: RenewalRepository = {
  async list(params) {
    let out = requests.map(isiRelasi)
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
    let dibuat = 0
    let dilewati = 0
    for (const memberId of memberIds) {
      const m = store.members.find((x) => x.id === memberId)
      if (!m) throw new Error(`Member ${memberId} tidak ditemukan.`)
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
        createdAt: nowISO(),
        updatedAt: nowISO(),
      })
      dibuat++
    }
    await delay(null, 300)
    return { dibuat, dilewati, total: dibuat + dilewati }
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
