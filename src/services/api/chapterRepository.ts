import { api, query, type ListResponse } from '@/lib/apiClient'
import type { ChapterRepository } from '@/services/types'
import type { Chapter } from '@/types'
import { runSync } from './syncService'

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
    // One server-side call refreshes chapters AND members — BNI VM has no
    // chapters endpoint, so chapters are derived from the member list.
    const result = await runSync()
    return { count: result.chapters, syncedAt: result.syncedAt }
  },
}

export function isNotFound(err: unknown): boolean {
  return typeof err === 'object' && err !== null && (err as { status?: number }).status === 404
}
