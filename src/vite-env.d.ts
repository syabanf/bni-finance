/// <reference types="vite/client" />

/**
 * Hanya variabel yang benar-benar dibaca kode. Ingat: Vite menanam setiap nilai
 * `VITE_*` ke dalam bundel JS publik, jadi tidak ada rahasia yang boleh masuk
 * ke sini. Kredensial (Paper.id, BNI VM, JWT, database) hidup di backend.
 */
interface ImportMetaEnv {
  /** Alamat backend Go. Kosong berarti origin yang sama. */
  readonly VITE_API_URL: string
  /**
   * Sumber data awal saat localStorage belum diisi. Setelah itu tombol di
   * halaman Pengaturan / Login yang menentukan — lihat services/dataSource.ts.
   */
  readonly VITE_USE_MOCK: string
  /** Host berbackend di origin yang sama, dipisah koma; lihat services/dataSource. */
  readonly VITE_API_HOSTS?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
