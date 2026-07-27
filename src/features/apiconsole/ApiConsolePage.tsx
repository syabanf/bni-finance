/**
 * Postman-style console for the whole API.
 *
 * The endpoint list is generated from the backend's OpenAPI spec
 * (scripts/build-api-collection.mjs), so it cannot drift from the real routes.
 * Requests run against the mock or the live backend depending on the data
 * source switch — the same toggle the rest of the app obeys.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  AlertTriangle,
  Check,
  ChevronDown,
  Copy,
  Database,
  Play,
  Search,
  Server,
  Trash2,
  Wand2,
} from 'lucide-react'
import {
  Badge,
  Button,
  Card,
  CardBody,
  ErrorState,
  Modal,
  PageHeader,
  Skeleton,
  useToast,
} from '@/components/ui'
import { getToken } from '@/lib/apiClient'
import { isMockMode } from '@/services/dataSource'
import { autofill } from '@/services/apiAutofill'
import {
  buildPath,
  loadCollection,
  sendConsoleRequest,
  toCurl,
  WRITE_METHODS,
  type ApiCollection,
  type ConsoleOperation,
  type ConsoleResult,
} from '@/services/apiConsole'

const METHOD_TONE: Record<string, string> = {
  GET: 'text-emerald-700 bg-emerald-50',
  POST: 'text-amber-700 bg-amber-50',
  PUT: 'text-blue-700 bg-blue-50',
  PATCH: 'text-purple-700 bg-purple-50',
  DELETE: 'text-red-700 bg-red-50',
}

/** Per-operation form state, kept so switching endpoints doesn't lose input. */
interface Draft {
  pathValues: Record<string, string>
  queryValues: Record<string, string>
  body: string
}

function emptyDraft(op: ConsoleOperation): Draft {
  const pathValues: Record<string, string> = {}
  const queryValues: Record<string, string> = {}
  for (const p of op.params) {
    if (p.in === 'path') pathValues[p.name] = ''
    if (p.in === 'query') queryValues[p.name] = ''
  }
  return {
    pathValues,
    queryValues,
    body: op.body === undefined ? '' : JSON.stringify(op.body, null, 2),
  }
}

