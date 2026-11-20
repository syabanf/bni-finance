import type { PeriodRepository } from '@/services/types'
import type { RenewalPeriod, SimpanPeriodeInput } from '@/types'
import { kodePeriode, namaDariRentang, rentangBerikutnya } from '@/lib/periode'
import { delay } from './store'
import { jumlahPermintaanPeriode } from './renewalRepository'

/**
 * Periode versi contoh, dengan aturan yang sama dengan server: id dari rentang
 * tanggal, nama tidak boleh ganda, rentang tidak boleh beririsan, dan periode
 * yang sudah dipakai permintaan tidak bisa dihapus.
 */
type Baris = Omit<RenewalPeriod, 'requestCount'>

/** Enam semester mulai semester berjalan, sama dengan isi awal di init.sql. */
function isiAwal(): Baris[] {
  const out: Baris[] = []
  for (let i = 0; i < 6; i++) {
    const { mulai, selesai } = rentangBerikutnya(out.map((p) => ({ ...p, requestCount: 0 })))
    out.push({ id: kodePeriode(mulai, selesai)[0], name: namaDariRentang(mulai, selesai), startDate: mulai, endDate: selesai })
  }
  return out
}

const periode: Baris[] = isiAwal()

export function periodeTerdaftar(id: string): boolean {
  return periode.some((p) => p.id === id)
}

const urut = () => periode.sort((a, b) => a.startDate.localeCompare(b.startDate) || a.id.localeCompare(b.id))

const lengkapi = (p: Baris): RenewalPeriod => ({ ...p, requestCount: jumlahPermintaanPeriode(p.id) })

function periksa(p: Baris, cekRentang: boolean) {
  if (!p.name) throw new Error('name wajib diisi')
  if (!p.startDate || !p.endDate) throw new Error('startDate dan endDate wajib diisi')
  if (p.endDate < p.startDate) throw new Error('endDate tidak boleh sebelum startDate')
  for (const lain of periode) {
    if (lain.id === p.id) continue
    if (lain.name.toLowerCase() === p.name.toLowerCase()) {
      throw new Error(`nama "${lain.name}" sudah dipakai periode lain`)
    }
    if (cekRentang && lain.startDate <= p.endDate && lain.endDate >= p.startDate) {
      throw new Error(`rentang ini beririsan dengan periode "${lain.name}" (${lain.startDate} sampai ${lain.endDate})`)
    }
  }
}

export const mockPeriodRepository: PeriodRepository = {
  async list() {
    return delay(urut().map(lengkapi), 150)
  },

  async create(input: SimpanPeriodeInput) {
    const baru = { id: '', name: input.name.trim(), startDate: input.startDate, endDate: input.endDate }
    periksa(baru, true)
    const id = kodePeriode(baru.startDate, baru.endDate).find((k) => !periodeTerdaftar(k))
    if (!id) throw new Error('periode dengan rentang tanggal itu sudah ada')
    periode.push({ ...baru, id })
    urut()
    return delay(lengkapi({ ...baru, id }), 200)
  },

  async update(id, input) {
    const lama = periode.find((p) => p.id === id)
    if (!lama) throw new Error('Periode tidak ditemukan.')
    const baru = { ...lama, ...input, name: (input.name ?? lama.name).trim() }
    periksa(baru, baru.startDate !== lama.startDate || baru.endDate !== lama.endDate)
    Object.assign(lama, baru)
    urut()
    return delay(lengkapi(lama), 200)
  },

  async remove(id) {
    const i = periode.findIndex((p) => p.id === id)
    if (i < 0) throw new Error('Periode tidak ditemukan.')
    const n = jumlahPermintaanPeriode(id)
    if (n > 0) throw new Error(`periode ini dipakai ${n} permintaan konfirmasi, jadi tidak bisa dihapus`)
    periode.splice(i, 1)
    return delay(undefined, 200)
  },
}
