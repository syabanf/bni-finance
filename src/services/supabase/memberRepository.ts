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
    const { count, error } = await supabase.from('members').select('id', { count: 'exact', head: true })
    if (error) throw new Error(error.message)
    return { count: count ?? 0, syncedAt: new Date().toISOString() }
  },
}
