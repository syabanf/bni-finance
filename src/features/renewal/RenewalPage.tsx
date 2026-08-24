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
  Field,
  Modal,
  PageHeader,
  Select,
  TBody,
  Table,
  Td,
  THead,
  Th,
  Tr,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, memberService, renewalService, userService } from '@/services'
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
  // Daftar akun hanya bisa dibaca admin. Untuk ST dan MC panggilannya gagal,
  // dan itu tidak boleh menggagalkan halaman — kolom "Ditugaskan ke" cukup
  // menampilkan idnya. Karena itu galatnya ditelan, bukan diteruskan.
  const akun = useAsync(() => userService.list().catch(() => []), [])
  const [memproses, setMemproses] = useState<string | null>(null)

  const bolehMenjawab = user?.role === 'mc' || user?.role === 'admin'
  const bolehMeminta = user?.role === 'st' || user?.role === 'admin'

  const chapters = useAsync(() => chapterService.list().catch(() => []), [])
  const namaChapterDari = useMemo(() => {
    const map = new Map<string, string>()
    for (const c of chapters.data ?? []) map.set(c.id, c.displayName)
    return map
  }, [chapters.data])

  const namaMc = useMemo(() => {
    const map = new Map<string, string>()
    for (const u of akun.data ?? []) map.set(u.id, u.name)
    return map
  }, [akun.data])

  // MC disaring ke chapter PEMANGGIL bila ia berlingkup chapter.
  //
  // Tanpa ini, menugaskan MC lintas chapter mungkin terjadi — dan terbukti
  // terjadi saat diuji: MC BNI Garuda ter-tag pada permintaan milik BNI
  // Nusantara dan BNI Bhinneka. Tidak ada yang gagal, tapi orang yang ditugaskan
  // tidak akan pernah melihat permintaannya, karena daftarnya sendiri dibatasi
  // chapter di server.
  const daftarMc = useMemo(() => {
    const mc = (akun.data ?? []).filter((u) => u.role === 'mc')
    if (user?.chapterId) return mc.filter((u) => u.chapterId === user.chapterId)
    return mc
  }, [akun.data, user?.chapterId])

  // Admin bisa meminta lintas chapter sekaligus, dan satu MC hanya masuk akal
  // untuk satu chapter. Dinyatakan di layar alih-alih dilarang: admin mungkin
  // memang sedang menangani satu chapter saja.
  const lintasChapter = !user?.chapterId

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
        action={
          bolehMeminta ? (
            <TombolMinta
              period={period}
              lintasChapter={lintasChapter}
              daftarMc={daftarMc.map((u) => ({
                id: u.id,
                nama: u.name,
                chapterId: u.chapterId,
                namaChapter: (u.chapterId && namaChapterDari.get(u.chapterId)) || null,
              }))}
              onDone={permintaan.reload}
            />
          ) : undefined
        }
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
                  <Th>Ditugaskan ke</Th>
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
                    <Td className="text-ink-600">
                      {r.assignedMc ? (
                        namaMc.get(r.assignedMc) ?? r.assignedMc
                      ) : (
                        // Bukan sel kosong: tidak ditugaskan ke siapa pun BUKAN
                        // berarti tidak ada yang menanganinya — permintaannya
                        // terlihat oleh seluruh MC chapter itu.
                        <span className="text-ink-400">Semua MC</span>
                      )}
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

function TombolMinta({
  period,
  daftarMc,
  lintasChapter,
  onDone,
}: {
  period: string
  daftarMc: { id: string; nama: string; chapterId: string | null; namaChapter: string | null }[]
  lintasChapter: boolean
  onDone: () => void
}) {
  const { toast } = useToast()
  const [buka, setBuka] = useState(false)
  const [assignedMc, setAssignedMc] = useState('')
  const [mengirim, setMengirim] = useState(false)

  const minta = async () => {
    setMengirim(true)
    try {
      // Diambil dari daftar member yang jatuh tempo, bukan seluruh member:
      // menanyakan konfirmasi kepada orang yang keanggotaannya masih lama
      // membuat daftar tugas MC penuh hal yang belum perlu dijawab.
      const semua = await memberService.list()
      const ids = semua.filter((m) => m.status === 'active').map((m) => m.id)
      if (ids.length === 0) {
        toast('Tidak ada member aktif yang perlu dikonfirmasi.', 'error')
        return
      }
      const hasil = await renewalService.request(ids, period, assignedMc || null)
      // Membedakan "dibuat" dari "dilewati" secara eksplisit: menekan dua kali
      // harus terbaca "0 baru, 12 sudah ada", bukan "12 dibuat" yang membuat
      // orang mengira permintaan pertamanya hilang.
      toast(
        hasil.dibuat > 0
          ? `${hasil.dibuat} permintaan dibuat${hasil.dilewati ? `, ${hasil.dilewati} sudah ada` : ''}.`
          : `Semua ${hasil.dilewati} permintaan sudah ada sebelumnya.`,
      )
      setBuka(false)
      onDone()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal meminta konfirmasi.', 'error')
    } finally {
      setMengirim(false)
    }
  }

  return (
    <>
      <Button onClick={() => setBuka(true)}>
        <Send className="h-4 w-4" />
        Minta Konfirmasi
      </Button>

      <Modal open={buka} onClose={() => setBuka(false)} title={`Minta konfirmasi periode ${period}`}>
        <div className="space-y-4">
          <Field
            label="Tugaskan ke MC"
            hint="Boleh dikosongkan — permintaan tetap terlihat oleh seluruh MC di chapter itu."
          >
            <Select value={assignedMc} onChange={(e) => setAssignedMc(e.target.value)}>
              {/* Pilihan pertama bukan placeholder kosong melainkan pilihan yang
                  SAH dan dinamai, supaya orang tahu mengosongkannya bukan
                  kelalaian melainkan keputusan. */}
              <option value="">Semua MC di chapter</option>
              {daftarMc.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.nama}
                  {m.namaChapter ? ` — ${m.namaChapter}` : ''}
                </option>
              ))}
            </Select>
          </Field>

          {lintasChapter && assignedMc && (
            <p className="rounded-lg bg-amber-50 p-3 text-xs leading-relaxed text-amber-800">
              Permintaan ini mencakup member dari BEBERAPA chapter, sedangkan satu MC hanya
              menangani chapternya sendiri. MC yang dipilih tidak akan melihat permintaan milik
              chapter lain — kosongkan pilihannya bila ingin setiap MC melihat chapternya
              masing-masing.
            </p>
          )}

          {daftarMc.length === 0 && (
            <p className="rounded-lg bg-ink-50 p-3 text-xs leading-relaxed text-ink-600">
              Belum ada akun MC yang bisa ditugaskan. Permintaan tetap bisa dibuat — ia akan
              terlihat oleh MC mana pun yang ditambahkan kemudian.
            </p>
          )}

          <div className="flex justify-end gap-2 pt-1">
            <Button variant="outline" onClick={() => setBuka(false)}>
              Batal
            </Button>
            <Button onClick={minta} loading={mengirim}>
              <Send className="h-4 w-4" />
              Kirim permintaan
            </Button>
          </div>
        </div>
      </Modal>
    </>
  )
}
