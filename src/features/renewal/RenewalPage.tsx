import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { CalendarCog, CheckCircle2, FilePlus2, FileText, FileUp, HelpCircle, Send, Trash2, Undo2, Upload, XCircle } from 'lucide-react'
import type { MemberWithChapter, RenewalAnswer, RenewalRequest, RenewalStage } from '@/types'
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
  Pagination,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, importService, memberService, periodService, renewalService, userService } from '@/services'
import type { HasilEmailMC } from '@/services/types'
import { useAuth } from '@/features/auth/AuthContext'
import { useCan } from '@/features/auth/usePermission'
import { TAHAP, TahapBadge } from './components/TahapBadge'
import { KelolaPeriode } from './components/KelolaPeriode'
import { periodeAwal } from '@/lib/periode'
import { formatDate } from '@/lib/format'
import { bolehDitagih } from '@/lib/status'
import { cn } from '@/lib/cn'
import { usePaginasi } from '@/hooks/usePaginasi'

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

const SEMUA_PERIODE = 'semua'

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

export function RenewalPage() {
  const { toast } = useToast()
  const { user } = useAuth()
  // Email ke MC menautkan ?periode=, supaya halaman membuka periode yang ditanyakan.
  const [params] = useSearchParams()
  const [period, setPeriod] = useState(params.get('periode') ?? '')
  // Email permintaan invoice ke finance menautkan ?periode=semua&tahap=invoice_requested.
  const [tahap, setTahap] = useState<RenewalStage | 'all'>((params.get('tahap') as RenewalStage | null) ?? 'all')
  // Daftar periode datang dari master data yang dikelola admin. Tanpa tautan,
  // halaman membuka periode yang memuat hari ini.
  const periode = useAsync(() => periodService.list(), [])
  const [kelolaPeriode, setKelolaPeriode] = useState(false)
  const bolehKelolaPeriode = useCan('settings:manage')
  useEffect(() => {
    if (!period && periode.data) setPeriod(periodeAwal(periode.data) ?? '')
  }, [period, periode.data])
  const periodeIni = periode.data?.find((p) => p.id === period)
  const [chapterFilter, setChapterFilter] = useState('all')
  // SEMUA_PERIODE membuka antrean lintas periode, misalnya permintaan invoice
  // yang menunggu finance.
  const permintaan = useAsync(
    () =>
      period ? renewalService.list(period === SEMUA_PERIODE ? {} : { period }) : Promise.resolve([]),
    [period],
  )
  const navigate = useNavigate()
  const bolehTagih = useCan('invoice:create')
  const bolehMintaInvoice = useCan('invoice:request')
  const bolehKirimBukti = useCan('payment:submit')
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
  const tersaring = useMemo(
    () =>
      (permintaan.data ?? []).filter(
        (r) => (chapterFilter === 'all' || r.chapterId === chapterFilter) && (tahap === 'all' || r.stage === tahap),
      ),
    [permintaan.data, chapterFilter, tahap],
  )
  const { potongan, paginasi } = usePaginasi(tersaring)

  const ringkas = useMemo(() => {
    const d = tersaring
    return {
      total: d.length,
      belum: d.filter((r) => r.answer === 'pending').length,
      akan: d.filter((r) => r.answer === 'will_renew').length,
      lunas: d.filter((r) => r.stage === 'paid' || r.stage === 'submitted').length,
    }
  }, [tersaring])

  // Langkah berikutnya setelah MC menjawab: yang Diterima dan belum ditagih
  // dibawa ke Buat Invoice dalam keadaan terpilih. Tanpa ini orangnya harus
  // pindah halaman dan mencari ulang nama yang baru saja ia lihat di sini.
  //
  // Yang sudah diminta ST ditagih lebih dulu: ST sedang menunggunya.
  const siapDitagih = useMemo(
    () =>
      [
        ...tersaring.filter((r) => r.stage === 'invoice_requested'),
        ...tersaring.filter((r) => r.stage === 'will_renew'),
      ].map((r) => r.memberId),
    [tersaring],
  )

  const [terpilih, setTerpilih] = useState<Set<string>>(new Set())
  const [massalSibuk, setMassalSibuk] = useState(false)

  // MC dan admin memilih yang BELUM dijawab, untuk dijawab sekaligus.
  // Menjawab ulang yang sudah dijawab bukan hal yang dilarang, tapi memilih
  // "semua" lalu menimpa jawaban yang sudah masuk adalah cara paling mudah
  // menghapus pekerjaan MC lain tanpa sadar.
  //
  // ST memilih yang sudah Diterima dan belum diminta invoicenya.
  const memilih = bolehMenjawab || bolehMintaInvoice
  const bisaDipilih = useMemo(
    () => tersaring.filter((r) => (bolehMenjawab ? r.answer === 'pending' : r.stage === 'will_renew')),
    [tersaring, bolehMenjawab],
  )
  const idDipilih = useMemo(() => new Set(bisaDipilih.map((r) => r.id)), [bisaDipilih])

  // Pilihan dibersihkan saat daftarnya berubah. Tanpa ini, id yang sudah tidak
  // tampil tetap ikut terkirim saat tombol massal ditekan.
  useEffect(() => {
    setTerpilih(new Set())
  }, [period, chapterFilter, tahap, permintaan.data])

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

  // Membatalkan mengikuti hak menulis (admin dan ST), sama seperti meminta.
  const batalkan = async (r: RenewalRequest) => {
    if (!window.confirm(`Batalkan permintaan konfirmasi ${r.memberName ?? r.memberId}?`)) return
    setMemproses(r.id)
    try {
      await renewalService.cancel(r.id)
      toast(`Permintaan ${r.memberName ?? r.memberId} dibatalkan.`)
      permintaan.reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal membatalkan permintaan.', 'error')
    } finally {
      setMemproses(null)
    }
  }

  // Langkah 3 alur renewal: ST meminta finance menerbitkan invoice.
  const mintaInvoice = async (ids: string[]) => {
    setMassalSibuk(true)
    try {
      const hasil = await renewalService.requestInvoice(ids)
      const surel = hasil.email?.belumDikonfigurasi
        ? ', email ke finance belum aktif di server'
        : hasil.email?.terkirim
          ? `, email terkirim ke ${hasil.email.terkirim} akun finance`
          : ''
      toast(
        hasil.diminta > 0
          ? `Invoice diminta untuk ${hasil.diminta} member${hasil.dilewati ? `, ${hasil.dilewati} dilewati` : ''}${surel}.`
          : 'Tidak ada yang diminta: member itu sudah diminta atau sudah punya invoice.',
        hasil.diminta > 0 ? 'success' : 'error',
      )
      setTerpilih(new Set())
      permintaan.reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal meminta invoice.', 'error')
    } finally {
      setMassalSibuk(false)
    }
  }

  const tarikPermintaanInvoice = async (r: RenewalRequest) => {
    setMemproses(r.id)
    try {
      await renewalService.cancelInvoiceRequest(r.id)
      toast(`Permintaan invoice ${r.memberName ?? r.memberId} ditarik.`)
      permintaan.reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menarik permintaan invoice.', 'error')
    } finally {
      setMemproses(null)
    }
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
      {bolehKelolaPeriode && (
        <KelolaPeriode
          open={kelolaPeriode}
          onClose={() => setKelolaPeriode(false)}
          periode={periode.data ?? []}
          onBerubah={(pilih) => {
            periode.reload()
            if (pilih) setPeriod(pilih)
          }}
        />
      )}
      <PageHeader
        title="Konfirmasi Renewal"
        description="Sistem menyusun daftar member yang jatuh tempo; ST menyetujui dan menanyakannya; MC menjawab per member."
        action={
          bolehMeminta && periodeIni ? (
            <TombolMinta
              period={period}
              namaPeriode={periodeIni.name}
              rentang={{ dari: periodeIni.startDate, sampai: periodeIni.endDate }}
              sudahDiminta={new Set((permintaan.data ?? []).map((r) => r.memberId))}
              lintasChapter={lintasChapter}
              chapterOptions={(chapters.data ?? []).map((c) => ({ id: c.id, displayName: c.displayName }))}
              namaChapter={(id) => namaChapterDari.get(id) ?? id}
              daftarMc={daftarMc.map((u) => ({
                id: u.id,
                nama: u.name,
                chapterId: u.chapterId,
                namaChapter: (u.chapterId && namaChapterDari.get(u.chapterId)) || null,
              }))}
              onDone={() => {
                permintaan.reload()
                periode.reload()
              }}
            />
          ) : undefined
        }
      />

      <Card>
        <CardHeader
          title={period === SEMUA_PERIODE ? 'Semua periode' : period ? `Periode ${periodeIni?.name ?? period}` : 'Periode'}
          subtitle={
            permintaan.data
              ? `${ringkas.total} member: ${ringkas.belum} belum dijawab, ${ringkas.akan} akan perpanjang, ${ringkas.lunas} lunas`
              : undefined
          }
          action={
            <div className="flex flex-wrap items-center gap-2">
              {bolehTagih && siapDitagih.length > 0 && (
                <Button
                  size="sm"
                  onClick={() => navigate(`/invoices/new?type=renewal&member=${siapDitagih.join(',')}`)}
                >
                  <FileText className="h-4 w-4" />
                  Buat Invoice ({siapDitagih.length})
                </Button>
              )}
              <Select
                value={period}
                onChange={(e) => setPeriod(e.target.value)}
                aria-label="Periode"
                className="w-52"
                disabled={!periode.data}
              >
                {/* Periode dari tautan yang sudah dihapus tetap tampil, supaya
                    dropdown tidak menunjuk periode lain dari yang dibuka. */}
                {period && period !== SEMUA_PERIODE && periode.data && !periodeIni && (
                  <option value={period}>{period}</option>
                )}
                {!period && <option value="">Belum ada periode</option>}
                <option value={SEMUA_PERIODE}>Semua periode</option>
                {(periode.data ?? []).map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </Select>
              {bolehKelolaPeriode && (
                <Button size="sm" variant="outline" onClick={() => setKelolaPeriode(true)} data-testid="kelola-periode">
                  <CalendarCog className="h-4 w-4" />
                  Kelola Periode
                </Button>
              )}
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
              <Select
                value={tahap}
                onChange={(e) => setTahap(e.target.value as RenewalStage | 'all')}
                aria-label="Status renewal"
                className="w-52"
              >
                <option value="all">Semua status</option>
                {TAHAP.map((t) => (
                  <option key={t.value} value={t.value}>
                    {t.label}
                  </option>
                ))}
              </Select>
            </div>
          }
        />
        {/* Bilah aksi massal muncul HANYA saat ada yang dipilih.
            Bilah yang selalu ada dengan tombol mati sepanjang waktu mengajarkan
            orang mengabaikannya, lalu tidak terlihat justru saat menyala. */}
        {memilih && terpilih.size > 0 && (
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-ink-100 bg-brand-50/40 px-5 py-3">
            <span className="text-sm font-medium text-ink-700">
              {terpilih.size} dipilih
            </span>
            <div className="flex flex-wrap items-center gap-2">
              {bolehMintaInvoice && !bolehMenjawab && (
                <Button size="sm" loading={massalSibuk} onClick={() => mintaInvoice([...terpilih])}>
                  <FilePlus2 className="h-4 w-4" />
                  Minta invoice ke finance
                </Button>
              )}
              {bolehMenjawab && (
              <>
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
              </>
              )}
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
          {periode.data && periode.data.length === 0 ? (
            <EmptyState
              title="Belum ada periode"
              description={
                bolehKelolaPeriode
                  ? 'Tambahkan periode lewat tombol "Kelola Periode" sebelum meminta konfirmasi.'
                  : 'Admin belum menambahkan periode konfirmasi renewal.'
              }
            />
          ) : permintaan.loading ? (
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
                  {memilih && (
                    <Th className="w-10">
                      <input
                        type="checkbox"
                        aria-label={bolehMenjawab ? 'Pilih semua yang belum dijawab' : 'Pilih semua yang siap diminta invoice'}
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
                  <Th>Jatuh tempo</Th>
                  <Th>Jawaban MC</Th>
                  <Th>Status renewal</Th>
                  {(bolehMenjawab || bolehTagih || bolehMintaInvoice) && <Th className="text-right">Tindakan</Th>}
                </Tr>
              </THead>
              <TBody>
                {potongan.map((r) => (
                  <Tr key={r.id}>
                    {memilih && (
                      <Td>
                        <input
                          type="checkbox"
                          aria-label={`Pilih ${r.memberName ?? r.memberId}`}
                          checked={terpilih.has(r.id)}
                          disabled={!idDipilih.has(r.id)}
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
                    <Td className="whitespace-nowrap">
                      <div className="font-medium text-ink-900">{r.memberName ?? r.memberId}</div>
                      <div className="text-xs text-ink-400">{r.chapterName ?? r.chapterId}</div>
                    </Td>
                    <Td className="whitespace-nowrap text-ink-600">
                      {r.renewalDate ? formatDate(r.renewalDate) : <span className="text-ink-400">—</span>}
                    </Td>
                    <Td className="whitespace-nowrap">
                      <Badge tone={TONE[r.answer]}>{LABEL[r.answer]}</Badge>
                      {r.note && <div className="mt-1 text-xs text-ink-500">{r.note}</div>}
                      <div className="mt-1 text-xs text-ink-400">
                        {r.assignedMc
                          ? // Nama dari server. Id akun tidak pernah ditampilkan:
                            // deretan huruf dan angka tidak memberi tahu siapa pun
                            // siapa yang harus menjawab.
                            (r.assignedMcName ?? namaMc.get(r.assignedMc) ?? 'MC tidak dikenal')
                          : // Tidak ditugaskan ke siapa pun BUKAN berarti tidak ada
                            // yang menanganinya: seluruh MC chapter itu melihatnya.
                            'Semua MC'}
                      </div>
                    </Td>
                    <Td className="whitespace-nowrap">
                      <TahapBadge tahap={r.stage} />
                      {r.invoiceId && r.invoiceNumber ? (
                        <Link
                          to={`/invoices/${r.invoiceId}`}
                          className="mt-1 block text-xs font-medium text-brand-600 hover:underline"
                        >
                          {r.invoiceNumber}
                        </Link>
                      ) : (
                        r.invoiceRequestedAt && (
                          <div className="mt-1 text-xs text-ink-500">
                            Diminta {formatDate(r.invoiceRequestedAt)}
                            {r.invoiceRequestedBy ? ` oleh ${r.invoiceRequestedBy}` : ''}
                          </div>
                        )
                      )}
                    </Td>
                    {(bolehMenjawab || bolehTagih || bolehMintaInvoice) && (
                      <Td className="text-right">
                        <div className="inline-flex flex-wrap justify-end gap-1">
                          {bolehMenjawab && (
                            <>
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
                            </>
                          )}
                          {bolehMintaInvoice && r.stage === 'will_renew' && (
                            <TombolJawab
                              icon={FilePlus2}
                              label="Minta invoice"
                              tone="hover:text-brand-600"
                              disabled={memproses === r.id || massalSibuk}
                              onClick={() => mintaInvoice([r.id])}
                            />
                          )}
                          {bolehMintaInvoice && r.stage === 'invoice_requested' && (
                            <TombolJawab
                              icon={Undo2}
                              label="Tarik permintaan"
                              tone="hover:text-red-600"
                              disabled={memproses === r.id}
                              onClick={() => tarikPermintaanInvoice(r)}
                            />
                          )}
                          {bolehKirimBukti && r.stage === 'invoiced' && r.invoiceId && (
                            <TombolJawab
                              icon={Upload}
                              label="Kirim bukti bayar"
                              tone="hover:text-brand-600"
                              onClick={() => navigate(`/invoices/${r.invoiceId}`)}
                            />
                          )}
                          {(bolehTagih || bolehMintaInvoice) && r.answer === 'pending' && (
                            <TombolJawab
                              icon={Trash2}
                              label="Batalkan"
                              tone="hover:text-red-600"
                              disabled={memproses === r.id}
                              onClick={() => batalkan(r)}
                            />
                          )}
                        </div>
                      </Td>
                    )}
                  </Tr>
                ))}
              </TBody>
            </Table>
          )}
        </CardBody>
        <Pagination {...paginasi} satuan="permintaan" />
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
      className={`inline-flex items-center gap-1 whitespace-nowrap rounded-lg border border-ink-200 px-1.5 py-1 text-xs text-ink-600 transition-colors disabled:opacity-40 ${tone}`}
    >
      <Icon className="h-3.5 w-3.5" />
      {label}
    </button>
  )
}

type Kelompok = 'jatuhTempo' | 'terlewat' | 'tanpaTanggal'

const LABEL_KELOMPOK: Record<Kelompok, string> = {
  jatuhTempo: 'Jatuh tempo di periode ini',
  terlewat: 'Sudah lewat jatuh tempo',
  tanpaTanggal: 'Tanpa tanggal jatuh tempo',
}

/**
 * Status pemberitahuan ke MC, untuk disebut di toast. Permintaan tetap tersimpan
 * saat email gagal, jadi ST harus tahu MC mana yang perlu dihubungi sendiri.
 */
function pesanEmail(email: HasilEmailMC | undefined, namaChapter: (id: string) => string): string | null {
  if (!email) return null
  const tanpaMc = email.chapterTanpaMc.map(namaChapter).join(', ')
  return (
    [
      email.belumDikonfigurasi ? 'email ke MC belum aktif di server' : null,
      email.terkirim > 0 ? `email terkirim ke ${email.terkirim} MC` : null,
      email.gagal > 0 ? `email gagal ke ${email.gagal} MC` : null,
      tanpaMc ? `${tanpaMc} belum punya akun MC` : null,
    ]
      .filter(Boolean)
      .join(', ') || null
  )
}

function TombolMinta({
  period,
  namaPeriode,
  rentang,
  sudahDiminta,
  daftarMc,
  lintasChapter,
  chapterOptions,
  namaChapter,
  onDone,
}: {
  period: string
  namaPeriode: string
  rentang: { dari: string; sampai: string }
  sudahDiminta: Set<string>
  daftarMc: { id: string; nama: string; chapterId: string | null; namaChapter: string | null }[]
  lintasChapter: boolean
  chapterOptions: { id: string; displayName: string }[]
  namaChapter: (id: string) => string
  onDone: () => void
}) {
  const { toast } = useToast()
  const [buka, setBuka] = useState(false)
  const [assignedMc, setAssignedMc] = useState('')
  const [chapterId, setChapterId] = useState('all')
  const [berkas, setBerkas] = useState<File | null>(null)
  const [mengirim, setMengirim] = useState(false)
  const [anggota, setAnggota] = useState<MemberWithChapter[] | null>(null)
  const [terpilih, setTerpilih] = useState<Set<string>>(new Set())

  // Daftar member dimuat setiap kali modal dibuka, supaya jatuh tempo yang
  // baru saja diimpor atau dilunasi langsung terbaca.
  useEffect(() => {
    if (!buka) return
    setAnggota(null)
    memberService.list().then(setAnggota).catch(() => setAnggota([]))
  }, [buka])

  /**
   * Daftar jatuh tempo disusun sistem; ST yang menyetujui.
   *
   * Calon: member yang boleh ditagih renewal (active, inactive, new_member) di
   * chapter terpilih. Yang jatuh temponya masuk periode ini dicentang otomatis.
   * Yang sudah lewat jatuh tempo dan yang belum punya tanggal ikut ditampilkan
   * tanpa centang, karena keduanya butuh keputusan orang: bisa jadi sudah
   * keluar, bisa jadi tanggalnya belum diimpor. Member yang jatuh temponya
   * sesudah periode ini tidak ditampilkan; belum waktunya ditanyakan.
   */
  const kandidat = useMemo(() => {
    const out: Record<Kelompok, MemberWithChapter[]> = { jatuhTempo: [], terlewat: [], tanpaTanggal: [] }
    for (const m of anggota ?? []) {
      if (!bolehDitagih(m.status, 'renewal')) continue
      if (chapterId !== 'all' && m.chapterId !== chapterId) continue
      const tgl = m.renewalDate?.slice(0, 10)
      if (!tgl) out.tanpaTanggal.push(m)
      else if (tgl >= rentang.dari && tgl <= rentang.sampai) out.jatuhTempo.push(m)
      else if (tgl < rentang.dari) out.terlewat.push(m)
    }
    for (const k of Object.keys(out) as Kelompok[]) {
      out[k].sort((a, b) => (a.renewalDate ?? '').localeCompare(b.renewalDate ?? '') || a.name.localeCompare(b.name))
    }
    return out
  }, [anggota, chapterId, rentang.dari, rentang.sampai])

  useEffect(() => {
    setTerpilih(new Set(kandidat.jatuhTempo.filter((m) => !sudahDiminta.has(m.id)).map((m) => m.id)))
    // Bergantung pada kandidat saja: sudahDiminta dibangun ulang tiap render
    // induk, dan memasukkannya akan menimpa centang ST di setiap render.
  }, [kandidat])

  const alihkan = (id: string) =>
    setTerpilih((lama) => {
      const baru = new Set(lama)
      if (baru.has(id)) baru.delete(id)
      else baru.add(id)
      return baru
    })

  /**
   * Member yang ditanyakan datang dari salah satu dua sumber.
   *
   * Tanpa berkas: seluruh member aktif di chapter terpilih. Dengan berkas:
   * nama-nama di laporan Membership Dues BNI Connect (format yang sama dengan
   * yang diterima halaman Impor), dicocokkan ke member chapter itu lewat
   * pratinjau impor. Pratinjau tidak menulis apa pun; ia hanya meminjam
   * pencocokan namanya, supaya ST yang sudah memegang laporan jatuh tempo dari
   * BNI Connect tidak perlu memilih nama satu per satu.
   */
  const idDariBerkas = async (f: File): Promise<{ ids: string[]; takDikenal: string[] }> => {
    const hasil = await importService.preview('members', f, { chapter: chapterId })
    const ids: string[] = []
    const takDikenal: string[] = []
    for (const b of hasil.baris) {
      // "baru" berarti namanya tidak ada di chapter ini; id-nya hanya usulan
      // dan belum ada barisnya, jadi tidak bisa ditanyai.
      if (b.tindakan === 'diperbarui' || b.tindakan === 'sama') ids.push(b.id)
      else takDikenal.push(b.nama)
    }
    return { ids, takDikenal }
  }

  const minta = async () => {
    if (berkas && chapterId === 'all') {
      toast('Pilih chapternya dulu: laporan Business Network International Connect tidak memuat kolom chapter.', 'error')
      return
    }
    setMengirim(true)
    try {
      let ids: string[]
      let takDikenal: string[] = []
      if (berkas) {
        ;({ ids, takDikenal } = await idDariBerkas(berkas))
      } else {
        // Hanya yang dicentang ST dari daftar jatuh tempo.
        ids = [...terpilih]
      }
      if (ids.length === 0) {
        toast(
          berkas
            ? 'Tidak ada nama di laporan yang cocok dengan member chapter itu.'
            : 'Belum ada member yang dicentang.',
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
        takDikenal.length > 0 ? `${takDikenal.length} nama di laporan tidak dikenal` : null,
        pesanEmail(hasil.email, namaChapter),
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

      <Modal open={buka} onClose={() => setBuka(false)} title={`Minta konfirmasi periode ${namaPeriode}`}>
        <div className="space-y-4">
          <Field
            label="Chapter"
            hint="Menentukan member siapa saja yang ditanyakan. Dengan Semua chapter, tiap MC menerima email untuk chapternya sendiri."
          >
            <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)} aria-label="Filter chapter">
              <option value="all">Semua chapter</option>
              {chapterOptions.map((c) => (
                <option key={c.id} value={c.id}>{c.displayName}</option>
              ))}
            </Select>
          </Field>

          <Field
            label="Dari laporan Business Network International Connect (opsional)"
            hint="Laporan Membership Dues (.xls) atau template jatuh tempo dari halaman Impor. Bila diisi, nama di laporan yang ditanyakan, menggantikan daftar di bawah."
          >
            <label className="flex cursor-pointer items-center gap-3 rounded-xl border border-dashed border-ink-300 px-4 py-3 text-sm transition-colors hover:bg-ink-50">
              <FileUp className="h-4 w-4 shrink-0 text-brand-500" />
              <span className="min-w-0 flex-1 truncate text-ink-700">
                {berkas ? berkas.name : 'Pilih berkas…'}
              </span>
              {berkas && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    setBerkas(null)
                  }}
                  className="text-xs text-ink-500 hover:text-ink-700"
                >
                  Hapus
                </button>
              )}
              <input
                type="file"
                accept=".xls,.xlsx,.csv"
                className="hidden"
                onChange={(e) => setBerkas(e.target.files?.[0] ?? null)}
              />
            </label>
          </Field>

          {!berkas && (
            <div>
              <div className="mb-2 flex items-center justify-between gap-2">
                <span className="text-sm font-medium text-ink-800">Member yang ditanyakan</span>
                <span className="text-xs text-ink-500">{terpilih.size} dicentang</span>
              </div>
              {anggota === null ? (
                <p className="rounded-xl border border-ink-200 p-4 text-sm text-ink-500">Menyusun daftar jatuh tempo…</p>
              ) : kandidat.jatuhTempo.length + kandidat.terlewat.length + kandidat.tanpaTanggal.length === 0 ? (
                <p className="rounded-xl border border-ink-200 p-4 text-sm text-ink-500">
                  Tidak ada member yang jatuh tempo di {namaPeriode} untuk chapter ini.
                </p>
              ) : (
                <div className="max-h-72 space-y-3 overflow-y-auto rounded-xl border border-ink-200 p-3">
                  {(Object.keys(LABEL_KELOMPOK) as Kelompok[])
                    .filter((k) => kandidat[k].length > 0)
                    .map((k) => (
                      <div key={k}>
                        <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-ink-400">
                          {LABEL_KELOMPOK[k]} ({kandidat[k].length})
                        </div>
                        {kandidat[k].map((m) => {
                          const sudah = sudahDiminta.has(m.id)
                          return (
                            <label
                              key={m.id}
                              className={cn(
                                'flex items-center gap-3 rounded-lg px-2 py-1.5 text-sm',
                                sudah ? 'text-ink-400' : 'cursor-pointer hover:bg-ink-50',
                              )}
                            >
                              <input
                                type="checkbox"
                                disabled={sudah}
                                checked={!sudah && terpilih.has(m.id)}
                                onChange={() => alihkan(m.id)}
                                className="h-4 w-4 accent-bni-red"
                              />
                              <span className="min-w-0 flex-1 truncate">
                                {m.name}
                                <span className="text-ink-400"> · {m.chapter?.displayName ?? m.chapterId}</span>
                              </span>
                              <span className="shrink-0 text-xs text-ink-500">
                                {sudah ? 'sudah diminta' : m.renewalDate ? formatDate(m.renewalDate) : '—'}
                              </span>
                            </label>
                          )
                        })}
                      </div>
                    ))}
                </div>
              )}
            </div>
          )}

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
