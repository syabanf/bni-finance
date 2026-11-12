import { getToken } from '@/lib/apiClient'

/**
 * Cache singkat untuk data referensi yang dibaca banyak halaman: daftar chapter
 * dan biaya keanggotaan. Tanpa ini setiap pindah halaman menarik ulang daftar
 * yang sama.
 *
 * Kuncinya token sesi, karena isi daftar chapter mengikuti lingkup chapter
 * pengguna: akun ST yang masuk sesudah admin tidak boleh menerima daftar
 * milik admin. Permintaan yang sedang berjalan dipakai bersama, dan kegagalan
 * tidak disimpan.
 */
export function cacheSingkat<T>(ambil: () => Promise<T>, umurMs: number) {
  let kunci: string | null = null
  let sampai = 0
  let janji: Promise<T> | null = null

  const get = (): Promise<T> => {
    const k = getToken() ?? ''
    if (janji && k === kunci && Date.now() < sampai) return janji
    kunci = k
    sampai = Date.now() + umurMs
    const ini: Promise<T> = ambil().catch((err: unknown) => {
      if (janji === ini) janji = null
      throw err
    })
    janji = ini
    return ini
  }
  const clear = () => {
    janji = null
  }
  return { get, clear }
}

/** Membungkus fungsi yang mengubah data supaya cache terkait dibuang sesudahnya. */
export function lalu<A extends unknown[], R>(fn: (...a: A) => Promise<R>, ...caches: { clear: () => void }[]) {
  return async (...a: A): Promise<R> => {
    try {
      return await fn(...a)
    } finally {
      caches.forEach((c) => c.clear())
    }
  }
}
