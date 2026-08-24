import { useMemo, useState, type FormEvent } from 'react'
import { ShieldCheck, Trash2, UserPlus } from 'lucide-react'
import type { ManagedUser, UserRole } from '@/types'
import {
  Badge,
  Button,
  Card,
  CardBody,
  CardHeader,
  EmptyState,
  ErrorState,
  Field,
  Input,
  LoadingState,
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
import { chapterService, userService } from '@/services'
import { ROLE_LABEL } from '@/lib/rbac'

/**
 * Pengelolaan akun — satu-satunya cara membuat peran ST dan MC dari aplikasi.
 *
 * Halaman ini ada karena tanpanya alur berlingkup chapter tidak bisa dipakai
 * siapa pun: backend sudah membatasi ST ke chapternya dan sudah menolak ST
 * tanpa chapter, tapi akun yang memakai aturan itu hanya bisa dibuat lewat
 * curl.
 */

const PERAN: { value: UserRole; deskripsi: string; berlingkup: boolean }[] = [
  { value: 'admin', deskripsi: 'Akses penuh, seluruh chapter', berlingkup: false },
  { value: 'st', deskripsi: 'Kelola invoice & pembayaran, HANYA chapternya', berlingkup: true },
  { value: 'mc', deskripsi: 'Lihat & jawab konfirmasi renewal, HANYA chapternya', berlingkup: true },
  { value: 'user', deskripsi: 'Hanya melihat & mengekspor, seluruh chapter', berlingkup: false },
]

function berlingkupChapter(role: UserRole) {
  return role === 'st' || role === 'mc'
}

const WARNA_PERAN: Record<UserRole, 'red' | 'amber' | 'green' | 'gray'> = {
  admin: 'red',
  st: 'amber',
  mc: 'green',
  user: 'gray',
}

export function UsersPage() {
  const { toast } = useToast()
  const users = useAsync(() => userService.list(), [])
  const chapters = useAsync(() => chapterService.list(), [])
  const [bukaForm, setBukaForm] = useState(false)

  const namaChapter = useMemo(() => {
    const map = new Map<string, string>()
    for (const c of chapters.data ?? []) map.set(c.id, c.displayName)
    return map
  }, [chapters.data])

  const hapus = async (u: ManagedUser) => {
    // Konfirmasi dulu: menghapus akun tidak bisa dibatalkan, dan orang yang
    // kehilangan akunnya tidak bisa masuk untuk memberi tahu.
    if (!window.confirm(`Hapus akun ${u.email}? Tindakan ini tidak bisa dibatalkan.`)) return
    try {
      await userService.remove(u.id)
      toast(`Akun ${u.email} dihapus.`)
      users.reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menghapus akun.', 'error')
    }
  }

  return (
    <div>
      <PageHeader
        title="Pengguna"
        description="Kelola akun dan perannya. ST dan MC terikat pada satu chapter."
        action={
          <Button onClick={() => setBukaForm(true)}>
            <UserPlus className="h-4 w-4" />
            Tambah Pengguna
          </Button>
        }
      />

      <Card>
        <CardHeader
          title="Daftar akun"
          subtitle="Peran menentukan APA yang boleh dilakukan; chapter menentukan DI MANA."
        />
        <CardBody>
          {users.loading ? (
            <LoadingState />
          ) : users.error ? (
            <ErrorState message={users.error} onRetry={users.reload} />
          ) : !users.data?.length ? (
            <EmptyState title="Belum ada akun" description="Tambah pengguna untuk mulai." />
          ) : (
            <Table>
              <THead>
                <Tr>
                  <Th>Nama</Th>
                  <Th>Email</Th>
                  <Th>Peran</Th>
                  <Th>Chapter</Th>
                  <Th className="text-right">Aksi</Th>
                </Tr>
              </THead>
              <TBody>
                {users.data.map((u) => (
                  <Tr key={u.id}>
                    <Td className="font-medium text-ink-900">{u.name}</Td>
                    <Td className="text-ink-600">{u.email}</Td>
                    <Td>
                      <Badge tone={WARNA_PERAN[u.role]}>{ROLE_LABEL[u.role]}</Badge>
                    </Td>
                    <Td className="text-ink-600">
                      {u.chapterId ? (
                        namaChapter.get(u.chapterId) ?? u.chapterId
                      ) : (
                        // Bukan sel kosong: kosong terbaca sebagai data yang
                        // hilang, padahal "nasional" adalah keadaan yang benar
                        // untuk admin dan user.
                        <span className="text-ink-400">Nasional</span>
                      )}
                    </Td>
                    <Td className="text-right">
                      <button
                        onClick={() => hapus(u)}
                        className="inline-flex items-center gap-1 text-sm text-ink-500 hover:text-red-600"
                      >
                        <Trash2 className="h-4 w-4" />
                        Hapus
                      </button>
                    </Td>
                  </Tr>
                ))}
              </TBody>
            </Table>
          )}
        </CardBody>
      </Card>

      <FormTambah
        open={bukaForm}
        chapters={(chapters.data ?? []).map((c) => ({ id: c.id, nama: c.displayName }))}
        onClose={() => setBukaForm(false)}
        onSaved={() => {
          setBukaForm(false)
          users.reload()
        }}
      />
    </div>
  )
}

function FormTambah({
  open,
  chapters,
  onClose,
  onSaved,
}: {
  open: boolean
  chapters: { id: string; nama: string }[]
  onClose: () => void
  onSaved: () => void
}) {
  const { toast } = useToast()
  const [email, setEmail] = useState('')
  const [nama, setNama] = useState('')
  const [sandi, setSandi] = useState('')
  const [role, setRole] = useState<UserRole>('st')
  const [chapterId, setChapterId] = useState('')
  const [menyimpan, setMenyimpan] = useState(false)

  const perluChapter = berlingkupChapter(role)

  const simpan = async (e: FormEvent) => {
    e.preventDefault()
    // Diperiksa di sini juga, meski backend menolaknya: pesan yang muncul
    // sebelum permintaan dikirim jauh lebih jelas daripada 400 yang datang
    // beberapa detik kemudian.
    if (perluChapter && !chapterId) {
      toast(`Peran ${ROLE_LABEL[role]} wajib punya chapter.`, 'error')
      return
    }
    setMenyimpan(true)
    try {
      await userService.create({
        email: email.trim(),
        name: nama.trim(),
        password: sandi,
        role,
        chapterId: perluChapter ? chapterId : null,
      })
      toast(`Akun ${email.trim()} dibuat.`)
      setEmail('')
      setNama('')
      setSandi('')
      onSaved()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal membuat akun.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Tambah Pengguna">
      <form onSubmit={simpan} className="space-y-4">
        <Field label="Nama">
          <Input value={nama} onChange={(e) => setNama(e.target.value)} required placeholder="Nama lengkap" />
        </Field>
        <Field label="Email">
          <Input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            placeholder="nama@bni-finance.com"
          />
        </Field>
        <Field label="Kata sandi" hint="Minimal 6 karakter.">
          <Input
            type="password"
            value={sandi}
            onChange={(e) => setSandi(e.target.value)}
            required
            minLength={6}
            placeholder="••••••••"
          />
        </Field>

        <Field label="Peran">
          <Select
            value={role}
            onChange={(e) => {
              const baru = e.target.value as UserRole
              setRole(baru)
              // Chapter dikosongkan saat berpindah ke peran nasional. Kalau
              // dibiarkan, nilai lamanya ikut terkirim dan backend menolaknya
              // dengan "peran admin tidak boleh punya chapterId" — pesan yang
              // membingungkan karena kolomnya bahkan tidak terlihat lagi.
              if (!berlingkupChapter(baru)) setChapterId('')
            }}
          >
            {PERAN.map((p) => (
              <option key={p.value} value={p.value}>
                {ROLE_LABEL[p.value]} — {p.deskripsi}
              </option>
            ))}
          </Select>
        </Field>

        {perluChapter && (
          <Field label="Chapter" hint={`${ROLE_LABEL[role]} hanya melihat dan mengelola chapter ini.`}>
            <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)} required>
              <option value="">Pilih chapter…</option>
              {chapters.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.nama}
                </option>
              ))}
            </Select>
          </Field>
        )}

        <div className="flex items-start gap-2 rounded-lg bg-ink-50 p-3 text-xs text-ink-600">
          <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-ink-400" />
          <span>
            Batas chapter ditegakkan di server, pada setiap query — bukan hanya dengan
            menyembunyikan tombol. ST chapter lain tidak akan melihat data chapter ini sama sekali.
          </span>
        </div>

        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" variant="outline" onClick={onClose}>
            Batal
          </Button>
          <Button type="submit" loading={menyimpan}>
            Simpan
          </Button>
        </div>
      </form>
    </Modal>
  )
}
