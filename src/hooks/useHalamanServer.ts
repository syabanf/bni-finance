import { useEffect, useState } from 'react'
import { useAsync } from './useAsync'

/** Nilai yang baru berubah setelah `jeda` ms tanpa perubahan lagi. */
export function useTertunda<T>(nilai: T, jeda = 300): T {
  const [tertunda, setTertunda] = useState(nilai)
  useEffect(() => {
    const t = setTimeout(() => setTertunda(nilai), jeda)
    return () => clearTimeout(t)
  }, [nilai, jeda])
  return tertunda
}

/**
 * Daftar yang halamannya diminta ke server.
 *
 * `filter` HARUS stabil antar-render (useMemo): perubahan identitasnya dibaca
 * sebagai saringan baru dan mengembalikan halaman ke satu. Tanpa itu,
 * menyaring dari 300 baris ke 4 baris sementara halaman masih di 5
 * menampilkan tabel kosong, dan kosong terbaca sebagai "tidak ada datanya".
 *
 * Halaman terakhir yang berhasil dipertahankan selama permintaan berikutnya
 * berjalan, supaya tabel tidak berkedip kosong tiap pindah halaman.
 */
export function useHalamanServer<F extends object, P extends { total: number }>(
  muat: (f: F & { limit: number; offset: number }) => Promise<P>,
  filter: F,
  ukuran = 25,
) {
  const [page, setPage] = useState(1)
  const [filterLama, setFilterLama] = useState(filter)
  if (filter !== filterLama) {
    setFilterLama(filter)
    setPage(1)
  }

  const { data, loading, error, reload } = useAsync<P>(
    () => muat({ ...filter, limit: ukuran, offset: (page - 1) * ukuran }),
    [filter, page],
  )

  const [terakhir, setTerakhir] = useState<P | null>(null)
  useEffect(() => {
    if (data) setTerakhir(data)
  }, [data])
  const halaman = data ?? terakhir

  return {
    halaman,
    loading,
    error,
    reload,
    paginasi: { page, pageSize: ukuran, total: halaman?.total ?? 0, onChange: setPage },
  }
}
