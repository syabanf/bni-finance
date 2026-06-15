import { supabase } from '@/lib/supabase'
import type { MemberRepository } from '@/services/types'
import type { Chapter, Member, MemberStatus, MemberWithChapter } from '@/types'

function rowToMember(r: Record<string, unknown>): Member {
  return {
    id: r.id as string,
    chapterId: r.chapter_id as string,
    name: r.name as string,
    email: r.email as string | undefined,
    phone: r.phone as string | undefined,
    company: r.company as string | undefined,
    businessField: r.business_field as string | undefined,
    status: r.status as MemberStatus,
    joinedDate: r.joined_date as string,
    syncedAt: r.synced_at as string,
  }
}

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

function withChapter(r: Record<string, unknown>): MemberWithChapter {
  const ch = r.chapters as Record<string, unknown> | null
  return {
    ...rowToMember(r),
    chapter: ch ? rowToChapter(ch) : null,
  }
}

export const supabaseMemberRepository: MemberRepository = {
  async list(params) {
    let q = supabase.from('members').select('*, chapters(*)').order('name')
    if (params?.chapterId) q = q.eq('chapter_id', params.chapterId)
    if (params?.search) q = q.ilike('name', `%${params.search}%`)
    const { data, error } = await q
    if (error) throw new Error(error.message)
    return (data ?? []).map(withChapter)
  },

  async getById(id) {
    const { data, error } = await supabase
      .from('members')
      .select('*, chapters(*)')
      .eq('id', id)
      .maybeSingle()
    if (error) throw new Error(error.message)
    return data ? withChapter(data) : null
  },

  async eligibleForRegistration() {
    // Members who have no registration invoice (or only draft ones)
    const { data: members, error } = await supabase.from('members').select('*, chapters(*)')
    if (error) throw new Error(error.message)

    const { data: regs } = await supabase
      .from('invoices')
      .select('member_id, status')
      .eq('type', 'registration')
      .in('status', ['sent', 'paid'])

    const hasSentOrPaid = new Set((regs ?? []).map((r: Record<string, unknown>) => r.member_id))
    return (members ?? [])
      .filter((m: Record<string, unknown>) => !hasSentOrPaid.has(m.id))
      .map(withChapter)
  },

  async sync() {
    const BNI_VM_URL = import.meta.env.VITE_BNI_VM_URL as string
    const BNI_VM_TOKEN = import.meta.env.VITE_BNI_VM_TOKEN as string

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

    const now = new Date().toISOString()

    // Upsert chapters first
    const chaptersMap: Record<string, string> = {}
    for (const m of allMembers) {
      const id = m.chapter_id as string
      if (!chaptersMap[id]) chaptersMap[id] = m.chapter as string
    }
    const chapterRows = Object.entries(chaptersMap).map(([id, name]) => ({
      id, name, display_name: name, synced_at: now,
    }))
    await supabase.from('chapters').upsert(chapterRows, { onConflict: 'id' })

    // Upsert members
    const memberRows = allMembers.map(m => ({
      id: m.id as string,
      chapter_id: m.chapter_id as string,
      name: m.name as string,
      email: m.email as string | null,
      phone: m.phone as string | null,
      company: m.company as string | null,
      business_field: m.business_field as string | null,
      status: (m.status as string) || 'active',
      joined_date: m.joined_date as string,
      synced_at: now,
    }))

    const { error } = await supabase.from('members').upsert(memberRows, { onConflict: 'id' })
    if (error) throw new Error(error.message)
    return { count: memberRows.length, syncedAt: now }
  },
}
