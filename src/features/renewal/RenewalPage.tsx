import { useEffect, useMemo, useState } from 'react'
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
  will_renew: 'Diterima',
  will_not: 'Tolak',
  unsure: 'Belum pasti',
}

const TONE: Record<RenewalAnswer, 'gray' | 'green' | 'red' | 'amber'> = {
  pending: 'gray',
  will_renew: 'green',
  will_not: 'red',
  unsure: 'amber',
}

/**
 * Periode konfirmasi dihitung per ENAM BULAN, bukan per tahun.
 *
 * Keanggotaan berakhir sepanjang tahun, tidak menumpuk di Januari. Satu
 * periode setahun berarti MC menerima seluruh pertanyaan sekaligus dan
 * menjawabnya untuk member yang jatuh temponya masih sepuluh bulan lagi —
 * jawaban yang sudah usang saat tagihannya benar-benar terbit.
 *
 * Bentuknya "2027-S1" / "2027-S2". Periode lama yang berupa tahun saja tetap
 * terbaca apa adanya: nilainya cuma string pembeda, jadi baris lama tidak
 * perlu dipindahkan ke bentuk baru.
 */
function periodeSemester(d: Date): string {
  return `${d.getFullYear()}-S${d.getMonth() < 6 ? 1 : 2}`
}

/** Enam pilihan ke depan, dimulai dari semester berjalan. */
function pilihanPeriode(dari: Date, jumlah = 6): string[] {
  const out: string[] = []
  let tahun = dari.getFullYear()
  let sem = dari.getMonth() < 6 ? 1 : 2
  for (let i = 0; i < jumlah; i++) {
    out.push(`${tahun}-S${sem}`)
    if (sem === 2) { tahun += 1; sem = 1 } else { sem = 2 }
  }
  return out
}

const labelPeriode = (p: string) =>
  p.endsWith('-S1') ? `${p.slice(0, 4)} · Jan–Jun`
  : p.endsWith('-S2') ? `${p.slice(0, 4)} · Jul–Des`
  : p

const periodeAwal = periodeSemester(new Date())

