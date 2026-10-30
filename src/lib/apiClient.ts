/**
 * HTTP client for the Go backend — the single place the app talks to the
 * network.
 *
 * The session token lives in localStorage rather than memory so a page reload
 * doesn't sign the user out — the usual trade for that convenience.
 */

const BASE_URL = (import.meta.env.VITE_API_URL as string | undefined)?.replace(/\/+$/, '') ?? ''

const TOKEN_KEY = 'bni.auth.token'
const USER_KEY = 'bni.auth.user'

/** Where the backend lives. Empty string means same-origin. */
export function apiBaseUrl(): string {
  return BASE_URL
}

/** An error carrying the HTTP status, so callers can branch on 401 vs 409. */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message)
    this.name = 'ApiError'
  }

  /** The session is gone or was never valid — the app should sign out. */
  get isUnauthorized(): boolean {
    return this.status === 401
  }

  get isForbidden(): boolean {
    return this.status === 403
  }
}

// --- session ---------------------------------------------------------------

export function getToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    // Private browsing can make localStorage throw rather than return null.
    return null
  }
}

export function setSession(token: string, user: unknown): void {
  try {
    localStorage.setItem(TOKEN_KEY, token)
    localStorage.setItem(USER_KEY, JSON.stringify(user))
  } catch {
    // Storage unavailable — the session simply won't survive a reload.
  }
}

export function clearSession(): void {
  try {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  } catch {
    // nothing to clean up
  }
}

/** The cached profile, so the app can render before /auth/me returns. */
export function getCachedUser<T>(): T | null {
  try {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? (JSON.parse(raw) as T) : null
  } catch {
    return null
  }
}

/** Called when the server rejects our token, so AuthContext can react. */
type UnauthorizedHandler = () => void
let onUnauthorized: UnauthorizedHandler | null = null

export function setUnauthorizedHandler(fn: UnauthorizedHandler | null): void {
  onUnauthorized = fn
}

// --- requests ---------------------------------------------------------------

const PESAN_PUTUS = 'Tidak bisa menghubungi server. Periksa koneksi Anda.'

interface RequestOptions {
  method?: string
  body?: unknown
  /** Skip the Authorization header — for login and the public payment page. */
  anonymous?: boolean
  signal?: AbortSignal
  /** Multipart body; the browser sets its own Content-Type boundary. */
  form?: FormData
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, anonymous = false, signal, form } = options

  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  if (!anonymous) {
    const token = getToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  let response: Response
  try {
    response = await fetch(`${BASE_URL}/api/v1${path}`, {
      method,
      headers,
      body: form ?? (body === undefined ? undefined : JSON.stringify(body)),
      signal,
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(PESAN_PUTUS, 0)
  }

  if (response.status === 204) return undefined as T

  // Proxy yang menolak unggahan besar bisa memutus koneksi saat jawabannya
  // masih dibaca. Status gagalnya tetap cukup untuk menyusun pesan.
  const text = await response.text().catch(() => {
    if (response.ok) throw new ApiError(PESAN_PUTUS, 0)
    return ''
  })
  let payload: unknown = null
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = null
    }
  }

  if (!response.ok) {
    // 413 tanpa badan JSON datang dari proxy di depan server, yang menolak
    // berkas sebelum batas server sendiri sempat menjawab.
    const message =
      (payload as { error?: string } | null)?.error ??
      (response.status === 413 ? 'Berkas terlalu besar untuk diunggah.' : `Permintaan gagal (HTTP ${response.status})`)

    // A rejected token means the session is over; drop it here so every caller
    // doesn't have to remember to.
    if (response.status === 401 && !anonymous) {
      clearSession()
      onUnauthorized?.()
    }
    throw new ApiError(message, response.status)
  }

  return payload as T
}

export const api = {
  get: <T>(path: string, signal?: AbortSignal) => request<T>(path, { signal }),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body }),
  patch: <T>(path: string, body: unknown) => request<T>(path, { method: 'PATCH', body }),
  put: <T>(path: string, body: unknown) => request<T>(path, { method: 'PUT', body }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),

  /** Unauthenticated calls: login and the public payment page. */
  publicGet: <T>(path: string) => request<T>(path, { anonymous: true }),
  publicPost: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body, anonymous: true }),

  /** Unggah bukti pembayaran atau faktur; server menjawab `{ url }`. */
  upload: (path: string, file: File) => request<{ url: string }>(path, { method: 'POST', form: berkasForm(file) }),

  /**
   * Unggah multipart yang mengembalikan bentuk apa pun, untuk impor yang
   * menjawab laporan berisi hitungan dan daftar baris.
   */
  uploadFor: <T>(path: string, file: File) => request<T>(path, { method: 'POST', form: berkasForm(file) }),
}

function berkasForm(file: File): FormData {
  const form = new FormData()
  form.append('file', file)
  return form
}

/** Turns a stored path like `/uploads/x.jpg` into a URL the browser can load. */
export function fileUrl(path: string | undefined): string | undefined {
  if (!path) return undefined
  if (/^https?:\/\//.test(path)) return path
  return `${BASE_URL}${path}`
}

/** Shape every list endpoint returns. */
export interface ListResponse<T> {
  data: T[]
  meta: { total: number; limit: number; offset: number }
}

/** Builds a query string, skipping empty values. */
export function query(params: Record<string, string | number | undefined | null>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    search.set(key, String(value))
  }
  const s = search.toString()
  return s ? `?${s}` : ''
}
