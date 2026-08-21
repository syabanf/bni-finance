/**
 * Perekam blackbox versi mock — cermin `internal/blackbox` di server.
 *
 * Sebelumnya halaman Blackbox mati total pada Data Contoh: "perekamnya berjalan
 * di server". Benar, tetapi akibatnya satu fitur hilang dari demo justru pada
 * mode yang dipakai untuk demo. Panggilan integrasi di mode mock memang tidak
 * pergi ke mana-mana, tapi bentuk rekamannya bisa dibuat identik — dan itu yang
 * perlu dilihat orang saat mengevaluasi aplikasinya.
 *
 * Ring buffer, terbaru dulu, persis seperti server.
 *
 * KEAMANAN: sama seperti server, yang masuk ke sini hanya body. Kredensial
 * hidup di header dan tidak pernah lewat jalur ini.
 */

export type BlackboxIntegration = 'paper_id' | 'bni_vm'
export type BlackboxDirection = 'outbound' | 'inbound'

export interface MockBlackboxCall {
  id: string
  time: string
  integration: BlackboxIntegration
  direction: BlackboxDirection
  method: string
  url: string
  status: number
  durationMs: number
  success: boolean
  request?: unknown
  response?: unknown
  error?: string
}

/** Sebesar ring buffer server (BLACKBOX_SIZE default 200). */
const MAX = 200

const PREFIX = 'mock.blackbox'

/**
 * Disimpan di localStorage, bukan di memori modul: halaman Blackbox dibuka
 * setelah navigasi, dan pada mode mock rekaman yang hilang tiap pindah halaman
 * membuat fiturnya tetap terasa mati.
 */
function read(): MockBlackboxCall[] {
  try {
    const raw = localStorage.getItem(PREFIX)
    return raw ? (JSON.parse(raw) as MockBlackboxCall[]) : []
  } catch {
    // Private browsing bisa melempar, bukan mengembalikan null.
    return []
  }
}

function write(calls: MockBlackboxCall[]): void {
  try {
    localStorage.setItem(PREFIX, JSON.stringify(calls))
  } catch {
    // Penyimpanan penuh atau tidak tersedia — rekaman sekadar tidak bertahan.
  }
}

let seq = 0

/** Mencatat satu panggilan. Terbaru diletakkan di depan, lalu dipotong. */
export function recordMockCall(call: Omit<MockBlackboxCall, 'id' | 'time'>): void {
  seq += 1
  const entry: MockBlackboxCall = {
    ...call,
    id: `mock-bb-${Date.now()}-${seq}`,
    time: new Date().toISOString(),
  }
  write([entry, ...read()].slice(0, MAX))
}

export interface MockBlackboxFilters {
  integration?: string
  direction?: string
  status?: string
}

export function listMockCalls(filters: MockBlackboxFilters = {}): MockBlackboxCall[] {
  const { integration, direction, status } = filters
  return read().filter((c) => {
    if (integration && integration !== 'all' && c.integration !== integration) return false
    if (direction && direction !== 'all' && c.direction !== direction) return false
    if (status === 'failed' && c.success) return false
    return true
  })
}

export function clearMockCalls(): void {
  write([])
}
