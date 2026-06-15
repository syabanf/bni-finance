import type { MemberRepository } from '@/services/types'
import type { MemberWithChapter } from '@/types'
import { delay, nowISO, store } from './store'

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
    // Members who have no registration invoice yet that is still active
    // (sent/paid). In the seed every member has one, so we surface those whose
    // registration invoice is still a draft — i.e. not yet issued.
    const hasIssuedRegistration = new Set(
      store.invoices
        .filter((i) => i.type === 'registration' && i.status !== 'draft' && i.status !== 'cancelled')
        .map((i) => i.memberId),
    )
    const result = store.members
      .filter((m) => !hasIssuedRegistration.has(m.id))
      .map((m) => withChapter(m.id)!)
    return delay(result.sort((a, b) => a.name.localeCompare(b.name)))
  },

  async sync() {
    const syncedAt = nowISO()
    store.members = store.members.map((m) => ({ ...m, syncedAt }))
    return delay({ count: store.members.length, syncedAt }, 900)
  },
}
