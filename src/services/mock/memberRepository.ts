import { hapusPermintaanMember } from './renewalRepository'
import type { MemberRepository } from '@/services/types'
import type { Member, MemberFilters, MemberPage, MemberStatus, MemberWithChapter } from '@/types'
import { bolehDitagih } from '@/lib/status'
import { delay, nextId, nowISO, store } from './store'

function withChapter(memberId: string): MemberWithChapter | null {
  const member = store.members.find((m) => m.id === memberId)
  if (!member) return null
  return {
    ...member,
    chapter: store.chapters.find((c) => c.id === member.chapterId) ?? null,
  }
}

const STATUS_VISITOR: MemberStatus[] = ['pending', 'visitor']

/**
 * Saringan yang SAMA dengan klausaMember di server.
 *
 * Dipakai bersama list() dan listPaged(). `denganStatus=false` meniru
 * ringkasan server, yang menghitung per status tanpa saringan statusnya.
 */
function saring(f: MemberFilters | undefined, denganStatus: boolean): MemberWithChapter[] {
  const q = f?.search?.trim().toLowerCase()
  const kosong = (v?: string | null) => !v?.trim()
  return store.members
    .filter((m) => {
      if (f?.chapterId && f.chapterId !== 'all' && m.chapterId !== f.chapterId) return false
      if (denganStatus && f?.status && f.status !== 'all' && m.status !== f.status) return false
      if (f?.kelompok && f.kelompok !== 'all' && STATUS_VISITOR.includes(m.status) !== (f.kelompok === 'visitor'))
        return false
      if (f?.kontak === 'no-phone' && !kosong(m.phone)) return false
      if (f?.kontak === 'no-email' && !kosong(m.email)) return false
      if (f?.kontak === 'no-contact' && !(kosong(m.phone) && kosong(m.email))) return false
      if (f?.punyaJatuhTempo && !m.renewalDate) return false
      if (f?.dueFrom && (!m.renewalDate || m.renewalDate < f.dueFrom)) return false
      if (f?.dueTo && (!m.renewalDate || m.renewalDate > f.dueTo)) return false
      if (
        q &&
        !m.name.toLowerCase().includes(q) &&
        !m.id.toLowerCase().includes(q) &&
        !(m.email ?? '').toLowerCase().includes(q) &&
        !(m.company ?? '').toLowerCase().includes(q)
      )
        return false
      return true
    })
    .map((m) => withChapter(m.id)!)
    .sort((a, b) => a.name.localeCompare(b.name))
}

export const mockMemberRepository: MemberRepository = {
  async list(params) {
    return delay(saring(params, true))
  },

  async listPaged(filters) {
    const semua = saring(filters, true)
    const offset = filters?.offset ?? 0
    const halaman: MemberPage = {
      rows: semua.slice(offset, offset + (filters?.limit ?? 25)),
      total: semua.length,
    }
    if (filters?.summary) {
      const byStatus: NonNullable<MemberPage['summary']>['byStatus'] = {}
      for (const m of saring(filters, false)) byStatus[m.status] = (byStatus[m.status] ?? 0) + 1
      halaman.summary = { byStatus }
    }
    return delay(halaman)
  },

  async getById(id) {
    return delay(withChapter(id))
  },

  async eligibleForRegistration() {
    // Visitor dan pending yang belum punya invoice pendaftaran terbit
    // (sent/paid). Member aktif tidak termasuk: mereka ditagih renewal.
    const hasIssuedRegistration = new Set(
      store.invoices
        .filter((i) => i.type === 'registration' && i.status !== 'draft' && i.status !== 'cancelled')
        .map((i) => i.memberId),
    )
    const result = store.members
      .filter((m) => bolehDitagih(m.status, 'registration') && !hasIssuedRegistration.has(m.id))
      .map((m) => withChapter(m.id)!)
    return delay(result.sort((a, b) => a.name.localeCompare(b.name)))
  },

  async create(input) {
    if (!store.chapters.some((c) => c.id === input.chapterId)) throw new Error('Chapter tidak ditemukan.')
    const member: Member = {
      id: nextId('mem'),
      chapterId: input.chapterId,
      name: input.name,
      email: input.email,
      phone: input.phone,
      company: input.company,
      businessField: input.businessField,
      status: input.status ?? 'active',
      joinedDate: null,
      renewalDate: null,
      syncedAt: nowISO(),
    }
    store.members.push(member)
    return delay(member, 300)
  },

  async update(id, input) {
    const member = store.members.find((m) => m.id === id)
    if (!member) throw new Error('Member tidak ditemukan.')
    Object.assign(member, input)
    return delay(member, 250)
  },

  async remove(id) {
    // Penjaga yang sama dengan server: member bertagihan diubah ke Inactive.
    if (!store.members.some((m) => m.id === id)) throw new Error('Member tidak ditemukan.')
    const n = store.invoices.filter((i) => i.memberId === id).length
    if (n > 0) throw new Error(`member masih punya ${n} invoice; ubah statusnya menjadi inactive`)
    hapusPermintaanMember(id)
    store.members = store.members.filter((m) => m.id !== id)
    return delay(undefined, 250)
  },

  async sync() {
    const syncedAt = nowISO()
    store.members = store.members.map((m) => ({ ...m, syncedAt }))
    return delay({ count: store.members.length, syncedAt }, 900)
  },
}
