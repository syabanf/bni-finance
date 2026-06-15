import type { ChapterRepository } from '@/services/types'
import { delay, nowISO, store } from './store'

export const mockChapterRepository: ChapterRepository = {
  async list() {
    return delay([...store.chapters].sort((a, b) => a.displayName.localeCompare(b.displayName)))
  },

  async getById(id) {
    return delay(store.chapters.find((c) => c.id === id) ?? null)
  },

  async sync() {
    const syncedAt = nowISO()
    store.chapters = store.chapters.map((c) => ({ ...c, syncedAt }))
    return delay({ count: store.chapters.length, syncedAt }, 700)
  },
}
