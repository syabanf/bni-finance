import { SUMBER_TERKUNCI } from '@/services/dataSource'

/**
 * Tarik data dari BNI Visitor Management.
 *
 * `false` menyembunyikan semuanya dari antarmuka: tombol "Tarik dari ... VM"
 * di halaman Member dan Chapter, kartu alamat dan token di halaman Integrasi,
 * saringan dan penyebutannya di Blackbox, langkah turnya, dan endpoint
 * /api/v1/sync di Konsol API.
 *
 * Yang disembunyikan hanya antarmukanya. Endpoint POST /api/v1/sync di backend
 * tetap ada dan tetap dijaga izin sync:run. Ubah ke `true` untuk menampilkan
 * semuanya lagi.
 */
export const TARIK_VM_TAMPIL = false

/**
 * Judul halaman /integrasi menurut isinya. Kosong berarti tidak ada kartu yang
 * tersisa, jadi halaman dan menunya ikut hilang.
 */
export const JUDUL_INTEGRASI = [TARIK_VM_TAMPIL && 'Integrasi', !SUMBER_TERKUNCI && 'Sumber Data']
  .filter(Boolean)
  .join(' & ')