export function RenewalPage() {
  const { toast } = useToast()
  const { user } = useAuth()
  const [period, setPeriod] = useState(periodeAwal)
  const [chapterFilter, setChapterFilter] = useState('all')
  const permintaan = useAsync(() => renewalService.list({ period }), [period])

  /**
   * Periode yang BENAR-BENAR ADA datanya, ditemukan sekali di awal.
   *
   * Tanpa ini, mengganti bentuk periode dari tahun ke semester membuat
   * permintaan lama tidak terjangkau: dropdown hanya menawarkan semester,
   * sedangkan barisnya tersimpan di bawah "2027". Datanya tidak hilang — hanya
   * tidak ada lagi cara membukanya, yang bagi yang memakainya sama saja.
   */
  const periodeAda = useAsync(
    () => renewalService.list({}).then((d) => [...new Set(d.map((r) => r.period))].sort()),
    [],
  )
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

  // Daftar yang TAMPIL, sesudah filter chapter. Ringkasan dihitung dari yang
  // sama — ringkasan yang menghitung seluruh data sementara tabelnya tersaring
  // membuat dua angka di layar yang sama saling membantah.
  const tersaring = useMemo(() => {
    const d = permintaan.data ?? []
    return chapterFilter === 'all' ? d : d.filter((r) => r.chapterId === chapterFilter)
  }, [permintaan.data, chapterFilter])

  const ringkas = useMemo(() => {
    const d = tersaring
    return {
      total: d.length,
      belum: d.filter((r) => r.answer === 'pending').length,
      akan: d.filter((r) => r.answer === 'will_renew').length,
    }
  }, [tersaring])

  const [terpilih, setTerpilih] = useState<Set<string>>(new Set())
  const [massalSibuk, setMassalSibuk] = useState(false)

  // Hanya yang BELUM dijawab yang bisa dipilih.
  //
  // Menjawab ulang yang sudah dijawab bukan hal yang dilarang — tapi memilih
  // "semua" lalu menimpa jawaban yang sudah masuk adalah cara paling mudah
  // menghapus pekerjaan MC lain tanpa sadar.
  const bisaDipilih = useMemo(
    () => tersaring.filter((r) => r.answer === 'pending'),
    [tersaring],
  )

  // Pilihan dibersihkan saat daftarnya berubah. Tanpa ini, id yang sudah tidak
  // tampil tetap ikut terkirim saat tombol massal ditekan.
  useEffect(() => {
    setTerpilih(new Set())
  }, [period, chapterFilter, permintaan.data])

  const jawabMassal = async (answer: RenewalAnswer) => {
    const daftar = bisaDipilih.filter((r) => terpilih.has(r.id))
    if (daftar.length === 0) return
    setMassalSibuk(true)
    let berhasil = 0
    const gagal: string[] = []
    // Satu per satu, dan SATU KEGAGALAN TIDAK MEMBATALKAN SISANYA. Melempar di
    // tengah loop meninggalkan sebagian terjawab dan sebagian tidak, tanpa ada
    // yang tahu batasnya di mana.
    for (const r of daftar) {
      try {
        await renewalService.answer(r.id, answer)
        berhasil += 1
      } catch {
        gagal.push(r.memberName ?? r.memberId)
      }
    }
    setMassalSibuk(false)
    setTerpilih(new Set())
    permintaan.reload()
    toast(
      gagal.length === 0
        ? `${berhasil} konfirmasi diperbarui.`
        : `${berhasil} berhasil, ${gagal.length} gagal: ${gagal.slice(0, 3).join(', ')}${gagal.length > 3 ? '…' : ''}`,
      gagal.length === 0 ? 'success' : 'error',
    )
  }

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
              chapterOptions={(chapters.data ?? []).map((c) => ({ id: c.id, displayName: c.displayName }))}
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
          title={`Periode ${labelPeriode(period)}`}
          subtitle={
            permintaan.data
              ? `${ringkas.total} member — ${ringkas.belum} belum dijawab, ${ringkas.akan} akan perpanjang`
              : undefined
          }
          action={
            <div className="flex flex-wrap items-center gap-2">
              {/* Dropdown, bukan ketik bebas: bentuk periodenya kini "2027-S1",
                  dan kolom bebas mengundang orang mengetik "2027" lalu melihat
                  daftar kosong tanpa tahu kenapa. */}
              <Select
                value={period}
                onChange={(e) => setPeriod(e.target.value)}
                aria-label="Periode"
                className="w-44"
              >
                {/* Periode yang sedang dilihat selalu ada di daftar, walau ia
                    periode lama berbentuk tahun — kalau tidak, memilihnya dari
                    tautan akan menampilkan dropdown yang menunjuk hal lain. */}
                {[...new Set([period, ...pilihanPeriode(new Date()), ...(periodeAda.data ?? [])])]
                  .sort()
                  .map((p) => (
                    <option key={p} value={p}>{labelPeriode(p)}</option>
                  ))}
              </Select>
              <Select
                value={chapterFilter}
                onChange={(e) => setChapterFilter(e.target.value)}
                aria-label="Chapter"
                className="w-44"
              >
                <option value="all">Semua Chapter</option>
                {(chapters.data ?? []).map((c) => (
                  <option key={c.id} value={c.id}>{c.displayName}</option>
                ))}
              </Select>
            </div>
          }
        />
        {/* Bilah aksi massal muncul HANYA saat ada yang dipilih.
            Bilah yang selalu ada dengan tombol mati sepanjang waktu mengajarkan
            orang mengabaikannya, lalu tidak terlihat justru saat menyala. */}
        {bolehMenjawab && terpilih.size > 0 && (
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-ink-100 bg-brand-50/40 px-5 py-3">
            <span className="text-sm font-medium text-ink-700">
              {terpilih.size} dipilih
            </span>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                size="sm"
                variant="outline"
                loading={massalSibuk}
                onClick={() => jawabMassal('will_renew')}
              >
                <CheckCircle2 className="h-4 w-4" />
                Diterima
              </Button>
              <Button
                size="sm"
                variant="outline"
                loading={massalSibuk}
                onClick={() => jawabMassal('will_not')}
              >
                <XCircle className="h-4 w-4" />
                Tolak
              </Button>
              <button
                type="button"
                onClick={() => setTerpilih(new Set())}
                className="text-sm text-ink-500 hover:text-ink-700"
              >
                Batal
              </button>
            </div>
          </div>
        )}

        <CardBody>
          {permintaan.loading ? (
            <LoadingState />
          ) : permintaan.error ? (
            <ErrorState message={permintaan.error} onRetry={permintaan.reload} />
          ) : !tersaring.length ? (
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
                  {bolehMenjawab && (
                    <Th className="w-10">
                      <input
                        type="checkbox"
                        aria-label="Pilih semua yang belum dijawab"
                        checked={bisaDipilih.length > 0 && terpilih.size === bisaDipilih.length}
                        onChange={(e) =>
                          setTerpilih(e.target.checked ? new Set(bisaDipilih.map((r) => r.id)) : new Set())
                        }
                        disabled={bisaDipilih.length === 0}
                        className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400"
                      />
                    </Th>
                  )}
                  <Th>Member</Th>
                  <Th>Chapter</Th>
                  <Th>Jatuh tempo</Th>
                  <Th>Ditugaskan ke</Th>
                  <Th>Jawaban</Th>
                  {bolehMenjawab && <Th className="text-right">Tindakan</Th>}
                </Tr>
              </THead>
              <TBody>
                {tersaring.map((r) => (
                  <Tr key={r.id}>
                    {bolehMenjawab && (
                      <Td>
                        <input
                          type="checkbox"
                          aria-label={`Pilih ${r.memberName ?? r.memberId}`}
                          checked={terpilih.has(r.id)}
                          disabled={r.answer !== 'pending'}
                          onChange={() =>
                            setTerpilih((lama) => {
                              const baru = new Set(lama)
                              if (baru.has(r.id)) baru.delete(r.id)
                              else baru.add(r.id)
                              return baru
                            })
                          }
                          className="h-4 w-4 cursor-pointer rounded border-ink-300 text-brand-500 focus:ring-brand-400 disabled:opacity-40"
                        />
                      </Td>
                    )}
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
                            label="Diterima"
                            tone="hover:text-emerald-600"
                            disabled={memproses === r.id}
                            onClick={() => jawab(r, 'will_renew')}
                          />
                          <TombolJawab
                            icon={XCircle}
                            label="Tolak"
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
  chapterOptions,
  onDone,
}: {
  period: string
  daftarMc: { id: string; nama: string; chapterId: string | null; namaChapter: string | null }[]
  lintasChapter: boolean
  chapterOptions: { id: string; displayName: string }[]
  onDone: () => void
}) {
  const { toast } = useToast()
  const [buka, setBuka] = useState(false)
  const [assignedMc, setAssignedMc] = useState('')
  const [chapterId, setChapterId] = useState('all')
  const [mengirim, setMengirim] = useState(false)

  const minta = async () => {
    setMengirim(true)
    try {
      // Diambil dari daftar member yang jatuh tempo, bukan seluruh member:
      // menanyakan konfirmasi kepada orang yang keanggotaannya masih lama
      // membuat daftar tugas MC penuh hal yang belum perlu dijawab.
      const semua = await memberService.list()
      const aktif = semua.filter((m) => m.status === 'active')
      // Disaring per chapter bila dipilih. MOM: "Konfirmasi Renewal ada yang
      // kirim by chapter" — ST yang menangani satu chapter tidak seharusnya
      // membuat tugas untuk seluruh MC di chapter lain hanya karena menekan
      // satu tombol.
      const relevan = chapterId === 'all' ? aktif : aktif.filter((m) => m.chapterId === chapterId)
      const ids = relevan.map((m) => m.id)
      if (ids.length === 0) {
        toast(
          chapterId === 'all'
            ? 'Tidak ada member aktif yang perlu dikonfirmasi.'
            : 'Tidak ada member aktif di chapter itu.',
          'error',
        )
        return
      }
      const hasil = await renewalService.request(ids, period, assignedMc || null)
      // Membedakan "dibuat" dari "dilewati" secara eksplisit: menekan dua kali
      // harus terbaca "0 baru, 12 sudah ada", bukan "12 dibuat" yang membuat
      // orang mengira permintaan pertamanya hilang.
      //
      // Tamu disebut TERPISAH, dengan alasannya. Menyatukannya ke "sudah ada"
      // membuat orang mengira pekerjaannya beres, padahal ada nama yang tidak
      // menerima apa pun — dan mungkin memang statusnya yang belum diperbarui
      // setelah ia mendaftar.
      const bagian = [
        hasil.dibuat > 0 ? `${hasil.dibuat} permintaan dibuat` : null,
        hasil.dilewati > 0 ? `${hasil.dilewati} sudah ada` : null,
        hasil.visitor > 0
          ? `${hasil.visitor} dilewati karena masih berstatus visitor`
          : null,
      ].filter(Boolean)
      toast(
        bagian.length > 0
          ? `${bagian.join(', ')}.`
          : 'Tidak ada permintaan yang dibuat.',
        hasil.dibuat === 0 && hasil.visitor > 0 ? 'error' : undefined,
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

      <Modal open={buka} onClose={() => setBuka(false)} title={`Minta konfirmasi periode ${labelPeriode(period)}`}>
        <div className="space-y-4">
          <Field
            label="Chapter"
            hint="Menentukan member siapa saja yang ditanyakan."
          >
            <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)}>
              <option value="all">Semua chapter</option>
              {chapterOptions.map((c) => (
                <option key={c.id} value={c.id}>{c.displayName}</option>
              ))}
            </Select>
          </Field>

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
