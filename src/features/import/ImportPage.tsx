import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { AlertTriangle, ArrowLeft, ArrowRight, Check, CheckCircle2, Download, FileUp, Upload, X } from 'lucide-react'
import type { Chapter, ImportBaris, ImportHasil, MemberStatus, MemberWithChapter } from '@/types'
import {
  Badge,
  Button,
  Card,
  CardBody,
  CardHeader,
  Field,
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
import { chapterService, importService, memberService } from '@/services'
import type { ImportOpsi } from '@/services/types'
import { downloadXlsx } from '@/lib/xlsx'
import { cn } from '@/lib/cn'

/**
 * Impor data sebagai wizard.
 *
 * Member satu chapter masuk lewat dua laporan BNI Connect, berurutan:
 *
 *   1. Chapter Roster   data lama di BNI: nama, perusahaan, bidang, telepon.
 *                       Satu-satunya langkah yang membuat member baru.
 *   2. Membership Dues  urutan jatuh tempo: mengisi tanggal perpanjangan
 *                       member yang sudah ada.
 *
 * Urutannya menentukan. Laporan Dues mencocokkan NAMA ke member di chapter
 * tujuan, jadi Roster harus sudah ditulis sebelum Dues ditinjau; wizard
 * menahan langkah 2 sampai langkah 1 diterapkan atau dilewati. Dengan begitu
 * pratinjau langkah 2 dihitung terhadap data yang sudah memuat hasil langkah 1.
 *
 * Setiap langkah tetap dua tahap, Tinjau lalu Terapkan, dan tombol Terapkan
 * baru ada setelah pratinjau. Angka pratinjau dihitung kode server yang sama
 * dengan yang menulis, jadi tidak mungkin berbeda dari yang akhirnya terjadi.
 *
 * Dua jalan masuk lain memakai kerangka yang sama dengan satu langkah: daftar
 * visitor (tombol "Impor Visitor" di halaman Member) dan daftar chapter.
 */

const KOLOM_CHAPTER = ['id', 'name', 'display_name', 'area_name', 'city_name']

/**
 * Judul kolom laporan "Membership Dues" BNI Connect, persis seperti ekspornya,
 * supaya template yang diisi tangan dan laporan asli menempuh jalur baca yang
 * satu.
 */
const KOLOM_JATUH_TEMPO = ['Member Name', 'Industry', 'Type', 'Membership Status', 'Due Date', 'Start Date']

/** Nilai "Membership Status" yang dipahami importer; status lain dibiarkan kosong. */
const STATUS_BNI: Partial<Record<MemberStatus, string>> = { active: 'Active', inactive: 'Dropped' }

const TONE: Record<ImportBaris['tindakan'], 'green' | 'amber' | 'gray' | 'red'> = {
  baru: 'green',
  diperbarui: 'amber',
  sama: 'gray',
  ditolak: 'red',
}

type KunciLangkah = 'roster' | 'dues' | 'visitor' | 'chapter'

interface Langkah {
  key: KunciLangkah
  judul: string
  berkas: string
  keterangan: string
  template?: 'jatuhTempo' | 'chapter'
}

const LANGKAH_MEMBER: Langkah[] = [
  {
    key: 'roster',
    judul: 'Data lama di BNI',
    berkas: 'Chapter_Roster_Report.xls',
    keterangan:
      'Laporan Chapter Roster dari BNI Connect: nama, perusahaan, bidang usaha, dan telepon yang tercatat di BNI. Nama yang belum ada di chapter dibuat sebagai member baru. Laporan Membership Length juga diterima di sini.',
  },
  {
    key: 'dues',
    judul: 'Urutan jatuh tempo',
    berkas: 'Chapter_Membership_Dues_Report.xls',
    keterangan:
      'Laporan Membership Dues dari BNI Connect, atau template jatuh tempo di bawah. Mengisi tanggal perpanjangan member yang namanya sudah ada di chapter.',
    template: 'jatuhTempo',
  },
]

const LANGKAH_VISITOR: Langkah[] = [
  {
    key: 'visitor',
    judul: 'Daftar visitor',
    berkas: 'CSV atau XLSX',
    keterangan: 'Daftar hadir atau daftar tamu. Baris tanpa kolom status disimpan sebagai visitor.',
  },
]

const LANGKAH_CHAPTER: Langkah[] = [
  {
    key: 'chapter',
    judul: 'Daftar chapter',
    berkas: 'CSV atau XLSX',
    keterangan: 'Template chapter di bawah sudah terisi chapter yang ada; tinggal lengkapi kolom kosongnya.',
    template: 'chapter',
  },
]

/** Hasil langkah yang sudah lewat; null berarti dilewati. */
type Riwayat = Partial<Record<KunciLangkah, ImportHasil | null>>

export function ImportPage() {
  const { toast } = useToast()
  const navigate = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null)
  const [params, setParams] = useSearchParams()

  /**
   * Chapter tujuan dan status bawaan dibaca dari URL.
   *
   * Datang dari KONTEKS tombolnya (kartu chapter, tombol "Impor Visitor"),
   * bukan dari isi berkasnya. Laporan BNI Connect tidak memuat kolom chapter,
   * dan menuntut orang menambahkannya hanya menambah satu tempat untuk salah
   * ketik yang memindahkan member beserta tagihannya.
   */
  const chapterTujuan = params.get('chapter') ?? ''
  const statusBawaan = params.get('status') === 'visitor' ? 'visitor' : undefined
  const opsi: ImportOpsi | undefined =
    chapterTujuan || statusBawaan ? { chapter: chapterTujuan || undefined, statusBawaan } : undefined

  const [jenis, setJenis] = useState<'members' | 'chapters'>('members')
  const [daftarChapter, setDaftarChapter] = useState<Chapter[]>([])
  const [indeks, setIndeks] = useState(0)
  const [berkas, setBerkas] = useState<File | null>(null)
  const [hasil, setHasil] = useState<ImportHasil | null>(null)
  const [riwayat, setRiwayat] = useState<Riwayat>({})
  const [sibuk, setSibuk] = useState(false)
  const [menyiapkan, setMenyiapkan] = useState(false)

  const langkah = jenis === 'chapters' ? LANGKAH_CHAPTER : statusBawaan ? LANGKAH_VISITOR : LANGKAH_MEMBER
  const selesai = indeks >= langkah.length
  const aktif = langkah[indeks] as Langkah | undefined
  const butuhChapter = jenis === 'members' && !chapterTujuan

  useEffect(() => {
    let hidup = true
    chapterService
      .list()
      .then((cs) => {
        if (hidup) setDaftarChapter(cs)
      })
      .catch(() => undefined)
    return () => {
      hidup = false
    }
  }, [])

  const namaChapter = daftarChapter.find((c) => c.id === chapterTujuan)?.displayName ?? ''

  const mulaiUlang = () => {
    setIndeks(0)
    setBerkas(null)
    setHasil(null)
    setRiwayat({})
    if (inputRef.current) inputRef.current.value = ''
  }

  /**
   * Mengganti chapter tujuan memulai wizard dari awal.
   *
   * Pratinjau dan riwayat yang tersisa menggambarkan chapter sebelumnya, sedang
   * tombol berikutnya akan menulis ke chapter yang baru. "12 baru" untuk BNI
   * Merdeka bisa berarti 12 penolakan di BNI Garuda. Ditulis ke URL supaya
   * tombol "Impor" di kartu chapter dan dropdown ini berbagi satu sumber.
   */
  const pilihChapter = (id: string) => {
    const baru = new URLSearchParams(params)
    if (id) baru.set('chapter', id)
    else baru.delete('chapter')
    setParams(baru, { replace: true })
    if (id) setJenis('members')
    mulaiUlang()
  }

  const pilihJenis = (j: 'members' | 'chapters') => {
    setJenis(j)
    mulaiUlang()
  }

  const pilihBerkas = (f: File | null) => {
    setBerkas(f)
    // Pratinjau lama dibuang saat berkasnya berganti, supaya tidak ada yang
    // menekan "Terapkan" atas laporan yang menggambarkan berkas lain.
    setHasil(null)
    if (!f && inputRef.current) inputRef.current.value = ''
  }

  const maju = (hasilLangkah: ImportHasil | null) => {
    if (!aktif) return
    setRiwayat((r) => ({ ...r, [aktif.key]: hasilLangkah }))
    setIndeks((i) => i + 1)
    setBerkas(null)
    setHasil(null)
    if (inputRef.current) inputRef.current.value = ''
  }

  const mundur = () => {
    setIndeks((i) => Math.max(0, i - 1))
    setBerkas(null)
    setHasil(null)
    if (inputRef.current) inputRef.current.value = ''
  }

  const jalankan = async (terapkan: boolean) => {
    if (!berkas || !aktif) return
    setSibuk(true)
    try {
      const out = terapkan
        ? await importService.apply(jenis, berkas, opsi)
        : await importService.preview(jenis, berkas, opsi)
      setHasil(out)
      if (terapkan) toast(`${aktif.judul}: ${out.baru} baru, ${out.diperbarui} diperbarui, ${out.ditolak} ditolak.`)
    } catch (err) {
      setHasil(null)
      toast(err instanceof Error ? err.message : 'Berkas tidak bisa dibaca.', 'error')
    } finally {
      setSibuk(false)
    }
  }

  const unduhTemplate = async (jenisTemplate: 'jatuhTempo' | 'chapter') => {
    setMenyiapkan(true)
    try {
      if (jenisTemplate === 'jatuhTempo') {
        // Terisi nama member chapter tujuan; kolom Due Date tinggal dilengkapi.
        const semua: MemberWithChapter[] = await memberService.list()
        downloadXlsx(
          `template-jatuh-tempo-${chapterTujuan}`,
          'Membership Dues',
          KOLOM_JATUH_TEMPO,
          semua
            .filter((m) => m.chapterId === chapterTujuan)
            .map((m) => [
              m.name,
              m.businessField ?? '',
              'Member',
              STATUS_BNI[m.status] ?? '',
              (m.renewalDate ?? '').slice(0, 10),
              (m.joinedDate ?? '').slice(0, 10),
            ]),
        )
      } else {
        const daftar: Chapter[] = await chapterService.list()
        downloadXlsx(
          'template-chapter',
          'Chapter',
          KOLOM_CHAPTER,
          daftar.map((c) => [c.id, c.name, c.displayName, c.areaName ?? '', c.cityName ?? '']),
        )
      }
      toast('Template diunduh. Lengkapi lalu unggah kembali di langkah ini.')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyiapkan template.', 'error')
    } finally {
      setMenyiapkan(false)
    }
  }

  return (
    <div>
      <PageHeader
        title="Impor Data"
        description={
          jenis === 'chapters'
            ? 'Impor daftar chapter. Selalu ditinjau dulu sebelum ditulis.'
            : statusBawaan
              ? 'Impor daftar visitor ke satu chapter. Selalu ditinjau dulu sebelum ditulis.'
              : 'Dua langkah dari laporan BNI Connect: data lama di BNI, lalu urutan jatuh tempo.'
        }
      />

      {/* Tujuan: satu baris di atas wizard. Mengubahnya memulai wizard dari awal. */}
      <Card className="mb-4">
        <div className="grid gap-4 p-4 sm:grid-cols-2">
          <Field
            label="Chapter tujuan"
            hint={
              jenis === 'members'
                ? 'Laporan BNI Connect tidak memuat kolom chapter; namanya dicocokkan di chapter ini.'
                : 'Impor chapter berlaku nasional.'
            }
          >
            <Select value={chapterTujuan} onChange={(e) => pilihChapter(e.target.value)} disabled={sibuk}>
              <option value="">{jenis === 'members' ? 'Pilih chapter…' : 'Semua chapter (nasional)'}</option>
              {daftarChapter.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.displayName}
                </option>
              ))}
            </Select>
          </Field>
          <Field
            label="Jenis data"
            hint={chapterTujuan ? 'Impor per chapter selalu tentang member.' : undefined}
          >
            <Select
              value={jenis}
              disabled={!!chapterTujuan || sibuk}
              onChange={(e) => pilihJenis(e.target.value as 'members' | 'chapters')}
            >
              <option value="members">{statusBawaan ? 'Visitor' : 'Member'}</option>
              <option value="chapters">Chapter</option>
            </Select>
          </Field>
        </div>
      </Card>

      <Stepper langkah={langkah} indeks={indeks} riwayat={riwayat} />

      {selesai ? (
        <Ringkasan
          langkah={langkah}
          riwayat={riwayat}
          namaChapter={namaChapter}
          onLihatMember={
            jenis === 'members' && chapterTujuan
              ? () => navigate(`/members?chapter=${encodeURIComponent(chapterTujuan)}`)
              : undefined
          }
          onUlang={mulaiUlang}
        />
      ) : aktif ? (
        <div className="grid gap-4 lg:grid-cols-[380px_1fr]">
          <Card>
            <CardHeader
              title={langkah.length > 1 ? `Langkah ${indeks + 1} · ${aktif.judul}` : aktif.judul}
              subtitle={aktif.berkas}
            />
            <CardBody className="space-y-4">
              <p className="text-sm leading-relaxed text-ink-600">
                {butuhChapter ? 'Pilih chapter tujuan di atas dulu.' : aktif.keterangan}
              </p>

              {aktif.template && !butuhChapter && (
                <TombolTemplate
                  onClick={() => unduhTemplate(aktif.template!)}
                  disabled={menyiapkan}
                  judul={
                    menyiapkan
                      ? 'Menyiapkan…'
                      : aktif.template === 'jatuhTempo'
                        ? 'Unduh template jatuh tempo'
                        : 'Unduh template chapter'
                  }
                  keterangan={
                    aktif.template === 'jatuhTempo'
                      ? `Format Membership Dues, terisi nama member ${namaChapter}. Pakai bila tidak memegang laporan aslinya.`
                      : 'Sudah terisi id dan nama chapter yang ada.'
                  }
                />
              )}

              <input
                ref={inputRef}
                type="file"
                accept=".csv,.xlsx,.xls,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-excel"
                className="hidden"
                onChange={(e) => pilihBerkas(e.target.files?.[0] ?? null)}
              />
              <div className="flex items-stretch gap-2">
                <button
                  type="button"
                  disabled={butuhChapter || sibuk || !!hasil?.diterapkan}
                  onClick={() => inputRef.current?.click()}
                  className="flex min-w-0 flex-1 items-center gap-3 rounded-xl border border-dashed border-ink-300 p-4 text-left transition-colors hover:border-ink-400 hover:bg-ink-50 disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:bg-transparent"
                >
                  <FileUp className="h-5 w-5 shrink-0 text-ink-400" />
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium text-ink-800">
                      {berkas ? berkas.name : 'Pilih berkas…'}
                    </span>
                    <span className="block text-xs text-ink-500">
                      {berkas ? `${(berkas.size / 1024).toFixed(0)} KB` : 'CSV, XLSX, atau .xls BNI Connect, maks. 10 MB'}
                    </span>
                  </span>
                </button>
                {berkas && !hasil?.diterapkan && (
                  <button
                    type="button"
                    onClick={() => pilihBerkas(null)}
                    aria-label="Hapus berkas"
                    className="rounded-xl border border-ink-200 px-3 text-ink-500 transition-colors hover:bg-ink-50 hover:text-ink-800"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>

              <div className="space-y-2">
                {!hasil?.diterapkan && (
                  <Button className="w-full" disabled={!berkas} loading={sibuk && !hasil} onClick={() => jalankan(false)}>
                    <Upload className="h-4 w-4" />
                    Tinjau
                  </Button>
                )}
                {hasil && !hasil.diterapkan && (
                  <Button variant="outline" className="w-full" loading={sibuk} onClick={() => jalankan(true)}>
                    Terapkan {hasil.baru + hasil.diperbarui} perubahan
                  </Button>
                )}
                {hasil?.diterapkan && (
                  <Button className="w-full" onClick={() => maju(hasil)}>
                    {indeks + 1 < langkah.length ? `Lanjut ke langkah ${indeks + 2}` : 'Selesai'}
                    <ArrowRight className="h-4 w-4" />
                  </Button>
                )}
              </div>

              <div className="flex items-center justify-between border-t border-ink-100 pt-3 text-sm">
                {indeks > 0 && !hasil?.diterapkan ? (
                  <button
                    type="button"
                    onClick={mundur}
                    disabled={sibuk}
                    className="flex items-center gap-1 text-ink-500 hover:text-ink-800"
                  >
                    <ArrowLeft className="h-4 w-4" />
                    Kembali
                  </button>
                ) : (
                  <span />
                )}
                {langkah.length > 1 && !hasil?.diterapkan && (
                  <button
                    type="button"
                    onClick={() => maju(null)}
                    disabled={sibuk || butuhChapter}
                    className="text-ink-500 hover:text-ink-800 disabled:opacity-50"
                  >
                    Lewati langkah ini
                  </button>
                )}
              </div>
            </CardBody>
          </Card>

          {hasil ? (
            <HasilImpor hasil={hasil} />
          ) : (
            <Card>
              <CardHeader title="Pratinjau" subtitle="Pilih berkas lalu tekan Tinjau." />
              <CardBody>
                <p className="py-8 text-center text-sm text-ink-500">
                  Belum ada berkas yang ditinjau. Tidak ada yang ditulis sebelum Anda menekan Terapkan.
                </p>
              </CardBody>
            </Card>
          )}
        </div>
      ) : null}
    </div>
  )
}

