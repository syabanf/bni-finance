import { useMemo, useState } from 'react'
import { CheckCircle2, HelpCircle, Send, XCircle } from 'lucide-react'
import type { RenewalAnswer, RenewalRequest } from '@/types'
import {
  Badge,
  Button,
  Card,
  CardBody,
  CardHeader,
  EmptyState,
  ErrorState,
  Input,
  LoadingState,
  PageHeader,
  TBody,
  Table,
  Td,
  THead,
  Th,
  Tr,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { memberService, renewalService } from '@/services'
import { useAuth } from '@/features/auth/AuthContext'
import { formatDate } from '@/lib/format'

/**
 * Konfirmasi renewal — ST menanyakan, MC menjawab.
 *
 * Satu halaman untuk keduanya, bukan dua: ST dan MC melihat DAFTAR YANG SAMA,
 * hanya tombolnya yang berbeda. Memisahkannya akan membuat ST tidak bisa
 * melihat jawaban yang sedang ia tunggu tanpa berpindah layar.
 *
 * ST TIDAK bisa menjawab, dan itu inti alurnya: ia yang bertanya, jadi
 * membiarkannya menjawab sendiri membuat konfirmasinya tidak berarti apa-apa —
 * ia hanya akan mencatat dugaannya sendiri sebagai jawaban orang lain. Server
 * yang menegakkannya; di sini tombolnya sekadar disembunyikan.
 */

const LABEL: Record<RenewalAnswer, string> = {
  pending: 'Belum dijawab',
  will_renew: 'Akan perpanjang',
  will_not: 'Tidak perpanjang',
  unsure: 'Belum pasti',
}

const TONE: Record<RenewalAnswer, 'gray' | 'green' | 'red' | 'amber'> = {
  pending: 'gray',
  will_renew: 'green',
  will_not: 'red',
  unsure: 'amber',
}

const tahunIni = String(new Date().getFullYear() + 1)

export function RenewalPage() {
  const { toast } = useToast()
  const { user } = useAuth()
  const [period, setPeriod] = useState(tahunIni)
  const permintaan = useAsync(() => renewalService.list({ period }), [period])
  const [memproses, setMemproses] = useState<string | null>(null)

  const bolehMenjawab = user?.role === 'mc' || user?.role === 'admin'
  const bolehMeminta = user?.role === 'st' || user?.role === 'admin'

  const ringkas = useMemo(() => {
    const d = permintaan.data ?? []
    return {
      total: d.length,
      belum: d.filter((r) => r.answer === 'pending').length,
      akan: d.filter((r) => r.answer === 'will_renew').length,
    }
  }, [permintaan.data])

  const jawab = async (r: RenewalRequest, answer: RenewalAnswer) => {
    setMemproses(r.id)
    try {
      await renewalService.answer(r.id, answer)
      toast(`${r.memberName ?? r.memberId}: ${LABEL[answer]}.`)
      permintaan.reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan jawaban.', 'error')
    } finally {
      setMemproses(null)
    }
  }

  return (
    <div>
      <PageHeader
        title="Konfirmasi Renewal"
        description="ST menanyakan siapa yang akan memperpanjang; MC menjawab per member."
        action={bolehMeminta ? <TombolMinta period={period} onDone={permintaan.reload} /> : undefined}
      />

      <Card>
        <CardHeader
          title={`Periode ${period}`}
          subtitle={
            permintaan.data
              ? `${ringkas.total} member — ${ringkas.belum} belum dijawab, ${ringkas.akan} akan perpanjang`
              : undefined
          }
          action={
            <div className="w-28">
              <Input
                value={period}
                onChange={(e) => setPeriod(e.target.value.replace(/\D/g, '').slice(0, 4))}
                placeholder="2027"
                aria-label="Periode"
              />
            </div>
          }
        />
        <CardBody>
          {permintaan.loading ? (
            <LoadingState />
          ) : permintaan.error ? (
            <ErrorState message={permintaan.error} onRetry={permintaan.reload} />
          ) : !permintaan.data?.length ? (
            <EmptyState
              title="Belum ada permintaan konfirmasi"
              description={
                bolehMeminta
                  ? 'Tekan "Minta Konfirmasi" untuk menanyakan ke MC siapa saja yang akan memperpanjang.'
                  : 'ST belum meminta konfirmasi untuk periode ini.'
              }
            />
          ) : (
            <Table>
              <THead>
                <Tr>
                  <Th>Member</Th>
                  <Th>Chapter</Th>
                  <Th>Jatuh tempo</Th>
                  <Th>Jawaban</Th>
                  {bolehMenjawab && <Th className="text-right">Jawab</Th>}
                </Tr>
              </THead>
              <TBody>
                {permintaan.data.map((r) => (
                  <Tr key={r.id}>
                    <Td className="font-medium text-ink-900">{r.memberName ?? r.memberId}</Td>
                    <Td className="text-ink-600">{r.chapterName ?? r.chapterId}</Td>
                    <Td className="text-ink-600">
                      {r.renewalDate ? formatDate(r.renewalDate) : <span className="text-ink-400">—</span>}
                    </Td>
                    <Td>
                      <Badge tone={TONE[r.answer]}>{LABEL[r.answer]}</Badge>
                      {r.note && <div className="mt-1 text-xs text-ink-500">{r.note}</div>}
                    </Td>
                    {bolehMenjawab && (
                      <Td className="text-right">
                        <div className="inline-flex gap-1">
                          <TombolJawab
                            icon={CheckCircle2}
                            label="Akan"
                            tone="hover:text-emerald-600"
                            disabled={memproses === r.id}
                            onClick={() => jawab(r, 'will_renew')}
                          />
                          <TombolJawab
                            icon={XCircle}
                            label="Tidak"
                            tone="hover:text-red-600"
                            disabled={memproses === r.id}
                            onClick={() => jawab(r, 'will_not')}
                          />
                          <TombolJawab
                            icon={HelpCircle}
                            label="Belum pasti"
                            tone="hover:text-amber-600"
                            disabled={memproses === r.id}
                            onClick={() => jawab(r, 'unsure')}
                          />
                        </div>
                      </Td>
                    )}
                  </Tr>
                ))}
              </TBody>
            </Table>
          )}
        </CardBody>
      </Card>
    </div>
  )
}

function TombolJawab({
  icon: Icon,
  label,
  tone,
  disabled,
  onClick,
}: {
  icon: typeof CheckCircle2
  label: string
  tone: string
  disabled?: boolean
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title={label}
      className={`inline-flex items-center gap-1 rounded-lg border border-ink-200 px-2 py-1 text-xs text-ink-600 transition-colors disabled:opacity-40 ${tone}`}
    >
      <Icon className="h-3.5 w-3.5" />
      {label}
    </button>
  )
}

function TombolMinta({ period, onDone }: { period: string; onDone: () => void }) {
  const { toast } = useToast()
  const [mengirim, setMengirim] = useState(false)

  const minta = async () => {
    setMengirim(true)
    try {
      // Diambil dari daftar member yang jatuh tempo, bukan seluruh member:
      // menanyakan konfirmasi kepada orang yang keanggotaannya masih lama
      // membuat daftar tugas MC penuh hal yang belum perlu dijawab.
      const jatuhTempo = await memberService.list()
      const ids = jatuhTempo.filter((m) => m.status === 'active').map((m) => m.id)
      if (ids.length === 0) {
        toast('Tidak ada member aktif yang perlu dikonfirmasi.', 'error')
        return
      }
      const hasil = await renewalService.request(ids, period)
      // Membedakan "dibuat" dari "dilewati" secara eksplisit: menekan dua kali
      // harus terbaca "0 baru, 12 sudah ada", bukan "12 dibuat" yang membuat
      // orang mengira permintaan pertamanya hilang.
      toast(
        hasil.dibuat > 0
          ? `${hasil.dibuat} permintaan dibuat${hasil.dilewati ? `, ${hasil.dilewati} sudah ada` : ''}.`
          : `Semua ${hasil.dilewati} permintaan sudah ada sebelumnya.`,
      )
      onDone()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal meminta konfirmasi.', 'error')
    } finally {
      setMengirim(false)
    }
  }

  return (
    <Button onClick={minta} loading={mengirim}>
      <Send className="h-4 w-4" />
      Minta Konfirmasi
    </Button>
  )
}
