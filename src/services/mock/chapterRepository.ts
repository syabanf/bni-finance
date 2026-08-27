import type { ChapterRepository } from '@/services/types'
import { delay, nowISO, store } from './store'

export const mockChapterRepository: ChapterRepository = {
  async list() {
    return delay([...store.chapters].sort((a, b) => a.displayName.localeCompare(b.displayName)))
  },

  // Meniru server: menghitung dari SELURUH data mock, lalu dibatasi ke chapter
  // pengguna bila memang berlingkup.
  async counts() {
    return delay(
      store.chapters.map((c) => ({
        chapterId: c.id,
        memberCount: store.members.filter((m) => m.chapterId === c.id).length,
        outstanding: store.invoices
          .filter((i) => i.chapterId === c.id && (i.status === 'sent' || i.status === 'overdue'))
          .reduce((a, i) => a + i.amount, 0),
      })),
    )
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