/** Penanda langkah di atas wizard: selesai, aktif, atau belum. */
function Stepper({ langkah, indeks, riwayat }: { langkah: Langkah[]; indeks: number; riwayat: Riwayat }) {
  const item = [...langkah.map((l) => l.judul), 'Selesai']
  return (
    <ol className="mb-4 flex flex-wrap items-center gap-x-2 gap-y-2">
      {item.map((judul, i) => {
        const lewat = i < indeks
        const sekarang = i === indeks
        const dilewati = lewat && langkah[i] && riwayat[langkah[i].key] === null
        return (
          <li key={judul} className="flex items-center gap-2">
            <span
              className={cn(
                'flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold',
                lewat && !dilewati && 'bg-emerald-500 text-white',
                dilewati && 'bg-ink-200 text-ink-500',
                sekarang && 'bg-brand-500 text-white',
                !lewat && !sekarang && 'bg-ink-100 text-ink-400',
              )}
            >
              {lewat && !dilewati ? <Check className="h-4 w-4" /> : i + 1}
            </span>
            <span className={cn('text-sm', sekarang ? 'font-semibold text-ink-900' : 'text-ink-500')}>
              {judul}
              {dilewati ? ' (dilewati)' : ''}
            </span>
            {i < item.length - 1 && <span className="mx-1 hidden h-px w-8 bg-ink-200 sm:block" />}
          </li>
        )
      })}
    </ol>
  )
}

