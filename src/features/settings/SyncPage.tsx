import { useState } from 'react'
import { Building2, Database, RefreshCw, Users } from 'lucide-react'
import type { Chapter, MemberWithChapter } from '@/types'
import {
  Button,
  Card,
  CardBody,
  CardHeader,
  PageHeader,
  Skeleton,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, memberService } from '@/services'
import { formatDateTime } from '@/lib/format'

interface SyncCardState {
  count: number
  syncedAt: string
}

export function SyncPage() {
  const { toast } = useToast()
  const { data: chapters, loading: chLoading } = useAsync<Chapter[]>(() => chapterService.list())
  const { data: members, loading: mLoading } = useAsync<MemberWithChapter[]>(() => memberService.list())

  const [memberState, setMemberState] = useState<SyncCardState | null>(null)
  const [chapterState, setChapterState] = useState<SyncCardState | null>(null)
  const [syncing, setSyncing] = useState<'members' | 'chapters' | null>(null)

  const memberInfo =
    memberState ??
    (members
      ? { count: members.length, syncedAt: members[0]?.syncedAt ?? new Date().toISOString() }
      : null)
  const chapterInfo =
    chapterState ??
    (chapters
      ? { count: chapters.length, syncedAt: chapters[0]?.syncedAt ?? new Date().toISOString() }
      : null)

  const syncMembers = async () => {
    setSyncing('members')
    try {
      const res = await memberService.sync()
      setMemberState(res)
      toast(`${res.count} member berhasil disinkronkan dari BNI VM.`)
    } catch {
      toast('Gagal menyinkronkan member.', 'error')
    } finally {
      setSyncing(null)
    }
  }

  const syncChapters = async () => {
    setSyncing('chapters')
    try {
      const res = await chapterService.sync()
      setChapterState(res)
      toast(`${res.count} chapter berhasil disinkronkan dari BNI VM.`)
    } catch {
      toast('Gagal menyinkronkan chapter.', 'error')
    } finally {
      setSyncing(null)
    }
  }

  return (
    <div>
      <PageHeader
        title="Sinkronisasi Data"
        description="Tarik data member dan chapter terbaru dari BNI Visitor Management."
      />

      {/* Source info */}
      <Card className="mb-5">
        <CardBody className="flex items-center gap-4 p-5">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-ink-900 text-white">
            <Database className="h-6 w-6" />
          </div>
          <div className="flex-1">
            <div className="font-semibold text-ink-900">BNI Visitor Management</div>
            <div className="text-sm text-ink-500">
              Sumber data · <span className="font-mono text-[13px]">bni-vh.com/api/finance/*</span>
            </div>
          </div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-3 py-1 text-xs font-medium text-emerald-600">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
            Terhubung
          </span>
        </CardBody>
      </Card>

      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <SyncCard
          icon={Users}
          title="Member"
          description="Data member aktif beserta status keanggotaan."
          loading={mLoading}
          info={memberInfo}
          syncing={syncing === 'members'}
          onSync={syncMembers}
        />
        <SyncCard
          icon={Building2}
          title="Chapter"
          description="Daftar chapter beserta area dan kota."
          loading={chLoading}
          info={chapterInfo}
          syncing={syncing === 'chapters'}
          onSync={syncChapters}
        />
      </div>
    </div>
  )
}

function SyncCard({
  icon: Icon,
  title,
  description,
  loading,
  info,
  syncing,
  onSync,
}: {
  icon: typeof Users
  title: string
  description: string
  loading: boolean
  info: SyncCardState | null
  syncing: boolean
  onSync: () => void
}) {
  return (
    <Card>
      <CardHeader
        title={
          <span className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-50 text-brand-500">
              <Icon className="h-5 w-5" />
            </span>
            {title}
          </span>
        }
      />
      <CardBody className="space-y-4">
        <p className="text-sm text-ink-500">{description}</p>

        <div className="grid grid-cols-2 gap-3 rounded-xl bg-ink-50 p-4">
          <div>
            <div className="text-xs text-ink-400">Total Record</div>
            {loading ? (
              <Skeleton className="mt-1 h-6 w-12" />
            ) : (
              <div className="mt-0.5 text-xl font-bold text-ink-900">{info?.count ?? '—'}</div>
            )}
          </div>
          <div>
            <div className="text-xs text-ink-400">Terakhir Sync</div>
            {loading ? (
              <Skeleton className="mt-1 h-6 w-24" />
            ) : (
              <div className="mt-0.5 text-[13px] font-medium text-ink-700">
                {info ? formatDateTime(info.syncedAt) : '—'}
              </div>
            )}
          </div>
        </div>

        <Button variant="outline" className="w-full" onClick={onSync} loading={syncing}>
          {!syncing && <RefreshCw className="h-4 w-4" />}
          Sinkronkan Sekarang
        </Button>
      </CardBody>
    </Card>
  )
}
