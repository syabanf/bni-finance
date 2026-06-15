import { supabase } from '@/lib/supabase'
import type { SettingsRepository } from '@/services/types'
import type { FeeSettings } from '@/types'

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
