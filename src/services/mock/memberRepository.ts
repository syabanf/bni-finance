import type { MemberRepository } from '@/services/types'
import type { Member, MemberWithChapter } from '@/types'
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

export const mockMemberRepository: MemberRepository = {
  async list(params) {
    let result = store.members.map((m) => withChapter(m.id)!).filter(Boolean)

    if (params?.chapterId && params.chapterId !== 'all') {
      result = result.filter((m) => m.chapterId === params.chapterId)
    }
    if (params?.search) {
      const q = params.search.toLowerCase()
      result = result.filter(
        (m) =>
          m.name.toLowerCase().includes(q) ||
          m.id.toLowerCase().includes(q) ||
          (m.email ?? '').toLowerCase().includes(q),
      )
    }
    return delay(result.sort((a, b) => a.name.localeCompare(b.name)))
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

  async sync() {
    const syncedAt = nowISO()
    store.members = store.members.map((m) => ({ ...m, syncedAt }))
    return delay({ count: store.members.length, syncedAt }, 900)
  },
}