function Ringkasan({
  langkah,
  riwayat,
  namaChapter,
  onLihatMember,
  onUlang,
}: {
  langkah: Langkah[]
  riwayat: Riwayat
  namaChapter: string
  onLihatMember?: () => void
  onUlang: () => void
}) {
  return (
    <Card>
      <CardHeader
        title="Impor selesai"
        subtitle={namaChapter ? `Chapter ${namaChapter}` : undefined}
      />
      <CardBody className="space-y-4">
        <ul className="divide-y divide-ink-100 rounded-xl border border-ink-200">
          {langkah.map((l, i) => {
            const h = riwayat[l.key]
            return (
              <li key={l.key} className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 text-sm">
                <span className="font-medium text-ink-800">
                  {langkah.length > 1 ? `${i + 1}. ` : ''}
                  {l.judul}
                </span>
                <span className="text-ink-600">
                  {h
                    ? `${h.baru} baru · ${h.diperbarui} diperbarui · ${h.sama} sama · ${h.ditolak} ditolak`
                    : 'Dilewati'}
                </span>
              </li>
            )
          })}
        </ul>
        <div className="flex flex-wrap gap-2">
          {onLihatMember && <Button onClick={onLihatMember}>Lihat member chapter</Button>}
          <Button variant="outline" onClick={onUlang}>
            Impor lagi
          </Button>
        </div>
      </CardBody>
    </Card>
  )
}

