import type { AuthRepository } from '@/services/types'
import type { AuthUser, UserRole } from '@/types'
import { delay } from './store'

const STORAGE_KEY = 'bni-finance.auth'

/** Akun demo — email menentukan peran. Email lain tetap masuk sebagai Admin. */
export const DEMO_ACCOUNTS: Record<string, { id: string; name: string; role: UserRole }> = {
  'admin@bni-finance.com': { id: 'admin-national', name: 'Admin Nasional', role: 'admin' },
  'user@bni-finance.com': { id: 'user-demo', name: 'User BNI', role: 'user' },
}

const DEFAULT_ACCOUNT = DEMO_ACCOUNTS['admin@bni-finance.com']

/**
 * Demo auth. Accepts any non-empty credentials and persists the session in
 * localStorage. The backend API repository is the real implementation — the
 * `AuthRepository` contract stays the same.
 */
export const mockAuthRepository: AuthRepository = {
  async login(email, password) {
    await delay(null, 500)
    if (!email.trim() || !password.trim()) {
      throw new Error('Email dan password wajib diisi.')
    }
    const profile = DEMO_ACCOUNTS[email.trim().toLowerCase()] ?? DEFAULT_ACCOUNT
    const user: AuthUser = { id: profile.id, name: profile.name, email, role: profile.role }
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

  async updateProfile({ name }) {
    await delay(null, 400)
    const trimmed = name.trim()
    if (!trimmed) throw new Error('Nama tidak boleh kosong.')
    const raw = localStorage.getItem(STORAGE_KEY)
    const current: AuthUser = raw
      ? (JSON.parse(raw) as AuthUser)
      : { ...DEFAULT_ACCOUNT, email: 'admin@bni-finance.com' }
    const user: AuthUser = { ...current, name: trimmed }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(user))
    return user
  },

  async updatePassword(newPassword) {
    await delay(null, 400)
    if (newPassword.trim().length < 6) throw new Error('Kata sandi minimal 6 karakter.')
    // Demo mode — no real credential store to update.
  },
}
