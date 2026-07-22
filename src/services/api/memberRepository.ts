import { api, query, type ListResponse } from '@/lib/apiClient'
import type { MemberRepository } from '@/services/types'
import type { Invoice, MemberWithChapter } from '@/types'
import { isNotFound } from './chapterRepository'

// The API returns each member with its chapter already joined, so the shape
// matches MemberWithChapter without extra work.

export const apiMemberRepository: MemberRepository = {
  async list(params) {
    const res = await api.get<ListResponse<MemberWithChapter>>(
      `/members${query({ chapterId: params?.chapterId, q: params?.search, limit: 200 })}`,
    )
    return res.data
  },

  async getById(id) {
    try {
      return await api.get<MemberWithChapter>(`/members/${encodeURIComponent(id)}`)
    } catch (err) {
      if (isNotFound(err)) return null
      throw err
    }
  },

  async eligibleForRegistration() {
    // "Eligible" = active with no registration invoice yet. The API has no
    // dedicated endpoint, so it's two reads and a set difference — cheap at
    // this data size, and it keeps the backend free of a one-caller query.
    const [members, invoices] = await Promise.all([
      api.get<ListResponse<MemberWithChapter>>(`/members${query({ status: 'active', limit: 200 })}`),
      api.get<ListResponse<Invoice>>(`/invoices${query({ type: 'registration', limit: 200 })}`),
    ])

    const alreadyInvoiced = new Set(
      invoices.data.filter((i) => i.status !== 'cancelled').map((i) => i.memberId),
    )
    return members.data.filter((m) => !alreadyInvoiced.has(m.id))
  },

  async sync() {
    throw new Error(
      'Sinkronisasi BNI VM belum tersedia pada backend lokal. Kelola member lewat halaman Member.',
    )
  },
}
