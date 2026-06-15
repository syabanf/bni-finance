import { supabase } from '@/lib/supabase'
import type { AuthRepository } from '@/services/types'
import type { AuthUser } from '@/types'

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
      role: 'national_admin',
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
}
