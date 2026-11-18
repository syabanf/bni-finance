import { api, query, type ListResponse } from '@/lib/apiClient'
import { STATUS_UNTUK_TIPE } from '@/lib/status'
import type { MemberRepository } from '@/services/types'
import type { Invoice, Member, MemberFilters, MemberPage, MemberWithChapter } from '@/types'
import { isNotFound } from './chapterRepository'
import { MAKS_PER_HALAMAN, tarikSemua, tarikUntuk } from './kumpulan'
import { runSync } from './syncService'

// The API returns each member with its chapter already joined, so the shape
// matches MemberWithChapter without extra work.

/** Menyusun query dari filter UI; "all" berarti tidak menyaring. */
function paramFilter(f: MemberFilters | undefined) {
  const nilai = (v?: string) => (v && v !== 'all' ? v : undefined)
  return {
    q: f?.search?.trim() || undefined,
    chapterId: nilai(f?.chapterId),
    status: nilai(f?.status),
    kelompok: nilai(f?.kelompok),
    kontak: nilai(f?.kontak),
    punyaJatuhTempo: f?.punyaJatuhTempo ? 'true' : undefined,
    renewalFrom: f?.dueFrom || undefined,
    renewalTo: f?.dueTo || undefined,
  }
}

export const apiMemberRepository: MemberRepository = {
  async list(params) {
    return tarikSemua<MemberWithChapter>('/members', paramFilter(params))
  },

  async listPaged(filters) {
    const res = await api.get<
      ListResponse<MemberWithChapter> & { meta: { summary?: MemberPage['summary'] } }
    >(
      `/members${query({
        ...paramFilter(filters),
        summary: filters?.summary ? 'true' : undefined,
        limit: Math.min(filters?.limit ?? 25, MAKS_PER_HALAMAN),
        offset: filters?.offset ?? 0,
      })}`,
    )
    return { rows: res.data, total: res.meta.total, summary: res.meta.summary }
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
    // "Eligible" = berstatus yang boleh ditagih pendaftaran dan belum punya
    // invoice pendaftaran. Calonnya diambil per status, lalu invoice
    // pendaftaran diminta hanya untuk mereka. Sebelumnya 200 member dan 200
    // invoice pertama dianggap seluruhnya, jadi calon ke-201 tidak pernah
    // muncul di halaman Perlu Tindakan.
    const perStatus = await Promise.all(
      STATUS_UNTUK_TIPE.registration.map((status) =>
        tarikSemua<MemberWithChapter>('/members', { status }),
      ),
    )
    const calon = perStatus.flat().sort((a, b) => a.name.localeCompare(b.name))
    const invoices = await tarikUntuk<Invoice>(
      '/invoices',
      'memberIds',
      calon.map((m) => m.id),
      { type: 'registration' },
    )

    const alreadyInvoiced = new Set(
      invoices.filter((i) => i.status !== 'cancelled').map((i) => i.memberId),
    )
    return calon.filter((m) => !alreadyInvoiced.has(m.id))
  },

  async create(input) {
    return api.post<Member>('/members', input)
  },

  async update(id, input) {
    return api.patch<Member>(`/members/${encodeURIComponent(id)}`, input)
  },

  async remove(id) {
    await api.delete(`/members/${encodeURIComponent(id)}`)
  },

  async sync() {
    const result = await runSync()
    return { count: result.members, syncedAt: result.syncedAt, needsReview: result.needsReview }
  },
}
