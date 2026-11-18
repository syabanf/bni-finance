import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ArrowRight, Eye, Search, Upload, UserPlus, Users } from 'lucide-react'
import type { Chapter, MemberFilters, MemberStatus } from '@/types'
import {
  Avatar,
  Button,
  Card,
  DateRangeFilter,
  EmptyState,
  ErrorState,
  ExportMenu,
  Input,
  MemberStatusBadge,
  PageHeader,
  Select,
  SummaryCard,
  TBody,
  THead,
  Table,
  TableSkeleton,
  Td,
  Th,
  Tr,
  useToast,
  Pagination,
  TampilanToggle,
  kelasTampilan,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { SyncButton } from '@/features/sync/SyncButton'
import { TambahMemberModal } from './components/TambahMemberModal'
import { useAuth } from '@/features/auth/AuthContext'
import { can } from '@/lib/rbac'
import { chapterService, memberService } from '@/services'
import { formatDate, formatDateTime } from '@/lib/format'
import { makeExportHandlers } from '@/lib/exporters'
import { useTampilan } from '@/hooks/useTampilan'
import { cn } from '@/lib/cn'
import { useHalamanServer, useTertunda } from '@/hooks/useHalamanServer'

/** Kelengkapan kontak — menentukan siapa yang bisa dijangkau Paper.id. */
type ContactFilter = 'all' | 'no-phone' | 'no-email' | 'no-contact'

const CONTACT_OPTIONS: { value: ContactFilter; label: string }[] = [
  { value: 'all', label: 'Semua Kontak' },
  { value: 'no-phone', label: 'Tanpa No. HP' },
  { value: 'no-email', label: 'Tanpa Email' },
  { value: 'no-contact', label: 'Tanpa HP & Email' },
]

/**
 * Member versus visitor, sebagai satu saringan kasar.
 *
 * Kartu status di atas tabel sudah membedakan empat status, tapi pertanyaan
 * yang sering diajukan lebih sederhana dari itu: siapa yang sudah anggota,
 * dan siapa yang masih dalam perjalanan ke sana. Visitor di saringan ini
 * mencakup tamu (status visitor) dan yang sedang menunggu pendaftarannya
 * diproses (pending).
 * Yang keanggotaannya lewat (inactive) tetap member: ia pernah bergabung,
 * dan yang ditagih darinya adalah perpanjangan, bukan pendaftaran.
 */
type Keanggotaan = 'all' | 'member' | 'visitor'

const KEANGGOTAAN_OPTIONS: { value: Keanggotaan; label: string }[] = [
  { value: 'all', label: 'Member & Visitor' },
  { value: 'member', label: 'Member' },
  { value: 'visitor', label: 'Visitor' },
]

const MEMBER_STATUS_LABEL: Record<string, string> = {
  active: 'Aktif',
  pending: 'Pending',
  inactive: 'Nonaktif',
  new_member: 'New Member',
}

export function MemberListPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  // Impor dijaga settings:manage di router dan admin-saja di server.
  const bolehImpor = can(user?.role, 'settings:manage')
  const { toast } = useToast()
  const { data: chapters } = useAsync<Chapter[]>(() => chapterService.list())

  const [searchParams] = useSearchParams()
  // Nilai awal dibaca dari ?q= — pencarian di topbar mengirimkannya ke sini.
  //
  // Tanpa ini, memilih "Member" di kotak pencarian membuka halaman ini dengan
  // kotak yang kosong: orangnya sudah mengetik, alamatnya sudah membawa
  // kata kuncinya, dan yang terlihat tetap seluruh daftar. Persis keluhan
  // yang membuat pencariannya disebut "belum berfungsi".
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [chapterId, setChapterId] = useState(searchParams.get('chapter') ?? 'all')
  const [hideNoDueDate, setHideNoDueDate] = useState(false)
  const [dueFrom, setDueFrom] = useState('')
  const [dueTo, setDueTo] = useState('')
  const [memberStatus, setMemberStatus] = useState<MemberStatus | 'all'>('all')
  const [contact, setContact] = useState<ContactFilter>('all')
  const [keanggotaan, setKeanggotaan] = useState<Keanggotaan>('all')
  const [tambahBuka, setTambahBuka] = useState(false)

  // Seluruh saringan dijalankan server. Sebelumnya 200 member pertama ditarik
  // lalu disaring di sini, jadi member ke-201 tidak pernah tampil dan hitungan
  // di kartu berhenti tumbuh di angka itu.
  const searchTertunda = useTertunda(search)
  const filterAktif = useMemo<MemberFilters>(
    () => ({
      search: searchTertunda,
      chapterId,
      status: memberStatus,
      kelompok: keanggotaan,
      kontak: contact,
      punyaJatuhTempo: hideNoDueDate,
      dueFrom,
      dueTo,
      summary: true,
    }),
    [searchTertunda, chapterId, memberStatus, keanggotaan, contact, hideNoDueDate, dueFrom, dueTo],
  )
  const { halaman, loading, error, reload, paginasi } = useHalamanServer(
    (f: MemberFilters) => memberService.listPaged(f),
    filterAktif,
  )
  const paginated = halaman?.rows ?? []
  const totalBaris = halaman?.total ?? 0

  // Hitungan per status untuk seluruh hasil saringan, tanpa saringan statusnya.
  const perStatus = halaman?.summary?.byStatus ?? {}
  const hitung = (s: MemberStatus) => perStatus[s] ?? 0
  const statusCounts = {
    all: Object.values(perStatus).reduce((a, n) => a + (n ?? 0), 0),
    active: hitung('active'),
    new_member: hitung('new_member'),
    pending: hitung('pending'),
    inactive: hitung('inactive'),
    visitor: hitung('visitor'),
  }

  const [tampilan, setTampilan] = useTampilan('member')
  const kelas = kelasTampilan(tampilan)

  // Ekspor memuat SELURUH hasil saringan, bukan halaman yang sedang tampil.
  const [mengekspor, setMengekspor] = useState(false)
  const ekspor = async (jenis: 'onExcel' | 'onPdf') => {
    setMengekspor(true)
    try {
      const semua = await memberService.list(filterAktif)
      makeExportHandlers({
        filename: 'member',
        title: 'Daftar Member',
        subtitle: `Status: ${memberStatus === 'all' ? 'Semua' : (MEMBER_STATUS_LABEL[memberStatus] ?? memberStatus)}`,
        meta: [`${semua.length} member`, `Dibuat ${formatDateTime(new Date())}`],
        columns: [
          { label: 'Nama' },
          { label: 'ID' },
          { label: 'Chapter' },
          { label: 'Email' },
          { label: 'Telepon' },
          { label: 'Status' },
          { label: 'Jatuh Tempo' },
        ],
        rows: semua.map((m) => [
          m.name,
          m.id,
          m.chapter?.displayName ?? '',
          m.email ?? '',
          m.phone ?? '',
          MEMBER_STATUS_LABEL[m.status] ?? m.status,
          m.renewalDate ? formatDate(m.renewalDate) : '',
        ]),
        onPopupBlocked: () => toast('Izinkan popup di browser untuk mengekspor PDF.', 'error'),
      })[jenis]()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal mengekspor member.', 'error')
    } finally {
      setMengekspor(false)
    }
  }

  return (
    <div>
      <PageHeader
        title="Member"
        description="Data member dari impor laporan Business Network International Connect dan Tambah Member."
        action={
          <div className="flex flex-wrap items-center gap-2">
            {/*
              Satu tombol yang mengikuti tab yang sedang dibuka.
              
              Versi sebelumnya hanya menampilkannya di tab Visitor, dengan alasan
              tombol keempat yang permanen akan kabur bedanya dari impor member
              biasa. Alasan itu benar soal kaburnya, dan salah soal
              menyembunyikannya: orang yang berdiri di tab Member dengan berkas
              di tangan tidak punya jalan masuk sama sekali, dan tab Visitor
              bukan tempat yang terpikir untuk dicari.
              
              Bedanya dijaga di tujuannya, bukan di ada-tidaknya tombol. Dari tab
              Visitor ia membawa `status=visitor`, yang memberi status itu pada
              baris yang kolom statusnya kosong. Dari tab lain ia tidak membawa
              apa pun, dan labelnya ikut berganti supaya yang menekan tahu mana
              yang sedang ia lakukan.
              
              Chapter yang sedang disaring ikut terbawa, jadi daftar yang dilihat
              di layar dan lingkup yang dituju impor adalah hal yang sama.
            */}
            {bolehImpor && (
              <Button
                variant="secondary"
                onClick={() => {
                  const q = new URLSearchParams()
                  if (memberStatus === 'visitor') q.set('status', 'visitor')
                  if (chapterId !== 'all') q.set('chapter', chapterId)
                  const qs = q.toString()
                  navigate(`/settings/import${qs ? `?${qs}` : ''}`)
                }}
              >
                <Upload className="h-4 w-4" />
                {memberStatus === 'visitor' ? 'Impor Visitor' : 'Impor Member'}
              </Button>
            )}
            {bolehImpor && (
              <Button onClick={() => setTambahBuka(true)}>
                <UserPlus className="h-4 w-4" />
                Tambah Member
              </Button>
            )}
            <SyncButton jenis="member" onSelesai={reload} />
            <ExportMenu
              onExcel={() => void ekspor('onExcel')}
              onPdf={() => void ekspor('onPdf')}
              disabled={totalBaris === 0 || mengekspor}
            />
          </div>
        }
      />

      {/* Summary cards (also filter by status) */}
      <div className="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <SummaryCard
          label="Total Member"
          value={statusCounts.all}
          tone="brand"
          active={memberStatus === 'all'}
          onClick={() => setMemberStatus('all')}
        />
        <SummaryCard
          label="Active"
          value={statusCounts.active}
          tone="green"
          active={memberStatus === 'active'}
          onClick={() => setMemberStatus('active')}
        />
        <SummaryCard
          label="New Member"
          value={statusCounts.new_member}
          tone="purple"
          active={memberStatus === 'new_member'}
          onClick={() => setMemberStatus('new_member')}
        />
        <SummaryCard
          label="Pending"
          value={statusCounts.pending}
          tone="amber"
          active={memberStatus === 'pending'}
          onClick={() => setMemberStatus('pending')}
        />
        <SummaryCard
          label="Inactive"
          value={statusCounts.inactive}
          tone="default"
          active={memberStatus === 'inactive'}
          onClick={() => setMemberStatus('inactive')}
        />
        <SummaryCard
          label="Visitor"
          value={statusCounts.visitor}
          tone="blue"
          active={memberStatus === 'visitor'}
          onClick={() => setMemberStatus('visitor')}
        />
      </div>

      <Card data-tour="member-table">
        <div data-tour="member-filters" className="space-y-3 border-b border-ink-100 p-4">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari nama, ID, atau email…"
              className="pl-10"
            />
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <TampilanToggle value={tampilan} onChange={setTampilan} />
            <Select
              aria-label="Filter keanggotaan"
              value={keanggotaan}
              onChange={(e) => setKeanggotaan(e.target.value as Keanggotaan)}
              className="w-full sm:w-44"
            >
              {KEANGGOTAAN_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
            <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)} className="w-full sm:w-52" aria-label="Filter chapter">
              <option value="all">Semua Chapter</option>
              {chapters?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.displayName}
                </option>
              ))}
            </Select>
            <Select
              value={contact}
              onChange={(e) => setContact(e.target.value as ContactFilter)}
              className="w-full sm:w-44"
              aria-label="Filter kelengkapan kontak"
            >
              {CONTACT_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
            {/* Filter jatuh tempo (rentang tanggal renewal) */}
            <DateRangeFilter
              label="Jatuh tempo"
              from={dueFrom}
              to={dueTo}
              onFrom={setDueFrom}
              onTo={setDueTo}
              onReset={() => {
                setDueFrom('')
                setDueTo('')
              }}
            />
            <label className="flex shrink-0 cursor-pointer items-center gap-2 text-sm text-ink-600">
              <input
                type="checkbox"
                checked={hideNoDueDate}
                onChange={(e) => setHideNoDueDate(e.target.checked)}
                className="h-4 w-4 rounded border-ink-300 accent-brand-500"
              />
              Ada jatuh tempo
            </label>
          </div>
        </div>

        {error ? (
          <ErrorState message={error} onRetry={reload} />
        ) : loading && !halaman ? (
          <TableSkeleton rows={8} cols={5} />
        ) : totalBaris === 0 ? (
          <EmptyState icon={Users} title="Tidak ada member" description="Tidak ada member yang cocok dengan filter." />
        ) : (
          <>
            {/* Kartu */}
            <div className={cn('divide-y divide-ink-100', kelas.kartu)}>
              {paginated.map((m) => (
                <div
                  key={m.id}
                  onClick={() => navigate(`/members/${m.id}`)}
                  className="flex items-center gap-3 px-4 py-3.5 active:bg-ink-50"
                >
                  <Avatar name={m.name} size="sm" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="truncate font-medium text-ink-900 text-sm">{m.name}</div>
                        <div className="text-xs text-ink-400 truncate">
                          {m.company && m.businessField
                            ? `${m.company} · ${m.businessField}`
                            : m.company ?? m.businessField ?? m.chapter?.displayName ?? '—'}
                        </div>
                      </div>
                      <div className="shrink-0 text-right">
                        <MemberStatusBadge status={m.status} />
                        <div className="text-xs text-ink-400 mt-1">
                          {m.renewalDate ? formatDate(m.renewalDate) : '—'}
                        </div>
                      </div>
                    </div>
                    {m.email && (
                      <div className="mt-1 truncate text-xs text-ink-400">{m.email}</div>
                    )}
                  </div>
                  <ArrowRight className="h-4 w-4 shrink-0 text-ink-300" />
                </div>
              ))}
            </div>

            {/* Tabel */}
            <div className={kelas.tabel}>
              <Table>
                <THead>
                  <Tr>
                    <Th>Member</Th>
                    <Th>Chapter</Th>
                    <Th>Email</Th>
                    <Th>Status</Th>
                    <Th>Jatuh Tempo</Th>
                    <Th className="text-right">Aksi</Th>
                  </Tr>
                </THead>
                <TBody>
                  {paginated.map((m) => (
                    <Tr key={m.id} onClick={() => navigate(`/members/${m.id}`)}>
                      <Td>
                        <div className="flex items-center gap-3">
                          <Avatar name={m.name} size="sm" />
                          <div className="leading-tight">
                            <div className="font-medium text-ink-900">{m.name}</div>
                            <div className="text-xs text-ink-400">
                              {m.company && m.businessField
                                ? `${m.company} · ${m.businessField}`
                                : m.company ?? m.businessField ?? '—'}
                            </div>
                          </div>
                        </div>
                      </Td>
                      <Td className="text-ink-600">{m.chapter?.displayName ?? '—'}</Td>
                      <Td className="text-ink-600">{m.email ?? '—'}</Td>
                      <Td>
                        <MemberStatusBadge status={m.status} />
                      </Td>
                      <Td className="whitespace-nowrap text-ink-600">
                        {m.renewalDate ? formatDate(m.renewalDate) : '—'}
                      </Td>
                      <Td className="text-right">
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            navigate(`/members/${m.id}`)
                          }}
                          className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-ink-400 transition-colors hover:bg-ink-100 hover:text-brand-500"
                          aria-label="Lihat detail"
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                      </Td>
                    </Tr>
                  ))}
                </TBody>
              </Table>
            </div>
            <Pagination {...paginasi} satuan="member" />
          </>
        )}
      </Card>
      <TambahMemberModal
        open={tambahBuka}
        onClose={() => setTambahBuka(false)}
        onDibuat={reload}
        chapters={chapters ?? []}
        chapterAwal={chapterId !== 'all' ? chapterId : undefined}
      />
    </div>
  )
}