function TombolTemplate({
  onClick,
  disabled,
  judul,
  keterangan,
}: {
  onClick: () => void
  disabled: boolean
  judul: string
  keterangan: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="flex w-full items-center gap-3 rounded-xl border border-ink-200 p-3 text-left transition-colors hover:bg-ink-50 disabled:opacity-50"
    >
      <Download className="h-4 w-4 shrink-0 text-brand-500" />
      <span className="min-w-0">
        <span className="block text-sm font-medium text-ink-800">{judul}</span>
        <span className="block text-xs leading-snug text-ink-500">{keterangan}</span>
      </span>
    </button>
  )
}

/** Ringkasan, status tulis, peringatan, dan tabel baris satu berkas. */
function HasilImpor({ hasil }: { hasil: ImportHasil }) {
  return (
    <Card>
      <CardHeader
        title={hasil.diterapkan ? 'Sudah diterapkan' : 'Pratinjau, belum ditulis'}
        subtitle={`${hasil.total} baris · ${hasil.baru} baru · ${hasil.diperbarui} diperbarui · ${hasil.sama} sama · ${hasil.ditolak} ditolak`}
      />
      <CardBody>
        <div className="space-y-3">
          <div
            className={cn(
              'flex items-start gap-2 rounded-lg p-3 text-xs',
              hasil.diterapkan ? 'bg-emerald-50 text-emerald-800' : 'bg-amber-50 text-amber-800',
            )}
          >
            {hasil.diterapkan ? (
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
            ) : (
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
            )}
            <span>
              {hasil.diterapkan
                ? 'Perubahan sudah ditulis ke basis data.'
                : 'Belum ada satu baris pun yang ditulis. Tekan "Terapkan" untuk menyimpannya.'}
            </span>
          </div>

          {hasil.peringatan?.map((p) => (
            <div key={p} className="rounded-lg bg-ink-50 p-3 text-xs text-ink-700">
              {p}
            </div>
          ))}

          <Table>
            <THead>
              <Tr>
                <Th className="w-16">Baris</Th>
                <Th>ID</Th>
                <Th>Nama</Th>
                <Th>Tindakan</Th>
                <Th>Keterangan</Th>
              </Tr>
            </THead>
            <TBody>
              {hasil.baris.map((b) => (
                <Tr key={`${b.nomor}-${b.id}`}>
                  {/* Nomor mengikuti Excel, supaya barisnya bisa dicari di berkas aslinya. */}
                  <Td className="tabular-nums text-ink-500">{b.nomor}</Td>
                  <Td className="font-mono text-xs text-ink-700">{b.id || '—'}</Td>
                  <Td className="text-ink-800">{b.nama || '—'}</Td>
                  <Td>
                    <Badge tone={TONE[b.tindakan]}>{b.tindakan}</Badge>
                  </Td>
                  <Td className="text-xs text-ink-600">{b.alasan ?? b.perubahan?.join(', ') ?? ''}</Td>
                </Tr>
              ))}
            </TBody>
          </Table>
        </div>
      </CardBody>
    </Card>
  )
}
