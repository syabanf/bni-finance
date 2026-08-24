import { api, query, type ListResponse } from '@/lib/apiClient'
import type { UserRepository } from '@/services/types'
import type { ManagedUser } from '@/types'

/**
 * Pengelolaan akun lewat backend Go.
 *
 * Seluruh endpoint di sini ADMIN SAJA — batasnya ditegakkan server, dan halaman
 * yang memanggilnya hanya menyembunyikan tombol.
 */
export const apiUserRepository: UserRepository = {
  async list() {
    const res = await api.get<ListResponse<ManagedUser>>(`/users${query({ limit: 200 })}`)
    return res.data
  },

  async create(input) {
    return api.post<ManagedUser>('/users', {
      email: input.email,
      name: input.name,
      password: input.password,
      role: input.role,
      // null, bukan string kosong: backend membedakan "tidak berlingkup" dari
      // "berlingkup ke chapter bernama kosong", dan yang kedua ditolak.
      chapterId: input.chapterId || null,
    })
  },

  async changeRole(id, role, chapterId) {
    return api.patch<ManagedUser>(`/users/${encodeURIComponent(id)}/role`, {
      role,
      chapterId: chapterId || null,
    })
  },

  async remove(id) {
    await api.delete(`/users/${encodeURIComponent(id)}`)
  },
}
