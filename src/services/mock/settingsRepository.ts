import type { SettingsRepository } from '@/services/types'
import { delay, nowISO, store } from './store'
import { turunkanBiaya } from '@/lib/rincianBiaya'

export const mockSettingsRepository: SettingsRepository = {
  async getFees() {
    return delay({ ...store.feeSettings })
  },

  async updateFees(input) {
    // Meniru server: harga diturunkan dari USD × kurs, atau dari rincian bila
    // aktif, bukan diterima dari klien. Mock yang menerima Rupiah apa adanya
    // akan membuat demo terlihat benar sementara produksi menghitung angka lain.
    store.feeSettings = {
      ...store.feeSettings,
      usdRate: input.usdRate,
      rincian: input.rincian,
      ...turunkanBiaya(input),
      notes: input.notes,
      updatedBy: 'admin-national',
      updatedAt: nowISO(),
    }
    return delay({ ...store.feeSettings }, 500)
  },
}
