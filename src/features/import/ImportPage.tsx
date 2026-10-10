import { useEffect, useRef, useState } from 'react'
import { AlertTriangle, CheckCircle2, Download, FileUp, Upload, X } from 'lucide-react'
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
import { useSearchParams } from 'react-router-dom'
import { chapterService, importService, memberService } from '@/services'
import type { ImportOpsi } from '@/services/types'
import { downloadXlsx } from '@/lib/xlsx'

/**
 * Impor chapter dan member dari berkas.
 *
 * SELALU dua langkah, dan tombol "Terapkan" baru muncul SETELAH pratinjau.
 * Bukan kenyamanan: berkas keanggotaan disusun manual, sering hasil salin-tempel
 * dari beberapa sumber, dan kesalahan di dalamnya tidak kelihatan sampai
 * tagihannya salah kirim — chapter yang salah ketik memindahkan member, kolom
 * yang tergeser menyimpan nomor telepon sebagai nama perusahaan, dan id yang
 * tanpa sengaja sama menimpa orang lain.
 *
 * Angka pratinjau dihitung KODE YANG SAMA dengan yang menulis, di server. Itu
 * yang membuatnya bisa dipercaya: tidak mungkin berbeda dari yang akhirnya
 * terjadi.
 *
 * IMPOR MEMBER ADALAH SATU ALUR TIGA BERKAS, bukan tiga impor terpisah.
 * Data satu chapter datang dari tiga sumber yang masing-masing hanya memuat
 * sebagian: data master (nama, HP, email), laporan jatuh tempo (urutan
 * perpanjangan), dan data yang tercatat di BNI (perusahaan, bidang usaha).
 * Urutannya menentukan: dua berkas terakhir mencocokkan NAMA ke member yang
 * sudah ada di chapter, jadi data master harus masuk lebih dulu. Halaman ini
 * menerapkannya berurutan dalam satu tekanan tombol supaya urutan itu tidak
 * bergantung pada ingatan orang yang mengunggah.
 */

/**
 * Judul kolom yang ditulis ke template.
 *
 * Sengaja memakai nama KANONIK yang dibaca importer, bukan nama yang enak
 * dibaca manusia. Importer menerima beberapa alias untuk kolom yang sama
 * ("phone", "telepon", "hp", "no_hp"), tapi template yang memakai alias
 * mengajarkan orang menulis judul yang kebetulan bekerja hari ini — dan alias
 * adalah hal pertama yang hilang saat berkasnya disalin-tempel ke tempat lain.
 */
const KOLOM_MEMBER = ['id', 'name', 'chapter_id', 'status', 'email', 'phone', 'company', 'business_field']
const KOLOM_CHAPTER = ['id', 'name', 'display_name', 'area_name', 'city_name']

/**
 * Judul kolom laporan "Membership Dues" BNI Connect, persis seperti ekspornya.
 *
 * Importer mengenali laporan itu dari pasangan "Member Name" + "Due Date" dan
 * mencocokkan barisnya ke member lewat nama di chapter tujuan. Template dengan
 * judul yang sama berarti berkas yang diisi tangan dan berkas yang diunduh dari
 * BNI Connect menempuh jalur baca yang satu, bukan dua format yang harus
 * dirawat berdampingan.
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

type Langkah = 'master' | 'jatuhTempo' | 'bni' | 'chapter'

interface DefinisiLangkah {
  key: Langkah
  judul: string
  keterangan: string
  /** Laporan BNI Connect tidak memuat kolom chapter; namanya dicocokkan di dalam chapter tujuan. */
  butuhChapter: boolean
}

/**
 * Urutan mengikuti ketergantungan datanya, bukan selera. Langkah 1 boleh
 * membuat member baru; langkah 2 dan 3 hanya melengkapi member yang sudah ada
 * dengan mencocokkan nama di chapter tujuan. Importer mengenali setiap format
 * dari isinya, jadi slot ini pengarah, bukan pembatas: laporan Membership
 * Length yang diunggah di langkah 3 tetap terbaca.
 */
