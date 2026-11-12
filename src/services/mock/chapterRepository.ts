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

  async create(input) {
    if (store.chapters.some((c) => c.id === input.id)) throw new Error('Chapter dengan id tersebut sudah ada.')
    const chapter = { ...input, syncedAt: nowISO() }
    store.chapters.push(chapter)
    return delay(chapter, 250)
  },

  async update(id, input) {
    const chapter = store.chapters.find((c) => c.id === id)
    if (!chapter) throw new Error('Chapter tidak ditemukan.')
    Object.assign(chapter, input)
    return delay(chapter, 250)
  },

  async remove(id) {
    // Penjaga yang sama dengan server.
    const member = store.members.filter((m) => m.chapterId === id).length
    const invoice = store.invoices.filter((i) => i.chapterId === id).length
    if (member > 0 || invoice > 0) {
      throw new Error(`chapter masih dipakai oleh ${member} member dan ${invoice} invoice`)
    }
    store.chapters = store.chapters.filter((c) => c.id !== id)
    return delay(undefined, 250)
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
