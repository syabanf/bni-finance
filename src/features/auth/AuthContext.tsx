import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import type { AuthUser } from '@/types'
import { authService } from '@/services'
import { clearSession, getToken, setUnauthorizedHandler } from '@/lib/apiClient'
import { fetchCurrentUser, PASSWORD_SEPARATOR, setCurrentUser } from '@/services/api/authRepository'
import { isMockMode } from '@/services/dataSource'

const useMock = isMockMode()

interface AuthContextValue {
  user: AuthUser | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  updateProfile: (name: string) => Promise<void>
  updatePassword: (currentPassword: string, newPassword: string) => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(() => authService.getCurrentUser())
  // In API mode a cached session still has to be confirmed with the server, so
  // start in a loading state only when there is a token worth checking.
  const [loading, setLoading] = useState(!useMock && getToken() !== null)

  useEffect(() => {
    if (useMock) return

    // A token the server rejects (expired, or signed with a rotated secret)
    // must drop the user out of the app rather than leave a dead session.
    setUnauthorizedHandler(() => {
      setCurrentUser(null)
      setUser(null)
    })

    if (getToken() === null) {
      setLoading(false)
      return () => setUnauthorizedHandler(null)
    }

    let cancelled = false
    fetchCurrentUser()
      .then((u) => {
        if (!cancelled) setUser(u)
      })
      .catch(() => {
        // Token no longer valid — start clean.
        clearSession()
        setCurrentUser(null)
        if (!cancelled) setUser(null)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
      setUnauthorizedHandler(null)
    }
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

  // The repository contract takes one string; the API also needs the current
  // password, so the pair travels joined and is split at the boundary.
  const updatePassword = useCallback(async (currentPassword: string, newPassword: string) => {
    await authService.updatePassword(
      useMock ? newPassword : `${currentPassword}${PASSWORD_SEPARATOR}${newPassword}`,
    )
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