const LANGKAH_MEMBER: DefinisiLangkah[] = [
  {
    key: 'master',
    judul: 'Data master',
    keterangan: 'Nama, HP, dan email: template member dari halaman ini (template-member-<chapter>.xlsx).',
    butuhChapter: false,
  },
  {
    key: 'jatuhTempo',
    judul: 'Urutan jatuh tempo',
    keterangan: 'Laporan Membership Dues BNI Connect, atau template jatuh tempo. Mengisi tanggal perpanjangan.',
    butuhChapter: true,
  },
  {
    key: 'bni',
    judul: 'Data lama di BNI',
    keterangan: 'Laporan Chapter Roster BNI Connect: perusahaan, bidang usaha, telepon yang tercatat di BNI. Laporan Membership Length juga diterima.',
    butuhChapter: true,
  },
]

const LANGKAH_CHAPTER: DefinisiLangkah[] = [
  {
    key: 'chapter',
    judul: 'Daftar chapter',
    keterangan: 'Template chapter dari halaman ini.',
    butuhChapter: false,
  },
]

export function ImportPage() {
  const { toast } = useToast()
  const inputRef = useRef<Partial<Record<Langkah, HTMLInputElement | null>>>({})
  const [params, setParams] = useSearchParams()

  /**
   * Chapter tujuan dan status bawaan dibaca dari URL.
   *
   * Datang dari KONTEKS tombolnya — kartu chapter tertentu, tombol "Impor
   * Visitor" — bukan dari isi berkasnya. Daftar hadir pertemuan tidak pernah
   * memuat kolom chapter maupun status, dan menuntut orang menambahkan kolom
   * yang isinya sama di setiap baris hanya menambah satu tempat untuk salah
   * ketik. Salah ketik di kolom chapter memindahkan member beserta tagihannya.
   */
  const chapterTujuan = params.get('chapter') ?? ''
  const statusBawaan = params.get('status') === 'visitor' ? 'visitor' : undefined
  const opsi: ImportOpsi | undefined =
    chapterTujuan || statusBawaan
      ? { chapter: chapterTujuan || undefined, statusBawaan }
      : undefined

  const [jenis, setJenis] = useState<'members' | 'chapters'>('members')
  const [daftarChapter, setDaftarChapter] = useState<Chapter[]>([])
  const [berkas, setBerkas] = useState<Partial<Record<Langkah, File>>>({})
  const [hasil, setHasil] = useState<Partial<Record<Langkah, ImportHasil>>>({})
  const [sibuk, setSibuk] = useState(false)
  const [menyiapkan, setMenyiapkan] = useState(false)

  const langkah = jenis === 'members' ? LANGKAH_MEMBER : LANGKAH_CHAPTER
  const terisi = langkah.filter((l) => berkas[l.key])
  const sudahDitinjau = terisi.length > 0 && terisi.every((l) => hasil[l.key])
  const belumDiterapkan = terisi.filter((l) => hasil[l.key] && !hasil[l.key]?.diterapkan)
  const perubahan = belumDiterapkan.reduce((n, l) => n + (hasil[l.key]!.baru + hasil[l.key]!.diperbarui), 0)

  useEffect(() => {
    let aktif = true
    chapterService
      .list()
      .then((cs) => {
        if (aktif) setDaftarChapter(cs)
      })
      .catch(() => undefined)
    return () => {
      aktif = false
    }
  }, [])

  // Nama yang dibaca orang, bukan id. Layar yang cuma menampilkan "ch-garuda"
  // menuntut orang mengingat pemetaannya sendiri, dan salah ingat di sini
  // berarti mengunggah daftar ke chapter yang salah.
  const namaChapter = daftarChapter.find((c) => c.id === chapterTujuan)?.displayName ?? ''

  /**
   * Mengganti chapter tujuan lewat dropdown.
   *
   * Ditulis ke URL, bukan disimpan di state saja. Tombol "Impor" di kartu
   * chapter sudah mengirim `?chapter=`, jadi satu sumber kebenaran menjaga
   * kedua jalan masuk itu tetap sama, dan alamatnya tetap bisa disalin ke orang
   * lain.
   *
   * PRATINJAU LAMA DIBUANG. Ia menggambarkan chapter yang tadi dipilih, sedang
   * tombol "Terapkan" di sebelahnya akan menulis ke chapter yang baru. Angka
   * yang terbaca "12 baru" dari daftar BNI Merdeka bisa berarti 12 penolakan di
   * BNI Garuda, dan yang menekan tombolnya tidak punya cara melihat bedanya.
   * Aturannya sama dengan pergantian berkas di bawah.
   */
  const pilihChapter = (id: string) => {
    const baru = new URLSearchParams(params)
    if (id) baru.set('chapter', id)
    else baru.delete('chapter')
    setParams(baru, { replace: true })
    setHasil({})
    // Impor yang ditujukan ke satu chapter selalu tentang member.
    if (id) setJenis('members')
  }

  const pilihBerkas = (key: Langkah, f: File | null) => {
    setBerkas((b) => {
      const baru = { ...b }
      if (f) baru[key] = f
      else delete baru[key]
      return baru
    })
    // Pratinjau lama DIBUANG saat satu berkas pun berganti. Langkah-langkah
    // ini saling bergantung, jadi pratinjau yang tersisa menggambarkan
    // rangkaian yang sudah tidak ada. Membiarkannya membuat orang menekan
    // "Terapkan" atas laporan yang menggambarkan berkas lain.
    setHasil({})
    const input = inputRef.current[key]
    if (input) input.value = ''
  }

  /**
   * Template kedua, untuk tanggal jatuh tempo, mengikuti format BNI Connect.
   *
   * Terikat ke chapter tujuan karena laporan BNI Connect tidak memuat kolom
   * chapter, dan importer mencocokkan nama hanya di dalam chapter itu.
   */
  const unduhTemplateJatuhTempo = async () => {
    setMenyiapkan(true)
    try {
      const semua: MemberWithChapter[] = await memberService.list()
      const anggota = semua.filter((m) => m.chapterId === chapterTujuan)
      downloadXlsx(
        `template-jatuh-tempo-${chapterTujuan}`,
        'Membership Dues',
        KOLOM_JATUH_TEMPO,
        anggota.map((m) => [
          m.name,
          m.businessField ?? '',
          'Member',
          STATUS_BNI[m.status] ?? '',
          (m.renewalDate ?? '').slice(0, 10),
          (m.joinedDate ?? '').slice(0, 10),
        ]),
      )
      toast('Template jatuh tempo diunduh. Isi kolom Due Date, lalu unggah kembali di langkah 2.')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyiapkan template.', 'error')
    } finally {
      setMenyiapkan(false)
    }
  }

  /**
   * Template BERISI DATA YANG SUDAH ADA, bukan lembar kosong.
   *
   * MOM meminta template "untuk import phone number dan email karena data
   * import kurang lengkap" — dan itu menentukan bentuknya. Lembar kosong
   * memaksa orang mengetik ulang id dan nama setiap member hanya untuk
   * menambahkan satu nomor telepon, dan setiap pengetikan ulang adalah peluang
   * id-nya salah — yang berarti bukan melengkapi data, melainkan membuat member
   * baru atau menimpa orang lain.
   *
   * Jadi template ini sudah terisi id, nama, dan chapter; kolom email dan
   * telepon dibiarkan kosong persis di baris yang memang belum punya. Yang
   * sudah terisi ikut dibawa supaya tidak terhapus saat diimpor kembali.
   */
  const unduhTemplate = async () => {
    setMenyiapkan(true)
    try {
      if (jenis === 'members') {
        const semua: MemberWithChapter[] = await memberService.list()
        // Template ikut menyempit ke chapter tujuan.
        //
        // Template yang memuat seluruh member nasional pada layar yang jelas
        // menyebut satu chapter adalah undangan untuk mengunggahnya kembali apa
        // adanya — dan setiap baris dari chapter lain akan ditolak, sehingga
        // yang terlihat adalah ratusan penolakan alih-alih pekerjaan yang beres.
        const anggota = chapterTujuan ? semua.filter((m) => m.chapterId === chapterTujuan) : semua
        downloadXlsx(
          chapterTujuan ? `template-member-${chapterTujuan}` : 'template-member',
          'Member',
          KOLOM_MEMBER,
          anggota.map((m) => [
            m.id,
            m.name,
            m.chapterId,
            m.status,
            m.email ?? '',
            m.phone ?? '',
            m.company ?? '',
            m.businessField ?? '',
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
      toast('Template diunduh. Isi kolom yang kosong, lalu unggah kembali di halaman ini.')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyiapkan template.', 'error')
    } finally {
      setMenyiapkan(false)
    }
  }

  /**
   * Menjalankan langkah-langkah yang berkasnya ada, BERURUTAN.
   *
   * Pratinjau: setiap berkas ditinjau terhadap data yang tersimpan sekarang.
   * Nama yang baru muncul di data master belum dikenal langkah 2 dan 3 sampai
   * langkah 1 ditulis; pratinjaunya menyebut mereka "baru", dan itu benar
   * untuk saat itu. Terapkan: langkah 1 ditulis dulu, baru langkah 2 membaca
   * basis data yang sudah memuat nama-nama itu. Satu langkah yang gagal
   * menghentikan sisanya; yang sudah ditulis tetap tertulis dan ditandai.
   */
  const jalankan = async (terapkan: boolean) => {
    const antrean = terapkan ? belumDiterapkan : terisi
    if (antrean.length === 0) return
    setSibuk(true)
    try {
      for (const l of antrean) {
        const f = berkas[l.key]!
        const out = terapkan
          ? await importService.apply(jenis, f, opsi)
          : await importService.preview(jenis, f, opsi)
        setHasil((h) => ({ ...h, [l.key]: out }))
        if (terapkan) {
          toast(`${l.judul}: ${out.baru} baru, ${out.diperbarui} diperbarui, ${out.ditolak} ditolak.`)
        }
      }
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Berkas tidak bisa dibaca.', 'error')
    } finally {
      setSibuk(false)
    }
  }

  return (
    <div>
      <PageHeader
        title="Impor Data"
        description="Satu alur tiga berkas untuk member: data master, jatuh tempo, lalu data lama di BNI. Selalu ditinjau dulu sebelum ditulis."
      />

      {chapterTujuan && (
        <div className="mb-4 flex flex-wrap items-center gap-x-2 gap-y-1 rounded-xl border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-900">
          <span className="font-semibold">
            Impor ditujukan ke {namaChapter || chapterTujuan}
            {statusBawaan === 'visitor' ? ' sebagai Visitor' : ''}.
          </span>
          <span className="text-blue-800">
            Kolom <code className="rounded bg-blue-100 px-1">chapter_id</code> boleh dikosongkan.
            Baris yang menyebut chapter lain <strong>ditolak</strong>, bukan dipindahkan.
          </span>
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-[400px_1fr]">
        <Card>
          <CardHeader title="Berkas" subtitle="Kolom dicari lewat judulnya, bukan urutannya." />
          <CardBody className="space-y-4">
            {/* Chapter tujuan bisa dipilih di sini, bukan hanya diwarisi dari
                tombol "Impor" di kartu chapter. Orang yang sudah berada di
                halaman ini tidak punya jalan lain selain menyunting alamatnya
                sendiri, dan yang tidak tahu caranya akan mengunggah daftar satu
                chapter ke lingkup nasional. */}
            <Field
              label="Chapter tujuan"
              hint={
                chapterTujuan
                  ? 'Kolom chapter_id di berkas boleh dikosongkan.'
                  : 'Nasional: setiap baris wajib menyebut chapter_id sendiri. Langkah 2 dan 3 butuh satu chapter.'
              }
            >
              <Select value={chapterTujuan} onChange={(e) => pilihChapter(e.target.value)}>
                <option value="">Semua chapter (nasional)</option>
                {daftarChapter.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.displayName}
                  </option>
                ))}
              </Select>
            </Field>

            <Field
              label="Jenis data"
              hint={chapterTujuan ? 'Terkunci: impor per chapter selalu tentang member.' : undefined}
            >
              <Select
                value={jenis}
                disabled={!!chapterTujuan}
                onChange={(e) => {
                  setJenis(e.target.value as 'members' | 'chapters')
                  setBerkas({})
                  setHasil({})
                }}
              >
                <option value="members">Member</option>
                <option value="chapters">Chapter</option>
              </Select>
            </Field>

            {/* Template diletakkan SEBELUM slot berkas, bukan sesudahnya.
                Urutannya mengikuti urutan pekerjaannya: orang datang ke sini
                justru karena datanya belum lengkap, jadi mengunduh template
                adalah langkah pertama, bukan pelengkap di bawah. */}
            <div className="space-y-2">
              <TombolTemplate
                onClick={unduhTemplate}
                disabled={menyiapkan}
                judul={menyiapkan ? 'Menyiapkan…' : jenis === 'members' ? 'Unduh template data master' : 'Unduh template chapter'}
                keterangan={
                  jenis === 'members'
                    ? 'Sudah terisi id, nama, dan chapter yang ada; tinggal lengkapi email dan nomor telepon.'
                    : 'Sudah terisi id dan nama yang ada; tinggal lengkapi kolom kosongnya.'
                }
              />
              {jenis === 'members' && chapterTujuan && (
                <TombolTemplate
                  onClick={unduhTemplateJatuhTempo}
                  disabled={menyiapkan}
                  judul={menyiapkan ? 'Menyiapkan…' : 'Unduh template jatuh tempo'}
                  keterangan={`Format laporan Membership Dues BNI Connect: nama member ${namaChapter} sudah terisi, tinggal isi kolom Due Date.`}
                />
              )}
            </div>

            <ol className="space-y-3">
              {langkah.map((l, i) => {
                const f = berkas[l.key]
                const terkunci = l.butuhChapter && !chapterTujuan
                return (
                  <li key={l.key} className={terkunci ? 'opacity-60' : undefined}>
                    <input
                      ref={(el) => {
                        inputRef.current[l.key] = el
                      }}
                      type="file"
                      accept=".csv,.xlsx,.xls,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-excel"
                      className="hidden"
                      onChange={(e) => pilihBerkas(l.key, e.target.files?.[0] ?? null)}
                    />
                    <div className="flex items-start gap-3">
                      <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-ink-900 text-xs font-semibold text-white">
                        {langkah.length > 1 ? i + 1 : <FileUp className="h-3.5 w-3.5" />}
                      </span>
                      <div className="min-w-0 flex-1">
                        <div className="text-sm font-semibold text-ink-900">{l.judul}</div>
                        <p className="mt-0.5 text-xs leading-snug text-ink-500">
                          {terkunci ? 'Pilih chapter tujuan dulu: laporan ini tidak memuat kolom chapter.' : l.keterangan}
                        </p>
                        <div className="mt-2 flex items-stretch gap-2">
                          <button
                            type="button"
                            disabled={terkunci || sibuk}
                            onClick={() => inputRef.current[l.key]?.click()}
                            className="flex min-w-0 flex-1 items-center gap-2 rounded-xl border border-dashed border-ink-300 px-3 py-2 text-left transition-colors hover:border-ink-400 hover:bg-ink-50 disabled:cursor-not-allowed disabled:hover:bg-transparent"
                          >
                            <FileUp className="h-4 w-4 shrink-0 text-ink-400" />
                            <span className="min-w-0">
                              <span className="block truncate text-sm font-medium text-ink-800">
                                {f ? f.name : 'Pilih berkas…'}
                              </span>
                              <span className="block text-xs text-ink-500">
                                {f ? `${(f.size / 1024).toFixed(0)} KB` : 'CSV, XLSX, atau .xls BNI Connect'}
                              </span>
                            </span>
                          </button>
                          {f && (
                            <button
                              type="button"
                              onClick={() => pilihBerkas(l.key, null)}
                              aria-label={`Hapus berkas ${l.judul}`}
                              className="rounded-xl border border-ink-200 px-2.5 text-ink-500 transition-colors hover:bg-ink-50 hover:text-ink-800"
                            >
                              <X className="h-4 w-4" />
                            </button>
                          )}
                        </div>
                      </div>
                    </div>
                  </li>
                )
              })}
            </ol>

            <div className="space-y-2">
              <Button
                className="w-full"
                disabled={terisi.length === 0}
                loading={sibuk && !sudahDitinjau}
                onClick={() => jalankan(false)}
              >
                <Upload className="h-4 w-4" />
                Tinjau{terisi.length > 1 ? ` ${terisi.length} berkas` : ''}
              </Button>
              {/* Tombol terapkan baru ADA setelah seluruh pratinjau, dan hilang
                  lagi begitu satu berkas berganti. Menyediakannya lebih awal
                  berarti menawarkan penulisan atas sesuatu yang belum pernah
                  dilihat. */}
              {sudahDitinjau && belumDiterapkan.length > 0 && (
                <Button variant="outline" className="w-full" loading={sibuk} onClick={() => jalankan(true)}>
                  Terapkan {perubahan} perubahan
                  {belumDiterapkan.length > 1 ? ` (${belumDiterapkan.length} langkah berurutan)` : ''}
                </Button>
              )}
            </div>

            <p className="text-xs leading-relaxed text-ink-500">
              Kolom yang <strong>tidak ada</strong> di berkas tidak mengosongkan data tersimpan:
              mengirim daftar nomor telepon saja tidak akan menghapus email siapa pun. Langkah 2
              dan 3 mencocokkan nama ke member yang sudah ada; nama yang baru masuk lewat langkah 1
              baru dikenal setelah langkah 1 ditulis.
            </p>
          </CardBody>
        </Card>

        <div className="space-y-4">
          {!terisi.some((l) => hasil[l.key]) ? (
            <Card>
              <CardHeader title="Hasil" subtitle="Pilih berkas lalu tekan Tinjau." />
              <CardBody>
                <p className="py-8 text-center text-sm text-ink-500">Belum ada berkas yang ditinjau.</p>
              </CardBody>
            </Card>
          ) : (
            terisi.map((l) => (
              <HasilLangkah
                key={l.key}
                nomor={langkah.length > 1 ? langkah.indexOf(l) + 1 : undefined}
                judul={l.judul}
                hasil={hasil[l.key]}
              />
            ))
          )}
        </div>
      </div>
    </div>
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

/** Hasil satu langkah: ringkasan, status tulis, peringatan, dan tabel barisnya. */
function HasilLangkah({ nomor, judul, hasil }: { nomor?: number; judul: string; hasil?: ImportHasil }) {
  const label = nomor ? `Langkah ${nomor} · ${judul}` : judul
  if (!hasil) {
    return (
      <Card>
        <CardHeader title={label} subtitle="Menunggu giliran ditinjau." />
      </Card>
    )
  }
  return (
    <Card>
      <CardHeader
        title={`${label} · ${hasil.diterapkan ? 'sudah diterapkan' : 'pratinjau, belum ditulis'}`}
        subtitle={`${hasil.total} baris · ${hasil.baru} baru · ${hasil.diperbarui} diperbarui · ${hasil.sama} sama · ${hasil.ditolak} ditolak`}
      />
      <CardBody>
        <div className="space-y-3">
          <div
            className={`flex items-start gap-2 rounded-lg p-3 text-xs ${
              hasil.diterapkan ? 'bg-emerald-50 text-emerald-800' : 'bg-amber-50 text-amber-800'
            }`}
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
                  {/* Nomor mengikuti Excel, supaya barisnya bisa langsung
                      dicari di berkas aslinya. */}
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
