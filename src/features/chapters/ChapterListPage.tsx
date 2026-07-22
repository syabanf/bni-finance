import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Building2, MapPin, Search, Users, X } from 'lucide-react'
import type { Chapter, InvoiceWithRelations, MemberWithChapter } from '@/types'
import {
  Card,
  EmptyState,
  ExportMenu,
  Input,
  LoadingState,
  PageHeader,
  Select,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, invoiceService, memberService } from '@/services'
import { formatCurrency, formatCurrencyCompact, formatDateTime } from '@/lib/format'
import { makeExportHandlers } from '@/lib/exporters'

export function ChapterListPage() {
  const navigate = useNavigate()
  const { toast } = useToast()
  const { data: chapters, loading } = useAsync<Chapter[]>(() => chapterService.list())
  const { data: members } = useAsync<MemberWithChapter[]>(() => memberService.list())
  const { data: invoices } = useAsync<InvoiceWithRelations[]>(() => invoiceService.list())

  const [city, setCity] = useState('all')
  const [search, setSearch] = useState('')

  const cities = useMemo(
    () =>
      (Array.from(new Set((chapters ?? []).map((c) => c.cityName).filter(Boolean))) as string[]).sort(),
    [chapters],
  )

  const filteredChapters = useMemo(() => {
    const q = search.trim().toLowerCase()
    return (chapters ?? []).filter((c) => {
      if (city !== 'all' && c.cityName !== city) return false
      if (
        q &&
        !c.displayName.toLowerCase().includes(q) &&
        !(c.cityName ?? '').toLowerCase().includes(q) &&
        !(c.areaName ?? '').toLowerCase().includes(q)
      )
        return false
      return true
    })
  }, [chapters, city, search])

  const memberCount = (chapterId: string) =>
    members?.filter((m) => m.chapterId === chapterId).length ?? 0

  // Outstanding = invoice yang sudah diterbitkan tapi belum dibayar (sent + overdue).
  const outstanding = (chapterId: string) =>
    invoices
      ?.filter((i) => i.chapterId === chapterId && (i.status === 'sent' || i.status === 'overdue'))
      .reduce((acc, i) => acc + i.amount, 0) ?? 0

  const totalMembers = filteredChapters.reduce((a, c) => a + memberCount(c.id), 0)
  const totalOutstanding = filteredChapters.reduce((a, c) => a + outstanding(c.id), 0)

  // Exports reflect the active filters.
  const exportHandlers = makeExportHandlers({
    filename: 'chapter',
    title: 'Daftar Chapter',
    subtitle: `Kota: ${city === 'all' ? 'Semua' : city}`,
    meta: [`${filteredChapters.length} chapter`, `Dibuat ${formatDateTime(new Date())}`],
    columns: [
      { label: 'Chapter' },
      { label: 'Kota' },
      { label: 'Member', align: 'right' },
      { label: 'Outstanding', align: 'right' },
    ],
    rows: filteredChapters.map((c) => [
      c.displayName,
      c.cityName ?? c.areaName ?? '',
      memberCount(c.id),
      outstanding(c.id),
    ]),
    pdfRows: filteredChapters.map((c) => [
      c.displayName,
      c.cityName ?? c.areaName ?? '—',
      memberCount(c.id),
      formatCurrency(outstanding(c.id)),
    ]),
    totals: ['Total', '', totalMembers, formatCurrency(totalOutstanding)],
    onPopupBlocked: () => toast('Izinkan popup di browser untuk mengekspor PDF.', 'error'),
  })

  return (
    <div>
      <PageHeader
        title="Chapter"
        description="Daftar chapter hasil sinkronisasi dari BNI Visitor Management."
        action={<ExportMenu {...exportHandlers} disabled={filteredChapters.length === 0} />}
      />

      {loading ? (
        <LoadingState />
      ) : (
        <>
          {/* Filter: pencarian + kota */}
          <Card className="mb-5">
            <div className="space-y-3 p-4">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
                <Input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Cari nama chapter, kota, atau area…"
                  className="pl-10"
                />
              </div>
              <div className="flex flex-wrap items-center gap-3">
                <span className="text-[13px] text-ink-500">Kota</span>
                <Select value={city} onChange={(e) => setCity(e.target.value)} className="w-full sm:w-56">
                  <option value="all">Semua Kota</option>
                  {cities.map((ct) => (
                    <option key={ct} value={ct}>
                      {ct}
                    </option>
                  ))}
                </Select>
                {(search || city !== 'all') && (
                  <button
                    type="button"
                    onClick={() => {
                      setSearch('')
                      setCity('all')
                    }}
                    className="rounded-lg p-1.5 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-700"
                    aria-label="Reset filter chapter"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
                <span className="ml-auto text-sm text-ink-400">
                  {filteredChapters.length} chapter · Outstanding {formatCurrencyCompact(totalOutstanding)}
                </span>
              </div>
            </div>
          </Card>

          {filteredChapters.length === 0 ? (
            <Card>
              <EmptyState
                icon={Building2}
                title="Tidak ada chapter"
                description="Tidak ada chapter yang cocok dengan filter saat ini."
              />
            </Card>
          ) : (
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
          )}
        </>
      )}
    </div>
  )
}
