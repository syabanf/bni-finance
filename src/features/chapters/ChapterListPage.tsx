import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Building2, MapPin, Users } from 'lucide-react'
import type { Chapter, InvoiceWithRelations, MemberWithChapter } from '@/types'
import { Card, LoadingState, PageHeader, Select } from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, invoiceService, memberService } from '@/services'
import { formatCurrencyCompact } from '@/lib/format'

export function ChapterListPage() {
  const navigate = useNavigate()
  const { data: chapters, loading } = useAsync<Chapter[]>(() => chapterService.list())
  const { data: members } = useAsync<MemberWithChapter[]>(() => memberService.list())
  const { data: invoices } = useAsync<InvoiceWithRelations[]>(() => invoiceService.list())

  const [city, setCity] = useState('all')

  const cities = useMemo(
    () =>
      (Array.from(new Set((chapters ?? []).map((c) => c.cityName).filter(Boolean))) as string[]).sort(),
    [chapters],
  )
  const filteredChapters = useMemo(
    () => (chapters ?? []).filter((c) => city === 'all' || c.cityName === city),
    [chapters, city],
  )

  const memberCount = (chapterId: string) =>
    members?.filter((m) => m.chapterId === chapterId).length ?? 0

  // Outstanding = invoice yang sudah diterbitkan tapi belum dibayar (sent + overdue).
  const outstanding = (chapterId: string) =>
    invoices
      ?.filter((i) => i.chapterId === chapterId && (i.status === 'sent' || i.status === 'overdue'))
      .reduce((acc, i) => acc + i.amount, 0) ?? 0

  return (
    <div>
      <PageHeader title="Chapter" description="Daftar chapter hasil sinkronisasi dari BNI Visitor Management." />

      {loading ? (
        <LoadingState />
      ) : (
        <>
          {/* Filter kota */}
          <div className="mb-5 flex items-center gap-2">
            <span className="text-sm text-ink-500">Kota</span>
            <Select value={city} onChange={(e) => setCity(e.target.value)} className="w-full sm:w-56">
              <option value="all">Semua Kota</option>
              {cities.map((ct) => (
                <option key={ct} value={ct}>
                  {ct}
                </option>
              ))}
            </Select>
            <span className="ml-auto text-sm text-ink-400">{filteredChapters.length} chapter</span>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {filteredChapters.map((c) => (
              <Card key={c.id} className="p-5 transition-shadow hover:shadow-card-hover">
                <div className="flex items-start justify-between">
                  <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-brand-50 text-brand-500">
                    <Building2 className="h-[22px] w-[22px]" />
                  </div>
                  <button
                    onClick={() => navigate(`/members?chapter=${c.id}`)}
                    className="text-xs font-medium text-brand-500 hover:text-brand-600"
                  >
                    Lihat member
                  </button>
                </div>
                <h3 className="mt-4 text-lg font-bold text-ink-900">{c.displayName}</h3>
                <div className="mt-1 flex items-center gap-1.5 text-sm text-ink-500">
                  <MapPin className="h-3.5 w-3.5" />
                  {c.areaName ?? c.cityName ?? '—'}
                </div>
                <div className="mt-5 grid grid-cols-2 gap-3 border-t border-ink-100 pt-4">
                  <div>
                    <div className="flex items-center gap-1.5 text-xs text-ink-400">
                      <Users className="h-3.5 w-3.5" />
                      Member
                    </div>
                    <div className="mt-0.5 text-lg font-bold text-ink-900">{memberCount(c.id)}</div>
                  </div>
                  <button
                    type="button"
                    onClick={() => navigate(`/invoices?chapter=${c.id}&status=outstanding`)}
                    className="text-left"
                  >
                    <div className="text-xs text-ink-400">Outstanding</div>
                    <div className="mt-0.5 text-lg font-bold text-amber-600 hover:underline">
                      {formatCurrencyCompact(outstanding(c.id))}
                    </div>
                  </button>
                </div>
              </Card>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
