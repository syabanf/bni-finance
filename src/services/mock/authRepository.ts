import type { AuthRepository } from '@/services/types'
import type { AuthUser } from '@/types'
import { delay } from './store'

const STORAGE_KEY = 'bni-finance.auth'

const DEMO_USER: AuthUser = {
  id: 'admin-national',
  name: 'Admin Nasional',
  email: 'admin@bni-finance.com',
  role: 'national_admin',
}

/**
 * Demo auth. Accepts any non-empty credentials and persists the session in
 * localStorage. Replace with Supabase Auth (`signInWithPassword`) later — the
 * `AuthRepository` contract stays the same.
 */
export const mockAuthRepository: AuthRepository = {
  async login(email, password) {
    await delay(null, 500)
    if (!email.trim() || !password.trim()) {
      throw new Error('Email dan password wajib diisi.')
    }
    const user: AuthUser = { ...DEMO_USER, email }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(user))
    return user
  },

  async logout() {
    await delay(null, 150)
    localStorage.removeItem(STORAGE_KEY)
  },

  getCurrentUser() {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    try {
      return JSON.parse(raw) as AuthUser
    } catch {
      return null
    }
  },
}
