#!/usr/bin/env node
/**
 * Memastikan setiap langkah tur menunjuk penanda `data-tour` yang benar-benar
 * ada di kode, dan sebaliknya.
 *
 * Tur berpandu gagal secara DIAM-DIAM: bila penandanya hilang karena komponen
 * dirapikan, turnya tetap berjalan dan hanya menyorot ruang kosong. Tidak ada
 * yang merah, tidak ada galat di konsol — pengguna hanya melihat panduan yang
 * menunjuk ke tempat yang salah.
 *
 * Dijalankan sebagai bagian dari `npm run typecheck`, jadi tidak perlu diingat.
 */
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const SRC = 'src'

function walk(dir) {
  return readdirSync(dir).flatMap((name) => {
    const full = join(dir, name)
    return statSync(full).isDirectory()
      ? walk(full)
      : /\.tsx?$/.test(name)
        ? [full]
        : []
  })
}

const files = walk(SRC)

// Penanda yang benar-benar dipasang di komponen.
const placed = new Map()
for (const f of files) {
  const src = readFileSync(f, 'utf8')
  for (const m of src.matchAll(/data-tour="([^"]+)"/g)) {
    // Lewati interpolasi template: `[data-tour="${anchor}"]` di TourOverlay
    // adalah query untuk MENCARI penanda, bukan penanda yang dipasang.
    if (m[1].includes('${')) continue
    if (!placed.has(m[1])) placed.set(m[1], f)
  }
}

// Penanda yang dirujuk oleh langkah tur.
const stepsFile = join(SRC, 'features/tour/tourSteps.ts')
const stepsSrc = readFileSync(stepsFile, 'utf8')
const referenced = new Map()
for (const m of stepsSrc.matchAll(/anchor:\s*'([^']+)'/g)) {
  referenced.set(m[1], (referenced.get(m[1]) ?? 0) + 1)
}

const problems = []

for (const anchor of referenced.keys()) {
  if (!placed.has(anchor)) {
    problems.push(
      `langkah tur menunjuk data-tour="${anchor}" — tidak ada elemen dengan penanda itu.\n` +
        `    turnya akan menyorot ruang kosong tanpa satu pun galat.`,
    )
  }
}

// Penanda yatim menandakan sisa refactor, atau langkah yang terhapus tanpa
// membersihkan penandanya. Bukan bug, tapi jejak yang menyesatkan pembaca.
for (const [anchor, file] of placed) {
  if (anchor === 'help-button') continue // pemicunya sendiri, bukan sasaran langkah
  if (!referenced.has(anchor)) {
    problems.push(`data-tour="${anchor}" di ${file} tidak dipakai langkah tur mana pun.`)
  }
}

if (problems.length > 0) {
  console.error(`\n✗ penanda tur tidak sinkron (${problems.length}):\n`)
  for (const p of problems) console.error(`  • ${p}`)
  console.error('')
  process.exit(1)
}

console.log(`✓ penanda tur sinkron — ${referenced.size} langkah beranchor, ${placed.size} penanda terpasang`)
