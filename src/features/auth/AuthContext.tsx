import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import type { AuthUser } from '@/types'
import { authService } from '@/services'

const useMock = import.meta.env.VITE_USE_MOCK !== 'false'

interface AuthContextValue {
  user: AuthUser | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  updateProfile: (name: string) => Promise<void>
  updatePassword: (newPassword: string) => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(() => authService.getCurrentUser())
  const [loading, setLoading] = useState(!useMock)

  useEffect(() => {
    if (useMock) return
    // Restore Supabase session on mount
    import('@/lib/supabase').then(({ supabase }) => {
      supabase.auth.getSession().then(({ data }) => {
        const u = data.session?.user
        if (u) {
          setUser({
            id: u.id,
            name: u.user_metadata?.name ?? u.email?.split('@')[0] ?? 'Admin',
            email: u.email ?? '',
            role: 'national_admin',
          })
        }
        setLoading(false)
      })

      const { data: listener } = supabase.auth.onAuthStateChange((_event, session) => {
        const u = session?.user
        if (u) {
          setUser({
            id: u.id,
            name: u.user_metadata?.name ?? u.email?.split('@')[0] ?? 'Admin',
            email: u.email ?? '',
            role: 'national_admin',
          })
        } else {
          setUser(null)
        }
      })
      return () => listener.subscription.unsubscribe()
    })
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const u = await authService.login(email, password)
    setUser(u)
  }, [])

  const logout = useCallback(async () => {
    await authService.logout()
    setUser(null)
  }, [])

  const updateProfile = useCallback(async (name: string) => {
    const u = await authService.updateProfile({ name })
    setUser(u)
  }, [])

  const updatePassword = useCallback(async (newPassword: string) => {
    await authService.updatePassword(newPassword)
  }, [])

  const value = useMemo(
    () => ({ user, loading, login, logout, updateProfile, updatePassword }),
    [user, loading, login, logout, updateProfile, updatePassword],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