export function ApiConsolePage() {
  const { toast } = useToast()
  const mock = isMockMode()

  const [collection, setCollection] = useState<ApiCollection | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [filter, setFilter] = useState('')
  const [drafts, setDrafts] = useState<Record<string, Draft>>({})
  const [tab, setTab] = useState<'params' | 'body' | 'docs'>('params')
  const [result, setResult] = useState<ConsoleResult | null>(null)
  const [sending, setSending] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [filling, setFilling] = useState(false)
  const [sendError, setSendError] = useState<string | null>(null)
  const [history, setHistory] = useState<
    { id: string; method: string; path: string; status: number; ms: number }[]
  >([])

  useEffect(() => {
    loadCollection()
      .then((c) => {
        setCollection(c)
        setSelectedId((id) => id ?? c.operations[0]?.id ?? null)
      })
      .catch((err: unknown) => setLoadError(err instanceof Error ? err.message : String(err)))
  }, [])

  const operation = useMemo(
    () => collection?.operations.find((o) => o.id === selectedId) ?? null,
    [collection, selectedId],
  )

  const draft = operation ? (drafts[operation.id] ?? emptyDraft(operation)) : null

  const patchDraft = useCallback(
    (patch: Partial<Draft>) => {
      if (!operation) return
      setDrafts((prev) => ({
        ...prev,
        [operation.id]: { ...(prev[operation.id] ?? emptyDraft(operation)), ...patch },
      }))
    },
    [operation],
  )

  // Grouped by tag, honouring the spec's tag order.
  const groups = useMemo(() => {
    if (!collection) return []
    const q = filter.trim().toLowerCase()
    const match = (op: ConsoleOperation) =>
      !q ||
      op.path.toLowerCase().includes(q) ||
      op.summary.toLowerCase().includes(q) ||
      op.method.toLowerCase() === q
    return collection.tags
      .map((t) => ({
        name: t.name,
        description: t.description,
        ops: collection.operations.filter((o) => o.tag === t.name && match(o)),
      }))
      .filter((g) => g.ops.length > 0)
  }, [collection, filter])

  const resolvedPath = operation && draft ? buildPath(operation.path, draft.pathValues) : ''
  const missingPathParam = resolvedPath.includes('{')

  /** Fires the request for real. Callers handle the confirmation gate. */
  const runSend = async (operation: ConsoleOperation, draft: Draft, resolvedPath: string) => {
    setSending(true)
    setSendError(null)
    try {
      const res = await sendConsoleRequest({
        method: operation.method,
        path: resolvedPath,
        queryValues: draft.queryValues,
        body: WRITE_METHODS.has(operation.method) ? draft.body : undefined,
      })
      setResult(res)
      setHistory((h) =>
        [
          { id: `${Date.now()}`, method: operation.method, path: resolvedPath, status: res.status, ms: res.durationMs },
          ...h,
        ].slice(0, 12),
      )
    } catch (err) {
      setSendError(err instanceof Error ? err.message : String(err))
      setResult(null)
    } finally {
      setSending(false)
    }
  }

  const handleAutofill = async () => {
    if (!operation || !draft) return
    setFilling(true)
    try {
      const res = await autofill(operation)
      patchDraft({
        pathValues: { ...draft.pathValues, ...res.pathValues },
        queryValues: { ...draft.queryValues, ...res.queryValues },
        ...(res.body !== undefined ? { body: res.body } : {}),
      })
      toast(res.notes.length ? `Terisi: ${res.notes.join(', ')}.` : 'Tidak ada yang perlu diisi.')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal mengisi otomatis.', 'error')
    } finally {
      setFilling(false)
    }
  }

  const handleSend = () => {
    if (!operation || !draft) return
    // A live write can change real records; mock mode needs no such ceremony.
    if (!mock && WRITE_METHODS.has(operation.method)) {
      setConfirming(true)
      return
    }
    void runSend(operation, draft, resolvedPath)
  }

  if (loadError) {
    return (
      <div>
        <PageHeader title="Konsol API" description="Uji setiap endpoint langsung dari browser." />
        <ErrorState message={loadError} onRetry={() => window.location.reload()} />
      </div>
    )
  }

  if (!collection || !operation || !draft) {
    return (
      <div>
        <PageHeader title="Konsol API" description="Uji setiap endpoint langsung dari browser." />
        <div className="space-y-3">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    )
  }

  const pathParams = operation.params.filter((p) => p.in === 'path')
  const queryParams = operation.params.filter((p) => p.in === 'query')
  const hasBody = WRITE_METHODS.has(operation.method) && operation.body !== undefined
  // Nothing to fill on an endpoint with neither path params nor a body.
  const canAutofill = pathParams.length > 0 || hasBody

  return (
    <div>
      <PageHeader
        title="Konsol API"
        description={`${collection.operations.length} endpoint, dibangkitkan dari spesifikasi OpenAPI backend.`}
        action={
          <Badge tone={mock ? 'amber' : 'green'} dot={false}>
            {mock ? (
              <span className="inline-flex items-center gap-1">
                <Database className="h-3.5 w-3.5" /> Data Contoh
              </span>
            ) : (
              <span className="inline-flex items-center gap-1">
                <Server className="h-3.5 w-3.5" /> Backend API
              </span>
            )}
          </Badge>
        }
      />

      <div className="grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)]">
        {/* --- collection sidebar --- */}
        <Card className="h-fit lg:sticky lg:top-4">
          <div className="border-b border-ink-100 p-3">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
              <input
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder="Cari endpoint…"
                className="w-full rounded-lg border border-ink-200 py-2 pl-8 pr-3 text-sm focus:border-brand-500 focus:outline-none"
              />
            </div>
          </div>
          <div className="max-h-[70vh] overflow-y-auto p-2">
            {groups.length === 0 && (
              <p className="px-2 py-6 text-center text-sm text-ink-400">Tidak ada yang cocok.</p>
            )}
            {groups.map((group) => (
              <TagGroup
                key={group.name}
                name={group.name}
                ops={group.ops}
                selectedId={selectedId}
                onSelect={(id) => {
                  setSelectedId(id)
                  setResult(null)
                  setSendError(null)
                  setTab('params')
                }}
                defaultOpen={group.ops.some((o) => o.id === selectedId) || filter.trim() !== ''}
              />
            ))}
          </div>
        </Card>

        {/* --- request + response --- */}
        <div className="min-w-0 space-y-4">
          <Card>
            <CardBody className="space-y-3">
              <div className="flex flex-wrap items-center gap-2">
                <span
                  className={`rounded-md px-2.5 py-1 font-mono text-xs font-bold ${
                    METHOD_TONE[operation.method] ?? 'bg-ink-100 text-ink-700'
                  }`}
                >
                  {operation.method}
                </span>
                <code className="min-w-0 flex-1 truncate rounded-lg bg-ink-50 px-3 py-2 font-mono text-[13px] text-ink-700">
                  {resolvedPath}
                </code>
                {canAutofill && (
                  <Button
                    variant="outline"
                    onClick={handleAutofill}
                    loading={filling}
                    title="Ambil id dan tanggal yang benar-benar ada, lewat sumber data yang sedang aktif"
                  >
                    <Wand2 className="h-4 w-4" />
                    Isi Otomatis
                  </Button>
                )}
                <Button onClick={handleSend} loading={sending} disabled={missingPathParam}>
                  <Play className="h-4 w-4" />
                  Kirim
                </Button>
              </div>

              <div className="flex flex-wrap items-center gap-2 text-xs text-ink-500">
                <span className="font-medium text-ink-700">{operation.summary}</span>
                {operation.admin && <Badge tone="purple" dot={false}>Admin saja</Badge>}
                {!operation.auth && <Badge tone="gray" dot={false}>Tanpa token</Badge>}
                {operation.auth && !getToken() && !mock && (
                  <span className="inline-flex items-center gap-1 text-amber-700">
                    <AlertTriangle className="h-3.5 w-3.5" /> belum login — akan dijawab 401
                  </span>
                )}
              </div>

              {missingPathParam && (
                <p className="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800">
                  Isi dulu parameter path yang masih bertanda kurung kurawal.
                </p>
              )}

              {/* tabs */}
              <div className="flex gap-1 border-b border-ink-100">
                <Tab active={tab === 'params'} onClick={() => setTab('params')}>
                  Parameter
                  {operation.params.length > 0 && (
                    <span className="ml-1.5 text-ink-400">{operation.params.length}</span>
                  )}
                </Tab>
                {hasBody && (
                  <Tab active={tab === 'body'} onClick={() => setTab('body')}>
                    Body
                  </Tab>
                )}
                <Tab active={tab === 'docs'} onClick={() => setTab('docs')}>
                  Keterangan
                </Tab>
              </div>

              {tab === 'params' && (
                <div className="space-y-4 pt-1">
                  {operation.params.length === 0 && (
                    <p className="py-3 text-sm text-ink-400">Endpoint ini tidak punya parameter.</p>
                  )}
                  {pathParams.length > 0 && (
                    <ParamTable
                      title="Path"
                      params={pathParams}
                      values={draft.pathValues}
                      onChange={(name, value) =>
                        patchDraft({ pathValues: { ...draft.pathValues, [name]: value } })
                      }
                    />
                  )}
                  {queryParams.length > 0 && (
                    <ParamTable
                      title="Query"
                      params={queryParams}
                      values={draft.queryValues}
                      onChange={(name, value) =>
                        patchDraft({ queryValues: { ...draft.queryValues, [name]: value } })
                      }
                    />
                  )}
                </div>
              )}

              {tab === 'body' && hasBody && (
                <div className="space-y-2 pt-1">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold uppercase tracking-wide text-ink-400">
                      JSON {operation.bodyRequired && <span className="text-red-500">wajib</span>}
                    </span>
                    <button
                      type="button"
                      className="text-xs text-brand-600 hover:underline"
                      onClick={() => patchDraft({ body: JSON.stringify(operation.body, null, 2) })}
                    >
                      Kembalikan contoh
                    </button>
                  </div>
                  <textarea
                    value={draft.body}
                    onChange={(e) => patchDraft({ body: e.target.value })}
                    spellCheck={false}
                    rows={12}
                    className="w-full rounded-lg border border-ink-200 bg-ink-900 p-3 font-mono text-xs leading-relaxed text-ink-100 focus:border-brand-500 focus:outline-none"
                  />
                </div>
              )}

              {tab === 'docs' && (
                <div className="space-y-3 pt-1 text-sm text-ink-600">
                  {operation.description ? (
                    <Prose text={operation.description} />
                  ) : (
                    <p className="text-ink-400">Belum ada keterangan tambahan di spesifikasi.</p>
                  )}
                  <CurlBlock
                    curl={toCurl(
                      {
                        method: operation.method,
                        path: resolvedPath,
                        queryValues: draft.queryValues,
                        body: hasBody ? draft.body : undefined,
                      },
                      getToken(),
                    )}
                    onCopied={() => toast('Perintah curl disalin.')}
                  />
                </div>
              )}
            </CardBody>
          </Card>

          {sendError && (
            <Card className="border-red-200">
              <CardBody className="text-sm text-red-700">{sendError}</CardBody>
            </Card>
          )}

          {result && <ResponsePane result={result} />}

          {history.length > 0 && (
            <Card>
              <CardBody>
                <div className="mb-2 flex items-center justify-between">
                  <h3 className="text-xs font-semibold uppercase tracking-wide text-ink-400">
                    Riwayat
                  </h3>
                  <button
                    type="button"
                    onClick={() => setHistory([])}
                    className="inline-flex items-center gap-1 text-xs text-ink-400 hover:text-ink-600"
                  >
                    <Trash2 className="h-3.5 w-3.5" /> Bersihkan
                  </button>
                </div>
                <ul className="divide-y divide-ink-100 text-[13px]">
                  {history.map((h) => (
                    <li key={h.id} className="flex items-center gap-2 py-1.5">
                      <span className="w-14 font-mono text-xs font-semibold text-ink-500">
                        {h.method}
                      </span>
                      <span className="min-w-0 flex-1 truncate font-mono text-ink-600">{h.path}</span>
                      <span
                        className={`font-mono text-xs ${
                          h.status >= 200 && h.status < 300 ? 'text-emerald-600' : 'text-red-600'
                        }`}
                      >
                        {h.status || 'ERR'}
                      </span>
                      <span className="w-14 text-right text-xs text-ink-400">{h.ms} ms</span>
                    </li>
                  ))}
                </ul>
              </CardBody>
            </Card>
          )}
        </div>
      </div>

      <Modal
        open={confirming}
        onClose={() => setConfirming(false)}
        title="Kirim ke backend sungguhan?"
        description="Permintaan ini mengubah data, bukan sekadar membaca."
        footer={
          <>
            <Button variant="outline" onClick={() => setConfirming(false)}>
              Batal
            </Button>
            <Button
              onClick={() => {
                setConfirming(false)
                void runSend(operation, draft, resolvedPath)
              }}
            >
              <Play className="h-4 w-4" />
              Kirim {operation.method}
            </Button>
          </>
        }
      >
        <code className="block break-all rounded-lg bg-ink-50 px-3 py-2 font-mono text-[13px] text-ink-700">
          {operation.method} {resolvedPath}
        </code>
        <p className="mt-3 text-sm text-ink-500">
          Sumber data sedang <strong>Backend API</strong>, jadi perubahan tersimpan sungguhan.
          Untuk mencoba tanpa risiko, alihkan ke Data Contoh di halaman Pengaturan.
        </p>
      </Modal>
    </div>
  )
}

