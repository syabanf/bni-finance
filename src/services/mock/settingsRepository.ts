import type { SettingsRepository } from '@/services/types'
import { delay, nowISO, store } from './store'

export const mockSettingsRepository: SettingsRepository = {
  async getFees() {
    return delay({ ...store.feeSettings })
  },

  async updateFees(input) {
    store.feeSettings = {
      ...store.feeSettings,
      registrationFee: input.registrationFee,
      renewalFee: input.renewalFee,
      notes: input.notes,
      updatedBy: 'admin-national',
      updatedAt: nowISO(),
    }
    return delay({ ...store.feeSettings }, 500)
  },
}
