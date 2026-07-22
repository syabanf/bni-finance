import { api, type ListResponse } from '@/lib/apiClient'
import type { SettingsRepository } from '@/services/types'
import type { FeeSettings } from '@/types'

interface AppSetting {
  key: string
  value: string
  updatedAt?: string
  /** True when the API redacted a credential-shaped value. */
  masked?: boolean
}

/**
 * Reads one app_settings key. Returns null when it isn't set.
 *
 * Note: keys that look like credentials (token, secret, password, …) read back
 * as `••••••` — the API redacts them deliberately. Writes still work.
 */
export async function getAppSetting(key: string): Promise<string | null> {
  try {
    const setting = await api.get<AppSetting>(`/app-settings/${encodeURIComponent(key)}`)
    return setting.value
  } catch (err) {
    if ((err as { status?: number }).status === 404) return null
    throw err
  }
}

export async function setAppSetting(key: string, value: string): Promise<void> {
  await api.put<AppSetting>(`/app-settings/${encodeURIComponent(key)}`, { value })
}

export async function listAppSettings(): Promise<AppSetting[]> {
  const res = await api.get<ListResponse<AppSetting>>('/app-settings')
  return res.data
}

export const apiSettingsRepository: SettingsRepository = {
  getFees() {
    return api.get<FeeSettings>('/fee-settings')
  },

  updateFees(input) {
    return api.patch<FeeSettings>('/fee-settings', {
      registrationFee: input.registrationFee,
      renewalFee: input.renewalFee,
      notes: input.notes,
    })
  },
}
