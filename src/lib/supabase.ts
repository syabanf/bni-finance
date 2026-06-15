import { createClient, type SupabaseClient } from '@supabase/supabase-js'

let client: SupabaseClient | null = null

/**
 * Create (once) and return the Supabase client. Throws only if the env vars
 * are missing AND the client is actually needed — i.e. when running against
 * the real backend (VITE_USE_MOCK=false).
 */
export function getSupabaseClient(): SupabaseClient {
  if (client) return client
  const url = import.meta.env.VITE_SUPABASE_URL as string
  const key = import.meta.env.VITE_SUPABASE_ANON_KEY as string
  if (!url || !key) {
    throw new Error(
      'Supabase belum dikonfigurasi: set VITE_SUPABASE_URL & VITE_SUPABASE_ANON_KEY, ' +
        'atau jalankan dengan VITE_USE_MOCK=true (default) untuk memakai data mock.',
    )
  }
  client = createClient(url, key)
  return client
}

/**
 * Lazy proxy over the Supabase client.
 *
 * The real client is created on first property access, NOT at import time.
 * This matters because `services/index.ts` statically imports the Supabase
 * repositories even in mock mode, so importing this module must not require
 * Supabase env vars. In mock mode the repo methods are never called, so the
 * client is never instantiated and nothing throws.
 */
export const supabase: SupabaseClient = new Proxy({} as SupabaseClient, {
  get(_target, prop) {
    const c = getSupabaseClient()
    const value = (c as unknown as Record<string | symbol, unknown>)[prop]
    return typeof value === 'function'
      ? (value as (...args: unknown[]) => unknown).bind(c)
      : value
  },
})
