import type { SettingsRepository } from '@/services/types'
import { delay, nowISO, store } from './store'
import { rupiahDari } from '@/lib/kurs'

export const mockSettingsRepository: SettingsRepository = {
  async getFees() {
    return delay({ ...store.feeSettings })
  },

  async updateFees(input) {
    // Meniru server: Rupiah diturunkan dari USD × kurs, bukan diterima dari
    // klien. Mock yang menerima Rupiah apa adanya akan membuat demo terlihat
    // benar sementara produksi menghitung angka lain.
    store.feeSettings = {
      ...store.feeSettings,
      registrationFeeUsd: input.registrationFeeUsd,
      renewalFeeUsd: input.renewalFeeUsd,
      usdRate: input.usdRate,
      registrationFee: rupiahDari(input.registrationFeeUsd, input.usdRate),
      renewalFee: rupiahDari(input.renewalFeeUsd, input.usdRate),
      notes: input.notes,
      updatedBy: 'admin-national',
      updatedAt: nowISO(),
    }
    return delay({ ...store.feeSettings }, 500)
  },
}
