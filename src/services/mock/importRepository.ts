import type { ImportRepository } from '@/services/types'
import type { ImportBaris, ImportHasil } from '@/types'
import { delay, store } from './store'

/**
 * Impor versi contoh.
 *
 * Membaca berkasnya SUNGGUHAN — CSV saja — lalu menerapkan aturan yang sama
 * dengan server: chapter yang tidak ada ditolak, id ganda ditolak, dan kolom
 * yang tidak dikirim tidak mengosongkan data.
 *
 * XLSX tidak dibaca di sini, dan itu dinyatakan terang-terangan lewat pesan
 * galat alih-alih menghasilkan laporan kosong yang terlihat seperti berkas yang
 * memang tidak berisi apa-apa. Menguraikan XLSX di peramban berarti menambah
 * pustaka hanya demi mode contoh.
 */

function bacaCSV(teks: string): string[][] {
  const bersih = teks.replace(/^﻿/, '')
  const baris = bersih.split(/\r?\n/).filter((b) => b.trim() !== '')
  // Pemisah ditebak dari BARIS JUDUL saja — sama seperti server. Titik koma di
  // dalam catatan tidak boleh membalik tebakannya.
  const pemisah = (baris[0].match(/;/g)?.length ?? 0) > (baris[0].match(/,/g)?.length ?? 0) ? ';' : ','
  return baris.map((b) => b.split(pemisah).map((sel) => sel.trim().replace(/^"|"$/g, '')))
}

const ALIAS: Record<string, string[]> = {
  id: ['id', 'member_id', 'memberid', 'chapter_id', 'chapterid', 'kode'],
  chapter: ['chapter_id', 'chapterid', 'chapter'],
  nama: ['name', 'nama'],
  email: ['email', 'surel'],
  phone: ['phone', 'telepon', 'hp'],
}

function normal(s: string) {
  return s.toLowerCase().replace(/[^a-z0-9]/g, '')
}

export const mockImportRepository: ImportRepository = {
  async preview(jenis, file) {
    return jalankan(jenis, file, false)
  },
  async apply(jenis, file) {
    return jalankan(jenis, file, true)
  },
}

async function jalankan(
  jenis: 'chapters' | 'members',
  file: File,
  terapkan: boolean,
): Promise<ImportHasil> {
  if (!file.name.toLowerCase().endsWith('.csv')) {
    throw new Error(
      'Mode Data Contoh hanya membaca CSV. Untuk menguji XLSX, beralih ke Backend API.',
    )
  }

  const rows = bacaCSV(await file.text())
  if (rows.length < 2) throw new Error('Berkas kosong atau hanya berisi judul kolom.')

  const judul = rows[0]
  const kolom = new Map<string, number>()
  judul.forEach((j, i) => {
    const n = normal(j)
    if (n && !kolom.has(n)) kolom.set(n, i)
  })

  const ambil = (baris: string[], nama: keyof typeof ALIAS) => {
    for (const a of ALIAS[nama]) {
      const i = kolom.get(normal(a))
      if (i !== undefined && baris[i]) return baris[i]
    }
    return ''
  }

  const hasil: ImportHasil = {
    format: 'csv',
    jenis,
    diterapkan: false,
    total: 0,
    baru: 0,
    diperbarui: 0,
    sama: 0,
    ditolak: 0,
    baris: [],
  }

  const dikenal = new Set(Object.values(ALIAS).flat().map(normal))
  const takDikenal = judul.filter((j) => normal(j) && !dikenal.has(normal(j)))
  if (takDikenal.length) {
    hasil.peringatan = [
      `kolom tidak dikenal dan diabaikan: ${takDikenal.join(', ')} — periksa ejaannya bila kolom itu seharusnya ikut terbaca`,
    ]
  }

  const terlihat = new Map<string, number>()

  for (let i = 1; i < rows.length; i++) {
    const nomor = i + 1
    const id = ambil(rows[i], 'id')
    const nama = ambil(rows[i], 'nama')
    if (!id && !nama) continue

    const b: ImportBaris = { nomor, id, nama, tindakan: 'baru' }
    const chapter = jenis === 'members' ? ambil(rows[i], 'chapter') : ''

    if (!id) {
      b.tindakan = 'ditolak'
      b.alasan = 'id kosong'
    } else if (!nama) {
      b.tindakan = 'ditolak'
      b.alasan = 'name kosong'
    } else if (terlihat.has(id)) {
      b.tindakan = 'ditolak'
      b.alasan = `id "${id}" sudah dipakai di baris ${terlihat.get(id)}`
    } else if (jenis === 'members' && !chapter) {
      b.tindakan = 'ditolak'
      b.alasan = 'chapter_id kosong'
    } else if (jenis === 'members' && !store.chapters.some((c) => c.id === chapter)) {
      // Kesalahan paling merusak yang bisa lolos: member berpindah ke chapter
      // yang salah, dan pendapatan chapter ikut salah hitung tanpa tanda apa pun.
      b.tindakan = 'ditolak'
      b.alasan = `chapter "${chapter}" tidak ada`
    }

    if (b.tindakan === 'ditolak') {
      hasil.ditolak++
      hasil.baris.push(b)
      continue
    }
    terlihat.set(id, nomor)

    const lama =
      jenis === 'members'
        ? store.members.find((m) => m.id === id)
        : store.chapters.find((c) => c.id === id)

    if (!lama) {
      b.tindakan = 'baru'
      hasil.baru++
    } else {
      const perubahan: string[] = []
      // Kolom KOSONG tidak dihitung sebagai perubahan — berkas yang hanya
      // memuat sebagian kolom adalah hal biasa, dan menganggap yang hilang
      // sebagai "kosongkan" akan menghapus data dalam satu impor yang wajar.
      if (nama && nama !== (lama as { name: string }).name) perubahan.push('name')
      if (jenis === 'members') {
        const m = lama as { chapterId: string; email: string | null; phone: string | null }
        const email = ambil(rows[i], 'email')
        const phone = ambil(rows[i], 'phone')
        if (chapter && chapter !== m.chapterId) perubahan.push('chapter_id')
        if (email && email !== m.email) perubahan.push('email')
        if (phone && phone !== m.phone) perubahan.push('phone')
      }
      if (perubahan.length === 0) {
        b.tindakan = 'sama'
        hasil.sama++
      } else {
        b.tindakan = 'diperbarui'
        b.perubahan = perubahan
        hasil.diperbarui++
      }
    }
    hasil.baris.push(b)
  }

  hasil.total = hasil.baris.length
  // Mode contoh tidak menulis ke store, dan itu dinyatakan lewat `diterapkan`
  // supaya layarnya tidak menjanjikan sesuatu yang tidak terjadi.
  hasil.diterapkan = terapkan
  return delay(hasil, 400)
}
