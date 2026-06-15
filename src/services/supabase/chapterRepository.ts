import { supabase } from '@/lib/supabase'
import type { ChapterRepository } from '@/services/types'
import type { Chapter } from '@/types'

function rowToChapter(r: Record<string, unknown>): Chapter {
  return {
    id: r.id as string,
    name: r.name as string,
    displayName: r.display_name as string,
    areaName: r.area_name as string | undefined,
    cityName: r.city_name as string | undefined,
    syncedAt: r.synced_at as string,
  }
}

export const supabaseChapterRepository: ChapterRepository = {
  async list() {
    const { data, error } = await supabase.from('chapters').select('*').order('name')
    if (error) throw new Error(error.message)
    return (data ?? []).map(rowToChapter)
  },

  async getById(id) {
    const { data, error } = await supabase.from('chapters').select('*').eq('id', id).maybeSingle()
    if (error) throw new Error(error.message)
    return data ? rowToChapter(data) : null
  },

  async sync() {
    // No external BNI VM integration yet — return current count
    const { count, error } = await supabase.from('chapters').select('id', { count: 'exact', head: true })
    if (error) throw new Error(error.message)
    return { count: count ?? 0, syncedAt: new Date().toISOString() }
  },
}
