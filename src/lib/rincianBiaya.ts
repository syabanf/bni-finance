import type { BarisBiaya, FeeSettings, InvoiceType, RincianBiaya } from '@/types'
import { rupiahDari } from './kurs'

/**
 * Rincian harga per komponen, cermin dari backend/internal/domain/rincian.go.
 *
 *   Pendaftaran  Registration Fee + Membership Fee + VAT + Online Training
 *   Renewal      Membership Fee + VAT
 *
 * Dipakai halaman Pengaturan untuk memperlihatkan harga yang AKAN berlaku
 * sebelum Simpan ditekan, dan mode Data Contoh untuk meniru server. Yang
 * disimpan tetap hitungan server, jadi aturan pembulatannya harus sama persis:
 * VAT dibulatkan ke sen, tiap baris dibulatkan ke rupiah, dan total adalah
 * jumlah baris.
 */

const sen = (usd: number) => Math.round(usd * 100) / 100

/** Komponen Dollar sebelum pajak untuk satu tipe invoice. */
function dasarUsd(r: RincianBiaya, t: InvoiceType): number {
  return t === 'registration' ? r.pendaftaranUsd + r.keanggotaanUsd : r.keanggotaanUsd
}

/** Pajak dalam Dollar, dibulatkan ke sen. */
export function ppnUsd(r: RincianBiaya, t: InvoiceType): number {
  return sen((dasarUsd(r, t) * r.ppnPersen) / 100)
}

/** Bagian Dollar dari harga, termasuk pajak. Biaya training tidak ikut. */
export function totalUsd(r: RincianBiaya, t: InvoiceType): number {
  return sen(dasarUsd(r, t) + ppnUsd(r, t))
}

/** Komponen dalam Rupiah pada kurs yang diberikan. Komponen nol tidak menjadi baris. */
export function barisBiaya(r: RincianBiaya, t: InvoiceType, kurs: number): BarisBiaya[] {
  const semua: BarisBiaya[] = [
    ...(t === 'registration'
      ? [{ kode: 'registration', nama: 'Registration Fee', jumlah: rupiahDari(r.pendaftaranUsd, kurs) }]
      : []),
    { kode: 'membership', nama: 'Membership Fee', jumlah: rupiahDari(r.keanggotaanUsd, kurs) },
    { kode: 'vat', nama: `VAT (${r.ppnPersen}%)`, jumlah: rupiahDari(ppnUsd(r, t), kurs) },
    ...(t === 'registration'
      ? [{ kode: 'training', nama: 'Online Training Investment Bundling', jumlah: r.training }]
      : []),
  ]
  return semua.filter((b) => b.jumlah > 0)
}

export const jumlahBaris = (baris: BarisBiaya[]) => baris.reduce((a, b) => a + b.jumlah, 0)

type Dasar = Pick<FeeSettings, 'registrationFeeUsd' | 'renewalFeeUsd' | 'usdRate' | 'rincian'>

/**
 * Harga Dollar dan Rupiah per tipe, seperti yang diturunkan server saat
 * menyimpan. Dengan rincian aktif, komponenlah sumbernya dan harga Dollar per
 * tipe ikut diturunkan; tanpa rincian, Rupiah adalah Dollar × kurs.
 */
export function turunkanBiaya(f: Dasar) {
  if (!f.rincian.aktif) {
    return {
      registrationFeeUsd: f.registrationFeeUsd,
      renewalFeeUsd: f.renewalFeeUsd,
      registrationFee: rupiahDari(f.registrationFeeUsd, f.usdRate),
      renewalFee: rupiahDari(f.renewalFeeUsd, f.usdRate),
    }
  }
  return {
    registrationFeeUsd: totalUsd(f.rincian, 'registration'),
    renewalFeeUsd: totalUsd(f.rincian, 'renewal'),
    registrationFee: jumlahBaris(barisBiaya(f.rincian, 'registration', f.usdRate)),
    renewalFee: jumlahBaris(barisBiaya(f.rincian, 'renewal', f.usdRate)),
  }
}

/**
 * Rincian yang disimpan bersama invoice baru, atau undefined bila invoicenya
 * satu baris. Cermin rincianUntuk di backend: hanya bila rincian aktif DAN
 * nominalnya sama dengan harga yang berlaku untuk tipenya.
 */
export function rincianInvoice(f: Dasar, t: InvoiceType, amount: number): BarisBiaya[] | undefined {
  if (!f.rincian.aktif) return undefined
  const baris = barisBiaya(f.rincian, t, f.usdRate)
  return baris.length > 0 && jumlahBaris(baris) === amount ? baris : undefined
}
