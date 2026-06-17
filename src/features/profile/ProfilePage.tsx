import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { LogOut } from 'lucide-react'
import { Avatar, Button, Card, CardHeader, Field, Input, PageHeader, useToast } from '@/components/ui'
import { useAuth } from '@/features/auth/AuthContext'

const useMock = import.meta.env.VITE_USE_MOCK !== 'false'

export function ProfilePage() {
  const { user, updateProfile, updatePassword, logout } = useAuth()
  const { toast } = useToast()
  const navigate = useNavigate()

  const [name, setName] = useState(user?.name ?? '')
  const [savingName, setSavingName] = useState(false)
  const [pw, setPw] = useState('')
  const [pw2, setPw2] = useState('')
  const [savingPw, setSavingPw] = useState(false)

  const nameChanged = name.trim().length > 0 && name.trim() !== (user?.name ?? '')

  const saveName = async () => {
    setSavingName(true)
    try {
      await updateProfile(name.trim())
      toast('Profil berhasil disimpan.')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan profil.', 'error')
    } finally {
      setSavingName(false)
    }
  }

  const savePassword = async () => {
    if (pw.length < 6) return toast('Kata sandi minimal 6 karakter.', 'error')
    if (pw !== pw2) return toast('Konfirmasi kata sandi tidak cocok.', 'error')
    setSavingPw(true)
    try {
      await updatePassword(pw)
      setPw('')
      setPw2('')
      toast('Kata sandi berhasil diperbarui.')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal memperbarui kata sandi.', 'error')
    } finally {
      setSavingPw(false)
    }
  }

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader title="Profil Saya" description="Kelola informasi akun dan keamanan." />

      {/* Identity */}
      <Card className="mb-5">
        <div className="flex items-center gap-4 p-5">
          <Avatar name={user?.name ?? 'Admin'} size="lg" />
          <div className="min-w-0">
            <div className="text-lg font-semibold text-ink-900">{user?.name ?? 'Admin'}</div>
            <div className="truncate text-sm text-ink-500">{user?.email}</div>
            <span className="mt-1.5 inline-flex items-center rounded-full bg-brand-50 px-2.5 py-1 text-xs font-medium text-brand-600 ring-1 ring-inset ring-brand-600/10">
              National Admin
            </span>
          </div>
        </div>
      </Card>

      {/* Account info */}
      <Card className="mb-5">
        <CardHeader title="Informasi Akun" />
        <div className="space-y-4 border-t border-ink-100 p-5">
          <Field label="Nama">
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Nama lengkap" />
          </Field>
          <Field label="Email" hint="Hubungi administrator untuk mengubah email.">
            <Input value={user?.email ?? ''} disabled />
          </Field>
          <div className="flex justify-end">
            <Button onClick={saveName} loading={savingName} disabled={!nameChanged}>
              Simpan Perubahan
            </Button>
          </div>
        </div>
      </Card>

      {/* Security */}
      <Card className="mb-5">
        <CardHeader title="Ubah Kata Sandi" />
        <div className="space-y-4 border-t border-ink-100 p-5">
          {useMock ? (
            <p className="text-sm text-ink-500">
              Penggantian kata sandi tersedia pada mode produksi (Supabase Auth).
            </p>
          ) : (
            <>
              <Field label="Kata Sandi Baru">
                <Input
                  type="password"
                  value={pw}
                  onChange={(e) => setPw(e.target.value)}
                  placeholder="Minimal 6 karakter"
                  autoComplete="new-password"
                />
              </Field>
              <Field label="Konfirmasi Kata Sandi">
                <Input
                  type="password"
                  value={pw2}
                  onChange={(e) => setPw2(e.target.value)}
                  autoComplete="new-password"
                />
              </Field>
              <div className="flex justify-end">
                <Button onClick={savePassword} loading={savingPw} disabled={!pw || !pw2}>
                  Perbarui Kata Sandi
                </Button>
              </div>
            </>
          )}
        </div>
      </Card>

      <div className="flex justify-end">
        <Button variant="outline" onClick={handleLogout}>
          <LogOut className="h-4 w-4" />
          Keluar
        </Button>
      </div>
    </div>
  )
}
