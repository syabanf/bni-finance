import { useMemo, useState } from 'react'
import { Search } from 'lucide-react'
import type { Chapter, MemberWithChapter } from '@/types'
import {
  Card,
  EmptyState,
  Input,
  LoadingState,
  MemberStatusBadge,
  PageHeader,
  Select,
  TBody,
  Table,
  Td,
  THead,
  Th,
  Tr,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { chapterService, memberService } from '@/services'
import { formatDate } from '@/lib/format'

/**
 * Lama keanggotaan tiap member, meniru laporan "Membership Length" BNI Connect.
 *
 * Dihitung dari tanggal bergabung yang tersimpan, bukan diimpor sebagai teks
 * "3 years (1 month)" dari laporannya. Teks itu benar pada hari laporan
 * diekspor dan basi keesokan harinya; tanggalnya yang bertahan. Laporan
 * Membership Length tetap bisa diimpor lewat halaman Impor, dan yang ia isi
 * adalah tanggal bergabung ini.
 *
 * Yang tidak ada di sini: kolom "Recent Length" untuk member yang pernah
 * keluar lalu bergabung lagi. Sistem ini hanya menyimpan satu tanggal
 * bergabung.
 */

/** Selisih bulan penuh dari tanggal bergabung sampai hari ini. */
export function bulanKeanggotaan(joined: string | null, hariIni = new Date()): number | null {
  if (!joined) return null
  const d = new Date(joined)
  if (Number.isNaN(d.getTime())) return null
  let bulan = (hariIni.getFullYear() - d.getFullYear()) * 12 + (hariIni.getMonth() - d.getMonth())
  if (hariIni.getDate() < d.getDate()) bulan -= 1
  return Math.max(0, bulan)
}

export function labelLama(bulan: number | null): string {
  if (bulan === null) return '—'
  const tahun = Math.floor(bulan / 12)
  const sisa = bulan % 12
  if (tahun === 0) return `${sisa} bulan`
  return sisa === 0 ? `${tahun} tahun` : `${tahun} tahun ${sisa} bulan`
}

export function MembershipLengthPage() {
  const { data: members, loading } = useAsync<MemberWithChapter[]>(() => memberService.list())
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())
  const [chapterId, setChapterId] = useState('all')
  const [search, setSearch] = useState('')

  const rows = useMemo(() => {
    if (!members) return []
    const q = search.trim().toLowerCase()
    return members
      .filter((m) => chapterId === 'all' || m.chapterId === chapterId)
      .filter((m) => !q || m.name.toLowerCase().includes(q))
      .map((m) => ({ m, bulan: bulanKeanggotaan(m.joinedDate) }))
      // Terlama di atas. Yang tanpa tanggal bergabung di bawah, bukan dibuang:
      // merekalah yang paling perlu dilengkapi lewat impor laporan Length.
      .sort((a, b) => (b.bulan ?? -1) - (a.bulan ?? -1) || a.m.name.localeCompare(b.m.name))
  }, [members, chapterId, search])

  const tanpaTanggal = rows.filter((r) => r.bulan === null).length

  if (loading || !members) return <LoadingState label="Memuat member…" />

  return (
    <div>
      <PageHeader
        title="Lama Keanggotaan"
        description="Dihitung dari tanggal bergabung sampai hari ini, terlama di atas."
      />

      <Card>
        <div className="space-y-3 border-b border-ink-100 p-4">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari nama…"
              className="pl-10"
            />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)} className="w-full sm:w-52">
              <option value="all">Semua Chapter</option>
              {chapters?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.displayName}
                </option>
              ))}
            </Select>
            <span className="text-xs text-ink-500">
              {rows.length} member
              {tanpaTanggal > 0 && `, ${tanpaTanggal} tanpa tanggal bergabung`}
            </span>
          </div>
        </div>

        {rows.length === 0 ? (
          <EmptyState title="Tidak ada member" description="Ubah saringan chapter atau kata kuncinya." />
        ) : (
          <Table>
            <THead>
              <Tr>
                <Th>Member</Th>
                <Th>Chapter</Th>
                <Th>Bergabung Sejak</Th>
                <Th>Lama Keanggotaan</Th>
                <Th>Jatuh Tempo</Th>
                <Th>Status</Th>
              </Tr>
            </THead>
            <TBody>
              {rows.map(({ m, bulan }) => (
                <Tr key={m.id}>
                  <Td className="font-medium text-ink-900">{m.name}</Td>
                  <Td className="text-ink-600">{m.chapter?.displayName ?? '—'}</Td>
                  <Td className="whitespace-nowrap text-ink-600">{formatDate(m.joinedDate)}</Td>
                  <Td className="whitespace-nowrap font-medium text-ink-900">{labelLama(bulan)}</Td>
                  <Td className="whitespace-nowrap text-ink-600">{formatDate(m.renewalDate)}</Td>
                  <Td>
                    <MemberStatusBadge status={m.status} />
                  </Td>
                </Tr>
              ))}
            </TBody>
          </Table>
        )}
      </Card>
    </div>
  )
}
