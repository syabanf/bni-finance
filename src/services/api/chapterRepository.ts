import { api, query, type ListResponse } from '@/lib/apiClient'
import type { ChapterRepository } from '@/services/types'
import type { Chapter } from '@/types'

// The API already speaks camelCase, so responses map straight onto the domain
// types — no row-mapping layer, unlike the Supabase client this replaces.

export const apiChapterRepository: ChapterRepository = {
  async list() {
    const res = await api.get<ListResponse<Chapter>>(`/chapters${query({ limit: 500 })}`)
    return res.data
  },

  async getById(id) {
    try {
      return await api.get<Chapter>(`/chapters/${encodeURIComponent(id)}`)
    } catch (err) {
      // The contract returns null for "not found" rather than throwing.
      if (isNotFound(err)) return null
      throw err
    }
  },

  async sync() {
    // BNI Visitor Management is a separate integration, not part of the local
    // stack. Fail loudly rather than pretending a sync happened.
    throw new Error(
      'Sinkronisasi BNI VM belum tersedia pada backend lokal. Kelola chapter lewat halaman Chapter.',
    )
  },
}

export function isNotFound(err: unknown): boolean {
  return typeof err === 'object' && err !== null && (err as { status?: number }).status === 404
}
