import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ArrowRight, Download, Eye, Search, Users } from 'lucide-react'
import type { Chapter, MemberWithChapter } from '@/types'
import {
  Avatar,
  Button,
  Card,
  EmptyState,
  Input,
  MemberStatusBadge,
  PageHeader,
  Select,
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

function downloadCsv(filename: string, headers: string[], rows: string[][]) {
  const escape = (v: string) => `"${v.replace(/"/g, '""')}"`
  const lines = [headers, ...rows].map((r) => r.map(escape).join(','))
  const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  URL.revokeObjectURL(url)
}

export function MemberListPage() {
  const navigate = useNavigate()
  const { data: members, loading } = useAsync<MemberWithChapter[]>(() => memberService.list())
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())

  const [searchParams] = useSearchParams()
  const [search, setSearch] = useState('')
  const [chapterId, setChapterId] = useState(searchParams.get('chapter') ?? 'all')

  const filtered = useMemo(() => {
    if (!members) return []
    const q = search.trim().toLowerCase()
    return members.filter((m) => {
      if (chapterId !== 'all' && m.chapterId !== chapterId) return false
      if (q && !m.name.toLowerCase().includes(q) && !m.id.toLowerCase().includes(q) && !(m.email ?? '').toLowerCase().includes(q))
        return false
      return true
    })
  }, [members, search, chapterId])

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
                  m.renewalDueDate ? formatDate(m.renewalDueDate) : '',
                ]),
              )
            }
          >
            <Download className="h-4 w-4" />
            Export CSV
          </Button>
        }
      />

      <Card>
        <div className="flex flex-col gap-3 border-b border-ink-100 p-4 sm:flex-row sm:items-center">
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari nama, ID, atau email…"
              className="pl-10"
            />
          </div>
          <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)} className="sm:w-52">
            <option value="all">Semua Chapter</option>
            {chapters?.map((c) => (
              <option key={c.id} value={c.id}>
                {c.displayName}
              </option>
            ))}
          </Select>
        </div>

        {loading ? (
          <TableSkeleton rows={8} cols={5} />
        ) : filtered.length === 0 ? (
          <EmptyState icon={Users} title="Tidak ada member" description="Tidak ada member yang cocok dengan filter." />
        ) : (
          <>
            {/* Mobile card list */}
            <div className="divide-y divide-ink-100 lg:hidden">
              {filtered.map((m) => (
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
                          {m.renewalDueDate ? formatDate(m.renewalDueDate) : '—'}
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
                  {filtered.map((m) => (
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
                        {m.renewalDueDate ? formatDate(m.renewalDueDate) : '—'}
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
            <div className="px-5 py-3 text-xs text-ink-400">
              Menampilkan {filtered.length} dari {members?.length ?? 0} member
            </div>
          </>
        )}
      </Card>
    </div>
  )
}