// --- pieces ------------------------------------------------------------------

function TagGroup({
  name,
  ops,
  selectedId,
  onSelect,
  defaultOpen,
}: {
  name: string
  ops: ConsoleOperation[]
  selectedId: string | null
  onSelect: (id: string) => void
  defaultOpen: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)
  // Searching should reveal matches even inside collapsed groups.
  const lastDefault = useRef(defaultOpen)
  useEffect(() => {
    if (defaultOpen !== lastDefault.current) {
      lastDefault.current = defaultOpen
      if (defaultOpen) setOpen(true)
    }
  }, [defaultOpen])

  return (
    <div className="mb-1">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-1.5 rounded-lg px-2 py-1.5 text-left text-xs font-semibold uppercase tracking-wide text-ink-500 hover:bg-ink-50"
      >
        <ChevronDown className={`h-3.5 w-3.5 transition ${open ? '' : '-rotate-90'}`} />
        {name}
        <span className="ml-auto font-normal text-ink-400">{ops.length}</span>
      </button>
      {open && (
        <ul>
          {ops.map((op) => (
            <li key={op.id}>
              <button
                type="button"
                onClick={() => onSelect(op.id)}
                aria-label={`${op.method} ${op.path} — ${op.summary}`}
                aria-current={op.id === selectedId ? 'true' : undefined}
                className={`flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left transition ${
                  op.id === selectedId ? 'bg-brand-50 text-brand-800' : 'hover:bg-ink-50'
                }`}
              >
                <span
                  className={`w-12 flex-shrink-0 rounded px-1 py-0.5 text-center font-mono text-[10px] font-bold ${
                    METHOD_TONE[op.method] ?? 'bg-ink-100 text-ink-700'
                  }`}
                >
                  {op.method}
                </span>
                <span className="min-w-0 flex-1 truncate text-[13px]" title={`${op.path} — ${op.summary}`}>
                  {op.path.replace(/^\/api\/v1/, '')}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function Tab({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`-mb-px border-b-2 px-3 py-2 text-sm transition ${
        active
          ? 'border-brand-500 font-medium text-brand-700'
          : 'border-transparent text-ink-500 hover:text-ink-700'
      }`}
    >
      {children}
    </button>
  )
}

function ParamTable({
  title,
  params,
  values,
  onChange,
}: {
  title: string
  params: { name: string; required: boolean; type: string; enum?: string[]; default?: unknown; description?: string }[]
  values: Record<string, string>
  onChange: (name: string, value: string) => void
}) {
  return (
    <div>
      <h4 className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-ink-400">{title}</h4>
      <div className="space-y-2">
        {params.map((p) => (
          <div key={p.name} className="grid gap-1.5 sm:grid-cols-[180px_minmax(0,1fr)] sm:items-center">
            <label className="flex items-baseline gap-1.5 font-mono text-[13px] text-ink-700">
              {p.name}
              {p.required && <span className="text-red-500">*</span>}
              <span className="font-sans text-[11px] text-ink-400">{p.type}</span>
            </label>
            <div>
              {p.enum?.length ? (
                <select
                  value={values[p.name] ?? ''}
                  onChange={(e) => onChange(p.name, e.target.value)}
                  className="w-full rounded-lg border border-ink-200 bg-white px-3 py-1.5 text-sm focus:border-brand-500 focus:outline-none"
                >
                  <option value="">— kosong —</option>
                  {p.enum.map((v) => (
                    <option key={v} value={v}>
                      {v}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  value={values[p.name] ?? ''}
                  onChange={(e) => onChange(p.name, e.target.value)}
                  placeholder={p.default !== undefined ? `bawaan: ${String(p.default)}` : ''}
                  className="w-full rounded-lg border border-ink-200 px-3 py-1.5 font-mono text-[13px] focus:border-brand-500 focus:outline-none"
                />
              )}
              {p.description && <p className="mt-0.5 text-xs text-ink-400">{p.description}</p>}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function ResponsePane({ result }: { result: ConsoleResult }) {
  const [view, setView] = useState<'pretty' | 'raw' | 'headers'>('pretty')
  const failed = !result.ok

  const pretty = useMemo(() => {
    if (result.transportError) return result.transportError
    if (result.raw === '') return '(tanpa isi — 204 No Content)'
    try {
      return JSON.stringify(result.body, null, 2)
    } catch {
      return result.raw
    }
  }, [result])

  return (
    <Card className={failed ? 'border-red-200' : undefined}>
      <div className="flex flex-wrap items-center gap-3 border-b border-ink-100 px-4 py-3">
        <Badge tone={failed ? 'red' : 'green'}>{failed ? 'FAILED' : 'SUCCESS'}</Badge>
        <span className="font-mono text-sm font-semibold text-ink-700">
          {result.status === 0 ? 'NO RESPONSE' : result.status}
        </span>
        <span className="text-xs text-ink-400">{result.durationMs} ms</span>
        <span className="text-xs text-ink-400">{result.bytes} B</span>
        <span className="ml-auto truncate font-mono text-xs text-ink-400" title={result.url}>
          {result.mode === 'mock' ? 'dijawab mock di browser' : result.url}
        </span>
      </div>

      <div className="flex gap-1 border-b border-ink-100 px-2">
        <Tab active={view === 'pretty'} onClick={() => setView('pretty')}>
          Response
        </Tab>
        <Tab active={view === 'raw'} onClick={() => setView('raw')}>
          Raw
        </Tab>
        <Tab active={view === 'headers'} onClick={() => setView('headers')}>
          Headers
        </Tab>
      </div>

      <div className="p-4">
        {view === 'headers' ? (
          <dl className="grid gap-1 font-mono text-xs sm:grid-cols-[220px_minmax(0,1fr)]">
            {Object.entries(result.headers).map(([k, v]) => (
              <div key={k} className="contents">
                <dt className="text-ink-500">{k}</dt>
                <dd className="break-all text-ink-700">{v}</dd>
              </div>
            ))}
            {Object.keys(result.headers).length === 0 && (
              <p className="text-ink-400">Tidak ada header terbaca.</p>
            )}
          </dl>
        ) : (
          <pre className="max-h-[28rem] overflow-auto rounded-lg bg-ink-900 p-3 text-xs leading-relaxed text-ink-100">
            {view === 'pretty' ? pretty : result.raw || '(kosong)'}
          </pre>
        )}
      </div>
    </Card>
  )
}

function CurlBlock({ curl, onCopied }: { curl: string; onCopied: () => void }) {
  const [copied, setCopied] = useState(false)
  return (
    <div>
      <div className="mb-1.5 flex items-center justify-between">
        <h4 className="text-xs font-semibold uppercase tracking-wide text-ink-400">curl</h4>
        <button
          type="button"
          className="inline-flex items-center gap-1 text-xs text-brand-600 hover:underline"
          onClick={() => {
            void navigator.clipboard.writeText(curl).then(() => {
              setCopied(true)
              onCopied()
              setTimeout(() => setCopied(false), 1500)
            })
          }}
        >
          {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          {copied ? 'Tersalin' : 'Salin'}
        </button>
      </div>
      <pre className="overflow-x-auto rounded-lg bg-ink-900 p-3 text-xs text-ink-100">{curl}</pre>
    </div>
  )
}

/** Renders the spec's light markdown: paragraphs, `code`, and **bold**. */
function Prose({ text }: { text: string }) {
  const html = useMemo(
    () =>
      text
        .split(/\n{2,}/)
        .map((para) =>
          para
            .replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' })[c]!)
            .replace(/`([^`]+)`/g, '<code class="rounded bg-ink-100 px-1 py-0.5 text-[12px]">$1</code>')
            .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
            .replace(/\n/g, ' '),
        )
        .map((para) => `<p class="mb-2">${para}</p>`)
        .join(''),
    [text],
  )
  return <div dangerouslySetInnerHTML={{ __html: html }} />
}
