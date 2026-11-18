import { api, query, type ListResponse } from '@/lib/apiClient'

/**
 * Batas per permintaan yang diterima server.
 *
 * Bukan angka pilihan klien: server memotong limit di atas 200, jadi
 * menaikkannya di sini hanya menghasilkan halaman yang lebih pendek dari yang
 * dikira pemanggilnya.
 */
export const MAKS_PER_HALAMAN = 200

/**
 * Di atas angka ini pengambilan dibatalkan dengan galat.
 *
 * Daftar yang terpotong diam-diam adalah kesalahan yang paling mahal di
 * aplikasi ini: jumlah uang di laporan tetap tampil, hanya terlalu kecil.
 * Galat yang terlihat memaksa orang tahu bahwa halamannya perlu dipindah ke
 * agregasi di server.
 */
const BATAS_BARIS = 50_000

/** Halaman yang diminta bersamaan setelah halaman pertama memberi totalnya. */
const SEJAJAR = 4

/** Id per permintaan ?ids=. 50 UUID menjaga URL di bawah 2 KB. */
const ID_PER_PERMINTAAN = 50

type Param = Record<string, string | number | undefined | null>

/**
 * Menarik SELURUH baris sebuah daftar, bukan hanya halaman pertama.
 *
 * Versi sebelumnya meminta satu halaman berisi 200 baris dan memperlakukannya
 * sebagai keseluruhan. Akibatnya tidak pernah terlihat sebagai galat: daftar
 * berhenti di baris ke-200, dan jumlah yang dihitung darinya ikut terlalu
 * kecil.
 */
export async function tarikSemua<T>(jalur: string, params: Param = {}): Promise<T[]> {
  const halaman = (offset: number) =>
    api.get<ListResponse<T>>(`${jalur}${query({ ...params, limit: MAKS_PER_HALAMAN, offset })}`)

  const pertama = await halaman(0)
  const total = pertama.meta.total
  if (total > BATAS_BARIS) {
    throw new Error(
      `Daftar ini berisi ${total.toLocaleString('id-ID')} baris, di atas batas ${BATAS_BARIS.toLocaleString('id-ID')} yang bisa dimuat sekaligus. Persempit saringannya.`,
    )
  }

  const kumpulan = [...pertama.data]
  const sisa: number[] = []
  for (let offset = pertama.data.length; offset < total; offset += MAKS_PER_HALAMAN) sisa.push(offset)

  for (let i = 0; i < sisa.length; i += SEJAJAR) {
    const hasil = await Promise.all(sisa.slice(i, i + SEJAJAR).map(halaman))
    for (const h of hasil) kumpulan.push(...h.data)
  }
  return kumpulan
}

/**
 * Menarik baris yang kolom `kunci`-nya ada di `nilai`, misalnya
 * `/invoices?memberIds=...`. Nilai dipecah supaya URL-nya tetap pendek.
 */
export async function tarikUntuk<T>(
  jalur: string,
  kunci: string,
  nilai: Iterable<string>,
  params: Param = {},
): Promise<T[]> {
  const unik = [...new Set(nilai)].filter(Boolean)
  const potongan: string[][] = []
  for (let i = 0; i < unik.length; i += ID_PER_PERMINTAAN) potongan.push(unik.slice(i, i + ID_PER_PERMINTAAN))

  const kumpulan: T[] = []
  for (let i = 0; i < potongan.length; i += SEJAJAR) {
    const hasil = await Promise.all(
      potongan.slice(i, i + SEJAJAR).map((p) => tarikSemua<T>(jalur, { ...params, [kunci]: p.join(',') })),
    )
    for (const h of hasil) kumpulan.push(...h)
  }
  return kumpulan
}

/**
 * Mengambil baris menurut id-nya dan mengembalikannya sebagai peta.
 *
 * Dipakai untuk menempelkan member ke invoice dan invoice ke pembayaran.
 * Sebelumnya seluruh member dan invoice ditarik untuk keperluan itu, padahal
 * satu halaman hanya menyebut 25 di antaranya.
 *
 * Di atas `ambang` id, menarik seluruh daftar lebih murah daripada ratusan
 * permintaan ?ids=, jadi jalurnya dialihkan ke sana.
 */
export async function ambilPerId<T extends { id: string }>(
  jalur: string,
  ids: Iterable<string>,
  ambang = 400,
): Promise<Map<string, T>> {
  const unik = [...new Set(ids)].filter(Boolean)
  const baris = unik.length > ambang ? await tarikSemua<T>(jalur) : await tarikUntuk<T>(jalur, 'ids', unik)
  return new Map(baris.map((b) => [b.id, b]))
}
