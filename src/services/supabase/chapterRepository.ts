import { supabase } from '@/lib/supabase'
import { getAppSetting } from './settingsRepository'
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
    const BNI_VM_URL = (await getAppSetting('bni_vm_url')) ?? import.meta.env.VITE_BNI_VM_URL as string
    const BNI_VM_TOKEN = (await getAppSetting('bni_vm_token')) ?? import.meta.env.VITE_BNI_VM_TOKEN as string
    if (!BNI_VM_TOKEN) throw new Error('Token BNI VM belum dikonfigurasi')

    // Fetch members to extract unique chapters (no dedicated /chapters endpoint)
    let allMembers: Record<string, unknown>[] = []
    let offset = 0
    const limit = 200
    while (true) {
      const res = await fetch(`${BNI_VM_URL}/members?limit=${limit}&offset=${offset}`, {
        headers: { Authorization: `Bearer ${BNI_VM_TOKEN}` },
      })
      if (!res.ok) throw new Error(`BNI VM API error: ${res.status}`)
      const json = await res.json() as { data: Record<string, unknown>[]; pagination: { hasMore: boolean } }
      allMembers = allMembers.concat(json.data)
      if (!json.pagination.hasMore) break
      offset += limit
    }

    // Extract unique chapters
    const chaptersMap: Record<string, { id: string; name: string }> = {}
    for (const m of allMembers) {
      const id = m.chapter_id as string
      if (!chaptersMap[id]) chaptersMap[id] = { id, name: m.chapter as string }
    }

    const now = new Date().toISOString()
    const rows = Object.values(chaptersMap).map(c => ({
      id: c.id,
      name: c.name,
      display_name: c.name,
      synced_at: now,
    }))

    const { error } = await supabase.from('chapters').upsert(rows, { onConflict: 'id' })
    if (error) throw new Error(error.message)
    return { count: rows.length, syncedAt: now }
  },
}
