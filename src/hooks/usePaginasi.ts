import { useMemo, useState } from 'react'

/**
 * Memotong daftar yang sudah ada di browser menjadi halaman.
 *
 * Untuk daftar yang seluruhnya sudah dimuat. Daftar yang halamannya diminta ke
 * server (invoice) menghitung offset-nya sendiri dan cukup memakai komponen
 * Pagination.
 */
export function usePaginasi<T>(items: readonly T[] | null | undefined, ukuran = 25) {
  const [halaman, setHalaman] = useState(1)
  const total = items?.length ?? 0

  // Jumlah baris berubah berarti saringannya berubah, jadi kembali ke awal.
  // Memuat ulang data yang jumlahnya sama tidak memindahkan halaman.
  const [totalLama, setTotalLama] = useState(total)
  if (total !== totalLama) {
    setTotalLama(total)
    setHalaman(1)
  }

  const potongan = useMemo(
    () => (items ?? []).slice((halaman - 1) * ukuran, halaman * ukuran),
    [items, halaman, ukuran],
  )

  return { potongan, paginasi: { page: halaman, pageSize: ukuran, total, onChange: setHalaman } }
}
