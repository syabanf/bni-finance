import { useState, useSyncExternalStore } from 'react'

export type Tampilan = 'kartu' | 'tabel'

const LAYAR_LEBAR = '(min-width: 1024px)'

function pantau(panggil: () => void) {
  const mq = window.matchMedia(LAYAR_LEBAR)
  mq.addEventListener('change', panggil)
  return () => mq.removeEventListener('change', panggil)
}

function baca(kunci: string): Tampilan | null {
  try {
    const v = localStorage.getItem(kunci)
    return v === 'kartu' || v === 'tabel' ? v : null
  } catch {
    // Private browsing bisa membuat localStorage melempar.
    return null
  }
}

/**
 * Tampilan sebuah daftar: kartu atau tabel.
 *
 * Selama pengguna belum memilih, tampilannya mengikuti lebar layar: kartu di
 * HP dan tablet, tabel dari 1024px ke atas. Itu perilaku sebelum pemilih ini
 * ada. `bawaan` menggantinya untuk halaman yang memang berangkat dari satu
 * bentuk di semua lebar layar.
 *
 * Pilihan disimpan per daftar di browser.
 */
export function useTampilan(nama: string, bawaan?: Tampilan) {
  const kunci = `bni.tampilan.${nama}`
  const [pilihan, setPilihan] = useState<Tampilan | null>(() => baca(kunci))
  const layarLebar = useSyncExternalStore(pantau, () => window.matchMedia(LAYAR_LEBAR).matches)

  const tampilan: Tampilan = pilihan ?? bawaan ?? (layarLebar ? 'tabel' : 'kartu')

  const ganti = (baru: Tampilan) => {
    setPilihan(baru)
    try {
      localStorage.setItem(kunci, baru)
    } catch {
      // Tanpa penyimpanan, pilihannya berlaku sampai halaman dimuat ulang.
    }
  }

  return [tampilan, ganti] as const
}
