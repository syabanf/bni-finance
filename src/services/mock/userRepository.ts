import type { UserRepository } from '@/services/types'
import type { ManagedUser, UserRole } from '@/types'
import { delay, nextId, nowISO } from './store'

/**
 * Akun contoh — satu per peran, supaya setiap tampilan halaman Pengguna punya
 * sesuatu untuk ditampilkan sejak awal.
 *
 * Mencerminkan aturan backend, bukan versi longgarnya: `st` dan `mc` selalu
 * punya chapter, `admin` dan `user` tidak pernah punya. Mock yang lebih longgar
 * dari server akan membuat orang menyusun sesuatu di demo yang ditolak sistem
 * sebenarnya.
 */
const SEKARANG = '2026-08-24T00:00:00Z'

const store: ManagedUser[] = [
  { id: 'usr-001', email: 'admin@bni-finance.com', name: 'Admin Nasional', role: 'admin', chapterId: null, createdAt: SEKARANG, updatedAt: SEKARANG },
  { id: 'usr-002', email: 'user@bni-finance.com', name: 'Pengamat', role: 'user', chapterId: null, createdAt: SEKARANG, updatedAt: SEKARANG },
  { id: 'usr-003', email: 'st.garuda@bni-finance.com', name: 'Sekretaris Garuda', role: 'st', chapterId: 'ch-garuda', createdAt: SEKARANG, updatedAt: SEKARANG },
  { id: 'usr-004', email: 'mc.garuda@bni-finance.com', name: 'Membership Garuda', role: 'mc', chapterId: 'ch-garuda', createdAt: SEKARANG, updatedAt: SEKARANG },
]

/** Aturan yang sama dengan backend; lihat domain.User.ValidateScope. */
function periksaLingkup(role: UserRole, chapterId: string | null) {
  const berlingkup = role === 'st' || role === 'mc'
  if (berlingkup && !chapterId) throw new Error(`Peran ${role} wajib punya chapter.`)
  if (!berlingkup && chapterId) throw new Error(`Peran ${role} tidak boleh punya chapter.`)
}

export const mockUserRepository: UserRepository = {
  async list() {
    return delay([...store], 200)
  },

  async create(input) {
    periksaLingkup(input.role, input.chapterId)
    if (store.some((u) => u.email.toLowerCase() === input.email.toLowerCase())) {
      throw new Error('Email tersebut sudah terdaftar.')
    }
    const user: ManagedUser = {
      id: nextId('usr'),
      email: input.email.toLowerCase(),
      name: input.name,
      role: input.role,
      chapterId: input.chapterId,
      createdAt: nowISO(),
      updatedAt: nowISO(),
    }
    store.push(user)
    return delay(user, 250)
  },

  async changeRole(id, role, chapterId) {
    periksaLingkup(role, chapterId)
    const user = store.find((u) => u.id === id)
    if (!user) throw new Error('Pengguna tidak ditemukan.')
    // Admin terakhir tidak boleh kehilangan perannya — sama seperti backend.
    // Tanpa penjaga ini, satu klik bisa membuat sistem tidak punya siapa pun
    // yang boleh mengubah pengaturan lagi.
    if (user.role === 'admin' && role !== 'admin' &&
        store.filter((u) => u.role === 'admin').length === 1) {
      throw new Error('Ini admin terakhir — sistem harus selalu punya satu admin.')
    }
    user.role = role
    user.chapterId = chapterId
    user.updatedAt = nowISO()
    return delay({ ...user }, 250)
  },

  async remove(id) {
    const i = store.findIndex((u) => u.id === id)
    if (i < 0) throw new Error('Pengguna tidak ditemukan.')
    if (store[i].role === 'admin' && store.filter((u) => u.role === 'admin').length === 1) {
      throw new Error('Ini admin terakhir — sistem harus selalu punya satu admin.')
    }
    store.splice(i, 1)
    await delay(null, 200)
  },
}
