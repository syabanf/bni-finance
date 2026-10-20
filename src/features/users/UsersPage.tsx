import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { Pencil, ShieldCheck, Trash2, UserPlus } from 'lucide-react'
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
 * Pengelolaan akun: peran menentukan APA yang boleh dilakukan, lingkup
 * menentukan DI MANA.
 *
 * ST, MC, dan User masing-masing bisa Nasional (seluruh chapter) atau satu
 * chapter. Admin selalu nasional, karena rute khusus admin (pengaturan,
 * pengguna, sinkron, impor) bekerja lintas chapter. Lingkup ditegakkan server
 * pada setiap query; perubahan berlaku pada login berikutnya.
 */

const PERAN: { value: UserRole; deskripsi: string }[] = [
  { value: 'admin', deskripsi: 'Akses penuh, selalu nasional' },
  { value: 'st', deskripsi: 'Kelola invoice & pembayaran' },
  { value: 'mc', deskripsi: 'Lihat & jawab konfirmasi renewal' },
  { value: 'user', deskripsi: 'Hanya melihat & mengekspor' },
]

const bolehBerchapter = (role: UserRole) => role !== 'admin'

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
  const [diubah, setDiubah] = useState<ManagedUser | null>(null)

  const namaChapter = useMemo(() => {
    const map = new Map<string, string>()
    for (const c of chapters.data ?? []) map.set(c.id, c.displayName)
    return map
  }, [chapters.data])

  const daftarChapter = useMemo(
    () => (chapters.data ?? []).map((c) => ({ id: c.id, nama: c.displayName })),
    [chapters.data],
  )

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
        description="Kelola akun, perannya, dan lingkupnya: nasional atau satu chapter."
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
                  <Th>Lingkup</Th>
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
                        // hilang, padahal "nasional" adalah keadaan yang sah.
                        <span className="text-ink-400">Nasional</span>
                      )}
                    </Td>
                    <Td className="text-right">
                      <div className="inline-flex items-center gap-3">
                        <button
                          onClick={() => setDiubah(u)}
                          className="inline-flex items-center gap-1 text-sm text-ink-500 hover:text-ink-900"
                        >
                          <Pencil className="h-4 w-4" />
                          Ubah Peran
                        </button>
                        <button
                          onClick={() => hapus(u)}
                          className="inline-flex items-center gap-1 text-sm text-ink-500 hover:text-red-600"
                        >
                          <Trash2 className="h-4 w-4" />
                          Hapus
                        </button>
                      </div>
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
        chapters={daftarChapter}
        onClose={() => setBukaForm(false)}
        onSaved={() => {
          setBukaForm(false)
          users.reload()
        }}
      />

      <FormUbahPeran
        user={diubah}
        chapters={daftarChapter}
        onClose={() => setDiubah(null)}
        onSaved={() => {
          setDiubah(null)
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

  const simpan = async (e: FormEvent) => {
    e.preventDefault()
    setMenyimpan(true)
    try {
      await userService.create({
        email: email.trim(),
        name: nama.trim(),
        password: sandi,
        role,
        chapterId: bolehBerchapter(role) ? chapterId || null : null,
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

        <PilihPeranLingkup
          role={role}
          chapterId={chapterId}
          chapters={chapters}
          onRole={setRole}
          onChapter={setChapterId}
        />

        <div className="flex items-start gap-2 rounded-lg bg-ink-50 p-3 text-xs text-ink-600">
          <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-ink-400" />
          <span>
            Batas chapter ditegakkan di server pada setiap query, bukan hanya dengan
            menyembunyikan tombol. Akun berchapter tidak melihat data chapter lain sama sekali.
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

/**
 * Peran dan lingkup, dipakai formulir tambah dan ubah. Lingkup hanya muncul
 * untuk peran yang boleh berchapter, dan dikosongkan saat berpindah ke admin:
 * nilai lama yang tersembunyi tetap terkirim dan ditolak server.
 */
function PilihPeranLingkup({
  role,
  chapterId,
  chapters,
  onRole,
  onChapter,
}: {
  role: UserRole
  chapterId: string
  chapters: { id: string; nama: string }[]
  onRole: (r: UserRole) => void
  onChapter: (id: string) => void
}) {
  return (
    <>
      <Field label="Peran">
        <Select
          value={role}
          onChange={(e) => {
            const baru = e.target.value as UserRole
            onRole(baru)
            if (!bolehBerchapter(baru)) onChapter('')
          }}
        >
          {PERAN.map((p) => (
            <option key={p.value} value={p.value}>
              {ROLE_LABEL[p.value]}: {p.deskripsi}
            </option>
          ))}
        </Select>
      </Field>

      {bolehBerchapter(role) && (
        <Field
          label="Lingkup"
          hint={
            chapterId
              ? `${ROLE_LABEL[role]} hanya melihat dan mengelola chapter ini.`
              : `${ROLE_LABEL[role]} melihat dan mengelola seluruh chapter.`
          }
        >
          <Select value={chapterId} onChange={(e) => onChapter(e.target.value)}>
            <option value="">Nasional (seluruh chapter)</option>
            {chapters.map((c) => (
              <option key={c.id} value={c.id}>
                {c.nama}
              </option>
            ))}
          </Select>
        </Field>
      )}
    </>
  )
}

function FormUbahPeran({
  user,
  chapters,
  onClose,
  onSaved,
}: {
  user: ManagedUser | null
  chapters: { id: string; nama: string }[]
  onClose: () => void
  onSaved: () => void
}) {
  const { toast } = useToast()
  const [role, setRole] = useState<UserRole>('user')
  const [chapterId, setChapterId] = useState('')
  const [menyimpan, setMenyimpan] = useState(false)

  // Diisi ulang dari akun yang dipilih setiap kali formulir dibuka.
  useEffect(() => {
    if (!user) return
    setRole(user.role)
    setChapterId(user.chapterId ?? '')
  }, [user])

  const simpan = async (e: FormEvent) => {
    e.preventDefault()
    if (!user) return
    setMenyimpan(true)
    try {
      await userService.changeRole(user.id, role, bolehBerchapter(role) ? chapterId || null : null)
      toast(`Peran ${user.email} diperbarui. Berlaku saat ia masuk lagi.`)
      onSaved()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal mengubah peran.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  return (
    <Modal open={!!user} onClose={onClose} title={user ? `Ubah Peran: ${user.name}` : 'Ubah Peran'}>
      <form onSubmit={simpan} className="space-y-4">
        <PilihPeranLingkup
          role={role}
          chapterId={chapterId}
          chapters={chapters}
          onRole={setRole}
          onChapter={setChapterId}
        />
        <p className="text-xs leading-relaxed text-ink-500">
          Peran dan lingkup dibawa di dalam sesi, jadi perubahan berlaku saat pengguna ini masuk lagi.
        </p>
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
