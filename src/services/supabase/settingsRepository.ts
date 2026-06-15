import { supabase } from '@/lib/supabase'
import type { SettingsRepository } from '@/services/types'
import type { FeeSettings } from '@/types'

// ---------------------------------------------------------------------------
// app_settings helpers — used by sync functions to read/write BNI VM token
// ---------------------------------------------------------------------------

export async function getAppSetting(key: string): Promise<string | null> {
  const { data } = await supabase.from('app_settings').select('value').eq('key', key).maybeSingle()
  return data ? (data as Record<string, unknown>).value as string : null
}

export async function setAppSetting(key: string, value: string): Promise<void> {
  await supabase.from('app_settings').upsert({ key, value, updated_at: new Date().toISOString() }, { onConflict: 'key' })
}

function rowToFees(r: Record<string, unknown>): FeeSettings {
  return {
    id: r.id as string,
    registrationFee: r.registration_fee as number,
    renewalFee: r.renewal_fee as number,
    currency: r.currency as string,
    notes: r.notes as string | undefined,
    updatedBy: r.updated_by as string | undefined,
    updatedAt: r.updated_at as string,
    createdAt: r.created_at as string,
  }
}

export const supabaseSettingsRepository: SettingsRepository = {
  async getFees() {
    const { data, error } = await supabase.from('fee_settings').select('*').eq('id', 'default').single()
    if (error) throw new Error(error.message)
    return rowToFees(data)
  },

  async updateFees(input) {
    const { data, error } = await supabase
      .from('fee_settings')
      .update({
        registration_fee: input.registrationFee,
        renewal_fee: input.renewalFee,
        notes: input.notes,
        updated_at: new Date().toISOString(),
      })
      .eq('id', 'default')
      .select('*')
      .single()
    if (error) throw new Error(error.message)
    return rowToFees(data)
  },
}
