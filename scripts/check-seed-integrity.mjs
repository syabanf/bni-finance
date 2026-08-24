#!/usr/bin/env node
/**
 * Menjaga data contoh mock tetap utuh dan tetap mencerminkan Postgres.
 *
 * TypeScript tidak bisa menangkap satu pun cacat di bawah ini: `chapterId`
 * hanyalah `string`, jadi member yang menunjuk chapter yang tidak ada tetap
 * lolos typecheck dan lolos build. Yang terlihat pengguna hanyalah kolom chapter
 * yang kosong di tabel member — tanpa galat, tanpa peringatan, tanpa petunjuk
 * apa pun tentang penyebabnya.
 *
 * Tiga hal yang diperiksa:
 *
 *   1. Tidak ada member yatim — setiap chapterId punya chapter yang benar-benar
 *      ada di chapterSeeds.
 *   2. Tidak ada chapter kosong — chapter tanpa member membuat laporan per
 *      chapter menampilkan baris nol yang terlihat seperti kerusakan data.
 *   3. Jumlah chapter dan member di mock sama dengan jumlah di db/init.sql
 *      beserta seluruh berkas di db/seeds/. Beralih antara Data Contoh dan
 *      Backend API tidak boleh mengubah dunia yang dilihat pengguna; kalau
 *      berbeda, demo menjanjikan sesuatu yang tidak akan mereka temukan di
 *      sistem sebenarnya.
 */

import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

const akar = new URL('..', import.meta.url).pathname
const merah = (s) => `\x1b[31m${s}\x1b[0m`
const hijau = (s) => `\x1b[32m${s}\x1b[0m`

// --- mock --------------------------------------------------------------------
const mock = readFileSync(join(akar, 'src/services/mock/seed.ts'), 'utf8')
const mockChapters = [...mock.matchAll(/\{ id: '(ch-[a-z]+)'/g)].map((m) => m[1])
const mockMembers = [...mock.matchAll(/\{ name: '([^']+)', chapterId: '(ch-[a-z]+)'/g)].map((m) => ({
  name: m[1],
  chapterId: m[2],
}))

// --- sql ---------------------------------------------------------------------
// Dibaca dari init.sql DAN seluruh berkas di db/seeds/, karena keduanya sama-sama
// menyumbang baris ke basis data yang sebenarnya.
const berkasSql = [
  join(akar, 'db/init.sql'),
  ...readdirSync(join(akar, 'db/seeds'))
    .filter((f) => f.endsWith('.sql'))
    .map((f) => join(akar, 'db/seeds', f)),
]
const sql = berkasSql.map((f) => readFileSync(f, 'utf8')).join('\n')
const sqlChapters = new Set([...sql.matchAll(/\('(ch-[a-z]+)',\s*'/g)].map((m) => m[1]))
const sqlMembers = new Set([...sql.matchAll(/\('(mem-\d+)',\s*'ch-/g)].map((m) => m[1]))

// --- pemeriksaan -------------------------------------------------------------
const masalah = []

const yatim = mockMembers.filter((m) => !mockChapters.includes(m.chapterId))
if (yatim.length) {
  masalah.push(
    `${yatim.length} member menunjuk chapter yang tidak ada:\n` +
      yatim.map((m) => `      ${m.name} -> ${m.chapterId}`).join('\n'),
  )
}

const kosong = mockChapters.filter((c) => !mockMembers.some((m) => m.chapterId === c))
if (kosong.length) {
  masalah.push(`${kosong.length} chapter tanpa member: ${kosong.join(', ')}`)
}

if (mockChapters.length !== sqlChapters.size) {
  masalah.push(
    `jumlah chapter berbeda — mock ${mockChapters.length}, SQL ${sqlChapters.size}\n` +
      `      hanya di mock: ${mockChapters.filter((c) => !sqlChapters.has(c)).join(', ') || '(tidak ada)'}\n` +
      `      hanya di SQL : ${[...sqlChapters].filter((c) => !mockChapters.includes(c)).join(', ') || '(tidak ada)'}`,
  )
}

if (mockMembers.length !== sqlMembers.size) {
  masalah.push(`jumlah member berbeda — mock ${mockMembers.length}, SQL ${sqlMembers.size}`)
}

// --- hasil -------------------------------------------------------------------
if (masalah.length) {
  console.error(merah('✗ data contoh tidak konsisten'))
  for (const m of masalah) console.error(`    - ${m}`)
  console.error(
    '\n  Mock dan SQL harus menggambarkan dunia yang sama. Perbarui\n' +
      '  src/services/mock/seed.ts, db/init.sql, atau db/seeds/*.sql sampai cocok.',
  )
  process.exit(1)
}

console.log(
  hijau('✓') +
    ` data contoh konsisten — ${mockChapters.length} chapter, ${mockMembers.length} member,` +
    ' mock dan SQL cocok',
)
