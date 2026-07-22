import { supabase } from '@/lib/supabase'
import type { AuthRepository } from '@/services/types'
import type { AuthUser, UserRole } from '@/types'

/** Peran diambil dari user_metadata.role. Default 'admin' agar akun lama
 *  (yang belum punya metadata) tidak kehilangan akses. Set role: 'user' di
 *  Supabase Auth untuk menurunkan hak akses. */
function roleOf(meta: Record<string, unknown> | undefined): UserRole {
  return meta?.role === 'user' ? 'user' : 'admin'
}

let currentUser: AuthUser | null = null

export const supabaseAuthRepository: AuthRepository = {
  async login(email, password) {
    const { data, error } = await supabase.auth.signInWithPassword({ email, password })
    if (error) throw new Error(error.message)
    const u = data.user
    currentUser = {
      id: u.id,
      name: u.user_metadata?.name ?? email.split('@')[0],
      email: u.email ?? email,
      role: roleOf(u.user_metadata),
    }
    return currentUser
  },

  async logout() {
    await supabase.auth.signOut()
    currentUser = null
  },

  getCurrentUser() {
    return currentUser
  },

  async updateProfile({ name }) {
    const trimmed = name.trim()
    if (!trimmed) throw new Error('Nama tidak boleh kosong.')
    const { data, error } = await supabase.auth.updateUser({ data: { name: trimmed } })
    if (error) throw new Error(error.message)
    const u = data.user
    currentUser = {
      id: u.id,
      name: u.user_metadata?.name ?? trimmed,
      email: u.email ?? currentUser?.email ?? '',
      role: roleOf(u.user_metadata),
    }
    return currentUser
  },

  async updatePassword(newPassword) {
    if (newPassword.trim().length < 6) throw new Error('Kata sandi minimal 6 karakter.')
    const { error } = await supabase.auth.updateUser({ password: newPassword })
    if (error) throw new Error(error.message)
  },
}
