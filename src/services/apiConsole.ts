/**
 * Transport for the API console.
 *
 * Deliberately does NOT go through `lib/apiClient`: that client hides the parts
 * the console exists to show (it unwraps errors, swallows headers, and signs the
 * user out on 401). Here a 401 is a result to display, not an event to react to.
 *
 * In Data Contoh mode requests are answered in-browser by `mock/mockApi`; in
 * Backend API mode they are real HTTP calls carrying the current session token.
 */

import { apiBaseUrl, getToken } from '@/lib/apiClient'
import { isMockMode } from './dataSource'
import { mockApiFetch } from './mock/mockApi'

export interface ConsoleParam {
  name: string
  in: 'query' | 'path' | 'header' | 'cookie'
  required: boolean
  type: string
  enum?: string[]
  default?: unknown
  description?: string
}

/** One top-level property of a request body, rendered as a labelled input. */
export interface ConsoleBodyField {
  name: string
  /** Human label from the spec's `title`; falls back to a derived one. */
  label?: string
  type: string
  format?: string
  required: boolean
  enum?: string[]
  default?: unknown
  description?: string
  /** Object/array — edited as JSON rather than a single input. */
  complex: boolean
}

export interface ConsoleOperation {
  id: string
  method: string
  path: string
  tag: string
  summary: string
  description: string
  params: ConsoleParam[]
  body?: unknown
  bodyFields?: ConsoleBodyField[]
  bodyRequired: boolean
  multipart: boolean
  auth: boolean
  admin: boolean
}

export interface ApiCollection {
  version: string
  tags: { name: string; description: string }[]
  operations: ConsoleOperation[]
}

export interface ConsoleResult {
  status: number
  ok: boolean
  durationMs: number
  /** Parsed JSON when the response was JSON, otherwise the raw text. */
  body: unknown
  raw: string
  headers: Record<string, string>
  bytes: number
  mode: 'mock' | 'api'
  url: string
  /** Transport failure (server down, CORS). Distinct from an HTTP error. */
  transportError?: string
}

let cached: Promise<ApiCollection> | null = null

/**
 * Loads the generated collection. Kept out of the bundle and fetched on demand
 * so the 43 KB only costs the people who open the console.
 */
export function loadCollection(): Promise<ApiCollection> {
  cached ??= fetch(`${import.meta.env.BASE_URL}api-collection.json`).then((res) => {
    if (!res.ok) throw new Error(`Gagal memuat koleksi API (HTTP ${res.status})`)
    return res.json() as Promise<ApiCollection>
  })
  return cached
}

/** Substitutes `{id}`-style placeholders; leaves unfilled ones visible. */
export function buildPath(template: string, pathValues: Record<string, string>): string {
  return template.replace(/\{(\w+)\}/g, (match, name: string) => {
    const value = pathValues[name]?.trim()
    return value ? encodeURIComponent(value) : match
  })
}

export function buildQuery(values: Record<string, string>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(values)) {
    if (value.trim() === '') continue
    search.set(key, value.trim())
  }
  const s = search.toString()
  return s ? `?${s}` : ''
}

/** Methods that can destroy or alter data — the console warns before these. */
export const WRITE_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE'])

export interface SendInput {
  method: string
  /** Full spec path, placeholders already substituted. */
  path: string
  queryValues: Record<string, string>
  /** Raw JSON text from the editor; empty means no body. */
  body?: string
  /** For multipart endpoints (uploads) — sent instead of a JSON body. */
  file?: File
}

export async function sendConsoleRequest(input: SendInput): Promise<ConsoleResult> {
  const { method, path, queryValues, body, file } = input
  const search = buildQuery(queryValues)
  const query = new URLSearchParams(search)

  let parsedBody: unknown
  if (!file && body && body.trim()) {
    try {
      parsedBody = JSON.parse(body)
    } catch (err) {
      throw new Error(`Body bukan JSON valid: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  if (isMockMode()) {
    const started = performance.now()
    const res = await mockApiFetch(method, path, query, file ? { fileName: file.name } : parsedBody)
    const raw = res.body === null ? '' : JSON.stringify(res.body, null, 2)
    return {
      status: res.status,
      ok: res.status >= 200 && res.status < 300,
      durationMs: Math.round(performance.now() - started),
      body: res.body,
      raw,
      headers: { 'content-type': 'application/json', 'x-data-source': 'mock' },
      bytes: new Blob([raw]).size,
      mode: 'mock',
      url: `(mock)${path}${search}`,
    }
  }

  const url = `${apiBaseUrl()}${path}${search}`
  const headers: Record<string, string> = {}
  // Never set Content-Type for multipart: the browser has to add its own
  // boundary, and overriding it makes the server fail to parse the form.
  if (!file && parsedBody !== undefined) headers['Content-Type'] = 'application/json'
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`

  let payload: BodyInit | undefined
  if (file) {
    const form = new FormData()
    form.append('file', file)
    payload = form
  } else if (parsedBody !== undefined) {
    payload = JSON.stringify(parsedBody)
  }

  const started = performance.now()
  let response: Response
  try {
    response = await fetch(url, { method, headers, body: payload })
  } catch (err) {
    return {
      status: 0,
      ok: false,
      durationMs: Math.round(performance.now() - started),
      body: null,
      raw: '',
      headers: {},
      bytes: 0,
      mode: 'api',
      url,
      transportError:
        err instanceof Error ? err.message : 'Tidak bisa menghubungi server.',
    }
  }
  const raw = await response.text()
  const durationMs = Math.round(performance.now() - started)

  let parsed: unknown = raw
  try {
    parsed = raw ? JSON.parse(raw) : null
  } catch {
    // Not JSON (the /docs page, a YAML spec) — the raw text is shown instead.
  }

  const outHeaders: Record<string, string> = {}
  response.headers.forEach((value, key) => {
    outHeaders[key] = value
  })

  return {
    status: response.status,
    ok: response.ok,
    durationMs,
    body: parsed,
    raw,
    headers: outHeaders,
    bytes: new Blob([raw]).size,
    mode: 'api',
    url,
  }
}

/** `curl` equivalent of a request, for pasting into a terminal or a ticket. */
export function toCurl(input: SendInput, token: string | null): string {
  const url = `${apiBaseUrl() || 'http://localhost:8080'}${input.path}${buildQuery(input.queryValues)}`
  const lines = [`curl -X ${input.method} '${url}'`]
  if (token) lines.push(`  -H 'Authorization: Bearer ${token}'`)
  if (input.body?.trim()) {
    lines.push(`  -H 'Content-Type: application/json'`)
    lines.push(`  -d '${input.body.replace(/'/g, `'\\''`).replace(/\s+/g, ' ')}'`)
  }
  return lines.join(' \\\n')
}
