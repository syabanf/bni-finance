import { useCallback, useMemo, useState } from 'react'
import {
  ArrowDownLeft,
  ArrowUpRight,
  CheckCircle2,
  Radio,
  RefreshCw,
  Trash2,
  XCircle,
} from 'lucide-react'
import {
  Badge,
  Button,
  Card,
  CardBody,
  EmptyState,
  ErrorState,
  PageHeader,
  Skeleton,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { isMockMode } from '@/services/dataSource'
import {
  clearBlackbox,
  INTEGRATION_LABEL,
  listBlackbox,
  type BlackboxEntry,
  type BlackboxFilters,
} from '@/services/api/blackboxService'
import { formatDateTime } from '@/lib/format'

const useMock = isMockMode()

const INTEGRATIONS = [
  { value: 'all', label: 'Semua Integrasi' },
  { value: 'paper_id', label: 'Paper.id' },
  { value: 'xendit', label: 'Xendit' },
  { value: 'bni_vm', label: 'BNI VM' },
] as const

const DIRECTIONS = [
  { value: 'all', label: 'Semua Arah' },
  { value: 'outbound', label: 'Keluar' },
  { value: 'inbound', label: 'Masuk' },
] as const

const STATUSES = [
  { value: 'all', label: 'Semua Status' },
  { value: 'failed', label: 'Gagal saja' },
] as const

/** Pretty-prints a recorded body; strings are shown as-is. */
function formatBody(body: unknown): string {
  if (body === undefined || body === null) return '—'
  if (typeof body === 'string') return body
  try {
    return JSON.stringify(body, null, 2)
  } catch {
    return String(body)
  }
}

export function BlackboxPage() {
  const { toast } = useToast()
  const [filters, setFilters] = useState<BlackboxFilters>({
    integration: 'all',
    direction: 'all',
    status: 'all',
  })
  const [expanded, setExpanded] = useState<string | null>(null)
  const [clearing, setClearing] = useState(false)

  const fetcher = useCallback(() => listBlackbox(filters), [filters])
  const { data: entries, loading, error, reload } = useAsync<BlackboxEntry[]>(fetcher, [filters])

  const stats = useMemo(() => {
    const list = entries ?? []
    return {
      total: list.length,
      failed: list.filter((e) => !e.success).length,
    }
  }, [entries])

  const handleClear = async () => {
    setClearing(true)
    try {
      await clearBlackbox()
      toast('Rekaman dikosongkan.')
      reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal mengosongkan rekaman.', 'error')
    } finally {
      setClearing(false)
    }
  }

  // The recorder lives on the server, where the integration calls actually
  // happen — there is nothing to show when the app runs on mock data.
  if (useMock) {
    return (
      <div>
        <PageHeader
          title="Blackbox Integrasi"
          description="Rekaman panggilan ke Paper.id, Xendit, dan BNI VM."
        />
        <Card>
          <CardBody>
            <EmptyState
              icon={Radio}
              title="Tidak tersedia pada Data Contoh"
              description="Perekam berjalan di server, tempat panggilan integrasi benar-benar terjadi. Beralihlah ke sumber data Backend API di halaman Pengaturan."
            />
          </CardBody>
        </Card>
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Blackbox Integrasi"
        description="Setiap panggilan ke dan dari Paper.id, Xendit, dan BNI VM — request, endpoint, response, dan statusnya."
        action={
          <div className="flex gap-2">
            <Button variant="outline" onClick={reload} disabled={loading}>
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              Muat Ulang
            </Button>
            <Button
              variant="outline"
              onClick={handleClear}
              loading={clearing}
              disabled={!entries?.length}
            >
              <Trash2 className="h-4 w-4" />
              Kosongkan
            </Button>
          </div>
        }
      />

      {/* Filters */}
      <Card className="mb-5">
        <CardBody className="flex flex-wrap items-center gap-3">
          <Select
            value={filters.integration ?? 'all'}
            options={INTEGRATIONS}
            onChange={(v) => setFilters((f) => ({ ...f, integration: v as BlackboxFilters['integration'] }))}
          />
          <Select
            value={filters.direction ?? 'all'}
            options={DIRECTIONS}
            onChange={(v) => setFilters((f) => ({ ...f, direction: v as BlackboxFilters['direction'] }))}
          />
          <Select
            value={filters.status ?? 'all'}
            options={STATUSES}
            onChange={(v) => setFilters((f) => ({ ...f, status: v as BlackboxFilters['status'] }))}
          />
          <span className="ml-auto text-sm text-ink-500">
            {stats.total} panggilan
            {stats.failed > 0 && <span className="ml-2 text-red-600">· {stats.failed} gagal</span>}
          </span>
        </CardBody>
      </Card>

      {error && <ErrorState message={error} onRetry={reload} />}

      {loading && !entries && (
        <div className="space-y-3">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      )}

      {entries && entries.length === 0 && (
        <Card>
          <CardBody>
            <EmptyState
              icon={Radio}
              title="Belum ada panggilan terekam"
              description="Terbitkan invoice ke Paper.id, jalankan sinkronisasi BNI VM, atau tunggu callback pembayaran — panggilannya akan muncul di sini."
            />
          </CardBody>
        </Card>
      )}

      <div className="space-y-2">
        {entries?.map((entry) => (
          <EntryRow
            key={entry.id}
            entry={entry}
            open={expanded === entry.id}
            onToggle={() => setExpanded(expanded === entry.id ? null : entry.id)}
          />
        ))}
      </div>

      {entries && entries.length > 0 && (
        <p className="mt-5 text-xs text-ink-400">
          Rekaman disimpan di memori server (ring buffer) dan hilang saat server dimulai ulang.
          Kredensial tidak pernah terekam — hanya body yang dicatat, sementara token dan
          client secret berada di header.
        </p>
      )}
    </div>
  )
}

function EntryRow({
  entry,
  open,
  onToggle,
}: {
  entry: BlackboxEntry
  open: boolean
  onToggle: () => void
}) {
  const outbound = entry.direction === 'outbound'

  return (
    <Card className={entry.success ? undefined : 'border-red-200'}>
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        className="flex w-full flex-wrap items-center gap-3 p-4 text-left transition hover:bg-ink-50"
      >
        <span
          className={`flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg ${
            entry.success ? 'bg-emerald-50 text-emerald-600' : 'bg-red-50 text-red-600'
          }`}
          title={entry.success ? 'Berhasil' : 'Gagal'}
        >
          {entry.success ? <CheckCircle2 className="h-5 w-5" /> : <XCircle className="h-5 w-5" />}
        </span>

        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex flex-wrap items-center gap-2">
            <Badge tone={entry.success ? 'green' : 'red'}>
              {entry.success ? 'SUCCESS' : 'FAILED'}
            </Badge>
            <Badge tone="blue" dot={false}>{INTEGRATION_LABEL[entry.integration] ?? entry.integration}</Badge>
            <span
              className="inline-flex items-center gap-1 text-xs text-ink-500"
              title={outbound ? 'Permintaan kita ke pihak luar' : 'Callback dari pihak luar'}
            >
              {outbound ? (
                <ArrowUpRight className="h-3.5 w-3.5" />
              ) : (
                <ArrowDownLeft className="h-3.5 w-3.5" />
              )}
              {outbound ? 'Keluar' : 'Masuk'}
            </span>
            {entry.status > 0 && (
              <span className="font-mono text-xs text-ink-500">HTTP {entry.status}</span>
            )}
            <span className="text-xs text-ink-400">{entry.durationMs} ms</span>
          </span>
          <span className="truncate font-mono text-[13px] text-ink-700">
            <span className="font-semibold">{entry.method}</span> {entry.url}
          </span>
        </span>

        <span className="text-xs text-ink-400">{formatDateTime(entry.time)}</span>
      </button>

      {open && (
        <div className="space-y-4 border-t border-ink-100 p-4">
          {entry.error && (
            <div className="rounded-lg bg-red-50 p-3 text-sm text-red-700">{entry.error}</div>
          )}
          <BodyBlock title="Request" body={entry.request} />
          <BodyBlock title="Response" body={entry.response} />
        </div>
      )}
    </Card>
  )
}

function BodyBlock({ title, body }: { title: string; body: unknown }) {
  const text = formatBody(body)
  return (
    <div>
      <h4 className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-ink-400">{title}</h4>
      <pre className="max-h-80 overflow-auto rounded-lg bg-ink-900 p-3 text-xs leading-relaxed text-ink-100">
        {text}
      </pre>
    </div>
  )
}

function Select({
  value,
  options,
  onChange,
}: {
  value: string
  options: readonly { value: string; label: string }[]
  onChange: (v: string) => void
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="rounded-lg border border-ink-200 bg-white px-3 py-2 text-sm text-ink-700 focus:border-brand-500 focus:outline-none"
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  )
}
