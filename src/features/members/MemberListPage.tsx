import { useMemo, useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ArrowRight, ChevronLeft, ChevronRight, Download, Eye, Search, Users, X } from 'lucide-react'
import type { Chapter, MemberStatus, MemberWithChapter } from '@/types'
import {
  Avatar,
  Button,
  Card,
  EmptyState,
  Input,
  MemberStatusBadge,
  PageHeader,
  Select,
  SummaryCard,
  Table,
  TBody,
  Td,
  Th,
  THead,
  Tr,
  TableSkeleton,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, memberService } from '@/services'
import { formatDate } from '@/lib/format'
import { downloadCsv } from '@/lib/csv'

export function MemberListPage() {
  const navigate = useNavigate()
  const { data: members, loading } = useAsync<MemberWithChapter[]>(() => memberService.list())
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())

  const [searchParams] = useSearchParams()
  const [search, setSearch] = useState('')
  const [chapterId, setChapterId] = useState(searchParams.get('chapter') ?? 'all')
  const [hideNoDueDate, setHideNoDueDate] = useState(false)
  const [dueFrom, setDueFrom] = useState('')
  const [dueTo, setDueTo] = useState('')
  const [memberStatus, setMemberStatus] = useState<MemberStatus | 'all'>('all')
  const [page, setPage] = useState(1)
  const PAGE_SIZE = 25

  // Everything except the status filter — drives the summary cards so they
  // reflect chapter/due-date/search while still showing the status breakdown.
  const baseFiltered = useMemo(() => {
    if (!members) return []
    const q = search.trim().toLowerCase()
    return members.filter((m) => {
      if (chapterId !== 'all' && m.chapterId !== chapterId) return false
      if (hideNoDueDate && !m.renewalDate) return false
      if (dueFrom && (!m.renewalDate || m.renewalDate < dueFrom)) return false
      if (dueTo && (!m.renewalDate || m.renewalDate > dueTo)) return false
      if (q && !m.name.toLowerCase().includes(q) && !m.id.toLowerCase().includes(q) && !(m.email ?? '').toLowerCase().includes(q))
        return false
      return true
    })
  }, [members, search, chapterId, hideNoDueDate, dueFrom, dueTo])

  const statusCounts = useMemo(() => {
    const list = baseFiltered
    return {
      all: list.length,
      active: list.filter((m) => m.status === 'active').length,
      pending: list.filter((m) => m.status === 'pending').length,
      inactive: list.filter((m) => m.status === 'inactive').length,
    }
  }, [baseFiltered])

  const filtered = useMemo(() => {
    if (memberStatus === 'all') return baseFiltered
    return baseFiltered.filter((m) => m.status === memberStatus)
  }, [baseFiltered, memberStatus])

  // Reset ke halaman 1 setiap kali filter berubah
  useEffect(() => { setPage(1) }, [filtered])

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const paginated = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  return (
    <div>
      <PageHeader
        title="Member"
        description="Data member hasil sinkronisasi dari BNI Visitor Management."
        action={
          <Button
            variant="outline"
            onClick={() =>
              downloadCsv(
                'member.csv',
                ['Nama', 'ID', 'Chapter', 'Email', 'Telepon', 'Status', 'Due Date'],
                filtered.map((m) => [
                  m.name,
                  m.id,
                  m.chapter?.displayName ?? '',
                  m.email ?? '',
                  m.phone ?? '',
                  m.status,
                  m.renewalDate ? formatDate(m.renewalDate) : '',
                ]),
              )
            }
          >
            <Download className="h-4 w-4" />
            Export CSV
          </Button>
        }
      />

      {/* Summary cards (also filter by status) */}
      <div className="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <SummaryCard
          label="Total Member"
          value={statusCounts.all}
          tone="brand"
          active={memberStatus === 'all'}
          onClick={() => setMemberStatus('all')}
        />
        <SummaryCard
          label="Active"
          value={statusCounts.active}
          tone="green"
          active={memberStatus === 'active'}
          onClick={() => setMemberStatus('active')}
        />
        <SummaryCard
          label="Pending"
          value={statusCounts.pending}
          tone="amber"
          active={memberStatus === 'pending'}
          onClick={() => setMemberStatus('pending')}
        />
        <SummaryCard
          label="Inactive"
          value={statusCounts.inactive}
          tone="default"
          active={memberStatus === 'inactive'}
          onClick={() => setMemberStatus('inactive')}
        />
      </div>

      <Card>
        <div className="space-y-3 border-b border-ink-100 p-4">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari nama, ID, atau email…"
              className="pl-10"
            />
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)} className="w-full sm:w-52">
              <option value="all">Semua Chapter</option>
              {chapters?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.displayName}
                </option>
              ))}
            </Select>
            {/* Filter due date (rentang tanggal renewal) */}
            <div className="flex items-center gap-2">
              <span className="text-[13px] text-ink-500">Due date</span>
              <Input
                type="date"
                value={dueFrom}
                max={dueTo || undefined}
                onChange={(e) => setDueFrom(e.target.value)}
                className="w-[150px]"
                aria-label="Due date dari"
              />
              <span className="text-ink-400">–</span>
              <Input
                type="date"
                value={dueTo}
                min={dueFrom || undefined}
                onChange={(e) => setDueTo(e.target.value)}
                className="w-[150px]"
                aria-label="Due date sampai"
              />
              {(dueFrom || dueTo) && (
                <button
                  type="button"
                  onClick={() => {
                    setDueFrom('')
                    setDueTo('')
                  }}
                  className="rounded-lg p-1.5 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-700"
                  aria-label="Reset filter due date"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
            <label className="flex shrink-0 cursor-pointer items-center gap-2 text-sm text-ink-600">
              <input
                type="checkbox"
                checked={hideNoDueDate}
                onChange={(e) => setHideNoDueDate(e.target.checked)}
                className="h-4 w-4 rounded border-ink-300 accent-brand-500"
              />
              Ada due date
            </label>
          </div>
        </div>

        {loading ? (
          <TableSkeleton rows={8} cols={5} />
        ) : filtered.length === 0 ? (
          <EmptyState icon={Users} title="Tidak ada member" description="Tidak ada member yang cocok dengan filter." />
        ) : (
          <>
            {/* Mobile card list */}
            <div className="divide-y divide-ink-100 lg:hidden">
              {paginated.map((m) => (
                <div
                  key={m.id}
                  onClick={() => navigate(`/members/${m.id}`)}
                  className="flex items-center gap-3 px-4 py-3.5 active:bg-ink-50"
                >
                  <Avatar name={m.name} size="sm" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="truncate font-medium text-ink-900 text-sm">{m.name}</div>
                        <div className="text-xs text-ink-400 truncate">
                          {m.company && m.businessField
                            ? `${m.company} · ${m.businessField}`
                            : m.company ?? m.businessField ?? m.chapter?.displayName ?? '—'}
                        </div>
                      </div>
                      <div className="shrink-0 text-right">
                        <MemberStatusBadge status={m.status} />
                        <div className="text-xs text-ink-400 mt-1">
                          {m.renewalDate ? formatDate(m.renewalDate) : '—'}
                        </div>
                      </div>
                    </div>
                    {m.email && (
                      <div className="mt-1 truncate text-xs text-ink-400">{m.email}</div>
                    )}
                  </div>
                  <ArrowRight className="h-4 w-4 shrink-0 text-ink-300" />
                </div>
              ))}
            </div>

            {/* Desktop table */}
            <div className="hidden lg:block">
              <Table>
                <THead>
                  <Tr>
                    <Th>Member</Th>
                    <Th>Chapter</Th>
                    <Th>Email</Th>
                    <Th>Status</Th>
                    <Th>Due Date</Th>
                    <Th className="text-right">Aksi</Th>
                  </Tr>
                </THead>
                <TBody>
                  {paginated.map((m) => (
                    <Tr key={m.id} onClick={() => navigate(`/members/${m.id}`)}>
                      <Td>
                        <div className="flex items-center gap-3">
                          <Avatar name={m.name} size="sm" />
                          <div className="leading-tight">
                            <div className="font-medium text-ink-900">{m.name}</div>
                            <div className="text-xs text-ink-400">
                              {m.company && m.businessField
                                ? `${m.company} · ${m.businessField}`
                                : m.company ?? m.businessField ?? '—'}
                            </div>
                          </div>
                        </div>
                      </Td>
                      <Td className="text-ink-600">{m.chapter?.displayName ?? '—'}</Td>
                      <Td className="text-ink-600">{m.email ?? '—'}</Td>
                      <Td>
                        <MemberStatusBadge status={m.status} />
                      </Td>
                      <Td className="whitespace-nowrap text-ink-600">
                        {m.renewalDate ? formatDate(m.renewalDate) : '—'}
                      </Td>
                      <Td className="text-right">
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            navigate(`/members/${m.id}`)
                          }}
                          className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-ink-400 transition-colors hover:bg-ink-100 hover:text-brand-500"
                          aria-label="Lihat detail"
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                      </Td>
                    </Tr>
                  ))}
                </TBody>
              </Table>
            </div>
            {/* Footer: info + pagination */}
            <div className="flex items-center justify-between gap-4 border-t border-ink-100 px-5 py-3">
              <span className="text-xs text-ink-400">
                {filtered.length === 0
                  ? 'Tidak ada member'
                  : `${(page - 1) * PAGE_SIZE + 1}–${Math.min(page * PAGE_SIZE, filtered.length)} dari ${filtered.length} member`}
              </span>
              {totalPages > 1 && (
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                    className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-ink-500 transition-colors hover:bg-ink-100 disabled:opacity-30 disabled:cursor-not-allowed"
                    aria-label="Halaman sebelumnya"
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </button>
                  {Array.from({ length: totalPages }, (_, i) => i + 1)
                    .filter((p) => p === 1 || p === totalPages || Math.abs(p - page) <= 1)
                    .reduce<(number | 'ellipsis')[]>((acc, p, idx, arr) => {
                      if (idx > 0 && p - (arr[idx - 1] as number) > 1) acc.push('ellipsis')
                      acc.push(p)
                      return acc
                    }, [])
                    .map((item, idx) =>
                      item === 'ellipsis' ? (
                        <span key={`e${idx}`} className="px-1 text-xs text-ink-400">…</span>
                      ) : (
                        <button
                          key={item}
                          onClick={() => setPage(item as number)}
                          className={`inline-flex h-8 min-w-[2rem] items-center justify-center rounded-lg px-2 text-xs font-medium transition-colors ${
                            page === item
                              ? 'bg-brand-500 text-white'
                              : 'text-ink-600 hover:bg-ink-100'
                          }`}
                        >
                          {item}
                        </button>
                      ),
                    )}
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages}
                    className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-ink-500 transition-colors hover:bg-ink-100 disabled:opacity-30 disabled:cursor-not-allowed"
                    aria-label="Halaman berikutnya"
                  >
                    <ChevronRight className="h-4 w-4" />
                  </button>
                </div>
              )}
            </div>
          </>
        )}
      </Card>
    </div>
  )
}
