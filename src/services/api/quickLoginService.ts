/**
 * Passwordless sign-in for demos and local development, in Backend API mode.
 *
 * The old path put the demo password in `VITE_DEMO_PASSWORD`, and Vite inlines
 * every `VITE_*` value into the public JS bundle — anyone who opened devtools
 * could read it. Here the credential never leaves the server: the browser only
 * ever sees a name, an email, and a role.
 *
 * The server decides whether this exists at all (`AUTH_QUICK_LOGIN`, an explicit
 * list of emails, empty by default). A 404 means "not enabled" and is the normal
 * answer in production — not an error worth showing.
 */

import { api, setSession, type ApiError } from '@/lib/apiClient'
import { setCurrentUser } from './authRepository'
import type { AuthUser } from '@/types'

export interface QuickLoginAccount {
  id: string
  name: string
  email: string
  role: AuthUser['role']
}

interface LoginResult {
  token: string
  expiresAt: string
  user: AuthUser
}

/** Lists the accounts the server allows. Empty when the feature is off. */
export async function listQuickLoginAccounts(): Promise<QuickLoginAccount[]> {
  try {
    const res = await api.publicGet<{ data: QuickLoginAccount[] }>('/auth/quick-login')
    return res.data ?? []
  } catch (err) {
    // 404 = disabled, 0 = backend unreachable. Both mean "no buttons", and
    // neither is worth an error banner on the sign-in page.
    const status = (err as ApiError)?.status
    if (status === 404 || status === 0) return []
    throw err
  }
}

/** Signs in as an allow-listed account and stores the session. */
export async function quickLogin(email: string): Promise<AuthUser> {
  const result = await api.publicPost<LoginResult>('/auth/quick-login', { email })
  setSession(result.token, result.user)
  // The repository caches the current user for getCurrentUser(); without this
  // the app would hold a valid token but believe nobody is signed in.
  setCurrentUser(result.user)
  return result.user
}
