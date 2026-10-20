/**
 * Sumber data aktif: mock in-memory atau REST API Go.
 *
 * Dulu ini ditentukan `VITE_USE_MOCK` saat build, jadi mengganti mode berarti
 * mengubah berkas .env lalu menjalankan ulang dev server — merepotkan saat demo.
 * Sekarang pilihannya tersimpan di browser dan bisa ditukar lewat tombol di
 * halaman Pengaturan; env hanya menjadi nilai awal.
 */

export type DataSource = 'mock' | 'api'

const STORAGE_KEY = 'bni.dataSource'

/**
 * Host yang punya backend di origin yang sama (reverse proxy meneruskan /api ke
 * Go). Di host ini Backend API menjadi nilai awal; di host lain, misalnya
 * pratinjau Vercel yang tidak punya /api, nilai awalnya tetap Data Contoh.
 */
const hostBackend = (import.meta.env.VITE_API_HOSTS ?? '')
  .split(',')
  .map((h: string) => h.trim().toLowerCase())
  .filter(Boolean)

function diHostBackend(): boolean {
  try {
    return hostBackend.includes(window.location.hostname.toLowerCase())
  } catch {
    return false
  }
}

/** Nilai awal, dipakai hanya saat pengguna belum pernah memilih. */
const envDefault: DataSource =
  import.meta.env.VITE_USE_MOCK === 'false' || diHostBackend() ? 'api' : 'mock'

function read(): DataSource {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'mock' || stored === 'api') return stored
  } catch {
    // Private browsing bisa membuat localStorage melempar.
  }
  return envDefault
}

/**
 * Dibaca SEKALI saat modul dimuat.
 *
 * Repository dipilih di titik komposisi pada waktu import, jadi nilainya harus
 * tetap sama sepanjang satu sesi halaman — kalau tidak, sebagian layar akan
 * memakai sumber lama dan sebagian sumber baru. Karena itu setDataSource()
 * memuat ulang halaman alih-alih menukar di tempat.
 */
const current: DataSource = read()

export function getDataSource(): DataSource {
  return current
}

export function isMockMode(): boolean {
  return current === 'mock'
}

/** Nilai tersimpan saat ini — bisa berbeda dari `current` bila baru diganti. */
export function getStoredDataSource(): DataSource {
  return read()
}

/**
 * Menyimpan pilihan lalu memuat ulang halaman.
 *
 * Reload disengaja: seluruh container layanan dirakit ulang, sehingga tidak ada
 * komponen yang masih memegang repository dari mode sebelumnya.
 */
export function setDataSource(next: DataSource): void {
  try {
    localStorage.setItem(STORAGE_KEY, next)
  } catch {
    // Tanpa penyimpanan, pilihan tidak bertahan — reload tetap dijalankan
    // supaya perilakunya tidak setengah-setengah.
  }
  window.location.reload()
}

export const DATA_SOURCE_LABEL: Record<DataSource, string> = {
  mock: 'Data Contoh',
  api: 'Backend API',
}
