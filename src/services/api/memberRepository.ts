import { api, query, type ListResponse } from '@/lib/apiClient'
import { bolehDitagih } from '@/lib/status'
import type { MemberRepository } from '@/services/types'
import type { Invoice, MemberWithChapter } from '@/types'
import { isNotFound } from './chapterRepository'
import { runSync } from './syncService'

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
    // "Eligible" = belum menjadi anggota (visitor, pending) dan belum punya
    // invoice pendaftaran. API tidak punya endpoint khusus, jadi dua kali baca
    // lalu selisih himpunan; murah pada ukuran data ini, dan backend tidak
    // perlu query yang pemanggilnya satu.
    const [members, invoices] = await Promise.all([
      api
        .get<ListResponse<MemberWithChapter>>(`/members${query({ limit: 200 })}`)
        .then((r) => ({ data: r.data.filter((m) => bolehDitagih(m.status, 'registration')) })),
      api.get<ListResponse<Invoice>>(`/invoices${query({ type: 'registration', limit: 200 })}`),
    ])

    const alreadyInvoiced = new Set(
      invoices.data.filter((i) => i.status !== 'cancelled').map((i) => i.memberId),
    )
    return members.data.filter((m) => !alreadyInvoiced.has(m.id))
  },

  async sync() {
    const result = await runSync()
    return { count: result.members, syncedAt: result.syncedAt }
  },
}
