import { useRef, useState } from 'react'
import { AlertTriangle, CheckCircle2, Download, FileUp, Upload } from 'lucide-react'
import type { Chapter, ImportBaris, ImportHasil, MemberWithChapter } from '@/types'
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

const TONE: Record<ImportBaris['tindakan'], 'green' | 'amber' | 'gray' | 'red'> = {
  baru: 'green',
  diperbarui: 'amber',
  sama: 'gray',
  ditolak: 'red',
}

export function ImportPage() {
  const { toast } = useToast()
  const inputRef = useRef<HTMLInputElement>(null)
  const [jenis, setJenis] = useState<'members' | 'chapters'>('members')
  const [file, setFile] = useState<File | null>(null)
  const [hasil, setHasil] = useState<ImportHasil | null>(null)
  const [sibuk, setSibuk] = useState(false)

  const pilihBerkas = (f: File | null) => {
    setFile(f)
    // Pratinjau lama DIBUANG saat berkasnya berganti. Membiarkannya membuat
    // orang menekan "Terapkan" atas laporan yang menggambarkan berkas lain.
    setHasil(null)
  }

  const [menyiapkan, setMenyiapkan] = useState(false)

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
        const anggota: MemberWithChapter[] = await memberService.list()
        downloadXlsx(
          'template-member',
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

  const jalankan = async (terapkan: boolean) => {
    if (!file) return
    setSibuk(true)
    try {
      const out = terapkan
        ? await importService.apply(jenis, file)
        : await importService.preview(jenis, file)
      setHasil(out)
      if (terapkan) {
        toast(`Diterapkan: ${out.baru} baru, ${out.diperbarui} diperbarui, ${out.ditolak} ditolak.`)
      }
    } catch (err) {
      setHasil(null)
      toast(err instanceof Error ? err.message : 'Berkas tidak bisa dibaca.', 'error')
    } finally {
      setSibuk(false)
    }
  }

  return (
    <div>
      <PageHeader
        title="Impor Data"
        description="Unggah CSV atau XLSX berisi chapter atau member. Selalu ditinjau dulu sebelum ditulis."
      />

      <div className="grid gap-4 lg:grid-cols-[360px_1fr]">
        <Card>
          <CardHeader title="Berkas" subtitle="Kolom dicari lewat judulnya, bukan urutannya." />
          <CardBody className="space-y-4">
            <Field label="Jenis data">
              <Select
                value={jenis}
                onChange={(e) => {
                  setJenis(e.target.value as 'members' | 'chapters')
                  setHasil(null)
                }}
              >
                <option value="members">Member</option>
                <option value="chapters">Chapter</option>
              </Select>
            </Field>

            <div>
              <input
                ref={inputRef}
                type="file"
                accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                className="hidden"
                onChange={(e) => pilihBerkas(e.target.files?.[0] ?? null)}
              />
              <button
                onClick={() => inputRef.current?.click()}
                className="flex w-full items-center gap-3 rounded-xl border border-dashed border-ink-300 p-4 text-left transition-colors hover:border-ink-400 hover:bg-ink-50"
              >
                <FileUp className="h-5 w-5 shrink-0 text-ink-400" />
                <span className="min-w-0">
                  <span className="block truncate text-sm font-medium text-ink-800">
                    {file ? file.name : 'Pilih berkas…'}
                  </span>
                  <span className="block text-xs text-ink-500">
                    {file ? `${(file.size / 1024).toFixed(0)} KB` : 'CSV atau XLSX, maksimal 10 MB'}
                  </span>
                </span>
              </button>
            </div>

            {/* Template diletakkan SEBELUM tombol unggah, bukan sesudahnya.
                Urutannya mengikuti urutan pekerjaannya: orang datang ke sini
                justru karena datanya belum lengkap, jadi mengunduh template
                adalah langkah pertama, bukan pelengkap di bawah. */}
            <button
              onClick={unduhTemplate}
              disabled={menyiapkan}
              className="flex w-full items-center gap-3 rounded-xl border border-ink-200 p-3 text-left transition-colors hover:bg-ink-50 disabled:opacity-50"
            >
              <Download className="h-4 w-4 shrink-0 text-brand-500" />
              <span className="min-w-0">
                <span className="block text-sm font-medium text-ink-800">
                  {menyiapkan ? 'Menyiapkan…' : 'Unduh template'}
                </span>
                <span className="block text-xs leading-snug text-ink-500">
                  Sudah terisi {jenis === 'members' ? 'id, nama, dan chapter' : 'id dan nama'} yang ada —
                  {jenis === 'members' ? ' tinggal lengkapi email dan nomor telepon.' : ' tinggal lengkapi kolom kosongnya.'}
                </span>
              </span>
            </button>

            <div className="space-y-2">
              <Button className="w-full" disabled={!file} loading={sibuk && !hasil} onClick={() => jalankan(false)}>
                <Upload className="h-4 w-4" />
                Tinjau
              </Button>
              {/* Tombol terapkan baru ADA setelah pratinjau, dan hilang lagi
                  begitu berkasnya berganti. Menyediakannya lebih awal berarti
                  menawarkan penulisan atas sesuatu yang belum pernah dilihat. */}
              {hasil && !hasil.diterapkan && (
                <Button
                  variant="outline"
                  className="w-full"
                  loading={sibuk}
                  onClick={() => jalankan(true)}
                >
                  Terapkan {hasil.baru + hasil.diperbarui} perubahan
                </Button>
              )}
            </div>

            <p className="text-xs leading-relaxed text-ink-500">
              Kolom yang <strong>tidak ada</strong> di berkas tidak mengosongkan data tersimpan —
              mengirim daftar nomor telepon saja tidak akan menghapus email siapa pun.
            </p>
          </CardBody>
        </Card>

        <Card>
          <CardHeader
            title={hasil ? (hasil.diterapkan ? 'Sudah diterapkan' : 'Pratinjau — belum ditulis') : 'Hasil'}
            subtitle={
              hasil
                ? `${hasil.total} baris · ${hasil.baru} baru · ${hasil.diperbarui} diperbarui · ${hasil.sama} sama · ${hasil.ditolak} ditolak`
                : 'Pilih berkas lalu tekan Tinjau.'
            }
          />
          <CardBody>
            {!hasil ? (
              <p className="py-8 text-center text-sm text-ink-500">Belum ada berkas yang ditinjau.</p>
            ) : (
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
                        <Td className="text-xs text-ink-600">
                          {b.alasan ?? b.perubahan?.join(', ') ?? ''}
                        </Td>
                      </Tr>
                    ))}
                  </TBody>
                </Table>
              </div>
            )}
          </CardBody>
        </Card>
      </div>
    </div>
  )
}
