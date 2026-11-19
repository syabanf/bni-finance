import { useEffect, useState } from 'react'
import { getAppSetting } from '@/services/appSettings'

/**
 * Teks invoice yang diatur dari Pengaturan dan dikirim ke Paper.id.
 *
 * Satu daftar untuk tiga pemakai: kartu Pengaturan yang menyuntingnya,
 * pratinjau invoice di aplikasi, dan dokumen unduhannya. Sebelumnya pratinjau
 * menulis nama produk sendiri, jadi teks yang diubah admin tampil di invoice
 * Paper.id tetapi tidak di salinan yang diunduh pengurus.
 *
 * Kunci dan teks bawaan produk harus sama dengan `produk()` di
 * backend/internal/paperid/service.go.
 */
export const KUNCI_TEKS = {
  produkPendaftaran: 'paper_produk_pendaftaran',
  deskripsiPendaftaran: 'paper_deskripsi_pendaftaran',
  produkRenewal: 'paper_produk_renewal',
  deskripsiRenewal: 'paper_deskripsi_renewal',
  catatan: 'paper_catatan',
  syarat: 'paper_syarat',
} as const

export type TeksInvoice = Record<keyof typeof KUNCI_TEKS, string>

/**
 * Yang dipakai bila pengaturannya kosong. Catatan dan syarat tidak punya
 * bawaan: kosong berarti bagian itu tidak dikirim ke Paper.id.
 */
export const TEKS_BAWAAN: TeksInvoice = {
  produkPendaftaran: 'Biaya Pendaftaran Member Business Network International',
  deskripsiPendaftaran: 'Pendaftaran anggota baru (berlaku 1 tahun)',
  produkRenewal: 'Perpanjangan Keanggotaan Business Network International',
  deskripsiRenewal: 'Perpanjangan keanggotaan tahunan',
  catatan: '',
  syarat: '',
}

const NAMA = Object.keys(KUNCI_TEKS) as (keyof TeksInvoice)[]

let tersimpan: Promise<TeksInvoice> | null = null

/**
 * Membaca keenam teks, dengan bawaan untuk yang kosong.
 *
 * Hasilnya disimpan selama halaman terbuka; `lupakanTeksInvoice` membuangnya
 * setelah admin menyimpan perubahan. Pengaturan yang gagal dibaca jatuh ke
 * bawaan: pratinjau dengan teks bawaan lebih baik daripada pratinjau yang
 * tidak tampil.
 */
export function muatTeksInvoice(): Promise<TeksInvoice> {
  tersimpan ??= Promise.all(NAMA.map((n) => getAppSetting(KUNCI_TEKS[n]).catch(() => null))).then((nilai) => {
    const teks = { ...TEKS_BAWAAN }
    NAMA.forEach((n, i) => {
      const v = nilai[i]?.trim()
      if (v) teks[n] = v
    })
    return teks
  })
  return tersimpan
}

export function lupakanTeksInvoice(): void {
  tersimpan = null
}

/** Teks invoice untuk komponen. Mulai dari bawaan, lalu berganti saat pengaturan tiba. */
export function useTeksInvoice(): TeksInvoice {
  const [teks, setTeks] = useState(TEKS_BAWAAN)
  useEffect(() => {
    let aktif = true
    void muatTeksInvoice().then((t) => {
      if (aktif) setTeks(t)
    })
    return () => {
      aktif = false
    }
  }, [])
  return teks
}
