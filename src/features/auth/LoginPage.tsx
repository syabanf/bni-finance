import { useEffect, useState, type FormEvent } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { ArrowRight, Eye, EyeOff, Lock, Mail, ShieldCheck, UserCog, UserRound } from 'lucide-react'
import type { UserRole } from '@/types'
import { BniLogo, Button, Field, Input } from '@/components/ui'
import { useAuth } from './AuthContext'
import { cn } from '@/lib/cn'
import { DATA_SOURCE_LABEL, getDataSource, isMockMode, setDataSource } from '@/services/dataSource'
import { listQuickLoginAccounts, type QuickLoginAccount } from '@/services/api/quickLoginService'

const useMock = isMockMode()
const dataSource = getDataSource()

/** Akun demo per peran — cocok dengan DEMO_ACCOUNTS di mock authRepository. */
const DEMO_ROLES: {
  role: UserRole
  label: string
  desc: string
  email: string
  password: string
  icon: typeof UserCog
}[] = [
  {
    role: 'admin',
    label: 'Admin',
    desc: 'Akses penuh',
    email: 'admin@bni-finance.com',
    password: 'admin123',
    icon: UserCog,
  },
  {
    role: 'user',
    label: 'User',
    desc: 'Lihat & ekspor',
    email: 'user@bni-finance.com',
    password: 'user123',
    icon: UserRound,
  },
]

/** Ikon & keterangan per peran, dipakai kartu mock maupun kartu mode API. */
const ROLE_ICON: Record<UserRole, typeof UserCog> = { admin: UserCog, user: UserRound }
const ROLE_DESC: Record<UserRole, string> = { admin: 'Akses penuh', user: 'Lihat & ekspor' }

export function LoginPage() {
  const { login, quickLogin: quickLoginAs, user, loading: authLoading } = useAuth()
  const navigate = useNavigate()

  const [email, setEmail] = useState(useMock ? DEMO_ROLES[0].email : '')
  const [password, setPassword] = useState(useMock ? DEMO_ROLES[0].password : '')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [quickRole, setQuickRole] = useState<string | null>(null)
  // Akun quick login mode API datang dari server — kata sandinya tidak pernah
  // sampai ke browser. Daftar kosong berarti fiturnya tidak diaktifkan.
  const [apiAccounts, setApiAccounts] = useState<QuickLoginAccount[]>([])

  useEffect(() => {
    if (useMock) return
    let cancelled = false
    listQuickLoginAccounts()
      .then((rows) => {
        if (!cancelled) setApiAccounts(rows)
      })
      .catch(() => {
        // Fitur mati atau backend tak terjangkau — cukup tampil tanpa tombol.
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Redirect lewat elemen, bukan navigate() di dalam render — memanggilnya saat
  // render memperbarui router sementara komponen ini masih dirender, dan React
  // memperingatkannya.
  if (!authLoading && user) {
    return <Navigate to="/dashboard" replace />
  }

  const doLogin = async (e: string, p: string, done: (busy: boolean) => void) => {
    setError(null)
    done(true)
    try {
      await login(e, p)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal masuk.')
      done(false)
    }
  }

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    doLogin(email, password, setLoading)
  }

  const quickLogin = (key: string, e: string, p: string) =>
    doLogin(e, p, (busy) => setQuickRole(busy ? key : null))

  /** Mode API: server memegang kredensialnya, kita hanya menyebut emailnya. */
  const quickLoginApi = async (account: QuickLoginAccount) => {
    setError(null)
    setQuickRole(account.email)
    try {
      await quickLoginAs(account.email)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal masuk cepat.')
      setQuickRole(null)
    }
  }

  const busy = loading || quickRole !== null

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-ink-50 px-4 py-10">
      {/* Latar dekoratif */}
      <div className="pointer-events-none absolute inset-0">
        <div className="absolute -left-32 -top-32 h-[28rem] w-[28rem] rounded-full bg-brand-500/15 blur-3xl" />
        <div className="absolute -bottom-40 -right-24 h-[26rem] w-[26rem] rounded-full bg-brand-700/10 blur-3xl" />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,transparent_35%,rgba(248,250,252,.9))]" />
      </div>

      <div className="relative w-full max-w-md">
        {/* Brand */}
        <div className="mb-6 flex flex-col items-center text-center">
          <span className="flex items-center rounded-2xl bg-white px-4 py-2.5 shadow-card">
            <BniLogo className="h-8 w-auto" />
          </span>
          <h1 className="mt-4 text-2xl font-bold tracking-tight text-ink-900">Finance Hub</h1>
          <p className="mt-1 text-sm text-ink-500">
            Invoice &amp; pembayaran keanggotaan BNI Grow Chapter
          </p>
        </div>

        {/* Kartu form */}
        <div className="rounded-2xl border border-ink-100 bg-white p-6 shadow-card-hover sm:p-7">
          <form onSubmit={handleSubmit} className="space-y-4">
            <Field label="Email">
              <div className="relative">
                <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="nama@bni-finance.com"
                  className="pl-10"
                  autoComplete="email"
                />
              </div>
            </Field>

            <Field label="Password">
              <div className="relative">
                <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
                <Input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="pl-10 pr-10"
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  aria-label={showPassword ? 'Sembunyikan password' : 'Tampilkan password'}
                  className="absolute right-2 top-1/2 -translate-y-1/2 rounded-lg p-1.5 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-700"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </Field>

            {error && (
              <div className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{error}</div>
            )}

            <Button type="submit" size="lg" loading={loading} disabled={busy && !loading} className="w-full">
              Masuk
              {!loading && <ArrowRight className="h-4 w-4" />}
            </Button>
          </form>

          {/* Masuk cepat — kartu mock, atau akun yang diizinkan server. */}
          {(useMock || apiAccounts.length > 0) && (
            <>
              <div className="my-5 flex items-center gap-3 text-xs text-ink-400">
                <span className="h-px flex-1 bg-ink-100" />
                masuk cepat
                <span className="h-px flex-1 bg-ink-100" />
              </div>

              <div className="grid grid-cols-2 gap-3">
                {useMock
                  ? DEMO_ROLES.map((r) => (
                      <QuickCard
                        key={r.role}
                        icon={r.icon}
                        title={r.label}
                        subtitle={r.desc}
                        active={quickRole === r.role}
                        disabled={busy}
                        onClick={() => quickLogin(r.role, r.email, r.password)}
                      />
                    ))
                  : apiAccounts.map((a) => (
                      <QuickCard
                        key={a.email}
                        icon={ROLE_ICON[a.role] ?? UserRound}
                        title={a.name}
                        subtitle={ROLE_DESC[a.role] ?? a.email}
                        active={quickRole === a.email}
                        disabled={busy}
                        onClick={() => void quickLoginApi(a)}
                      />
                    ))}
              </div>

              <p className="mt-4 text-center text-xs text-ink-400">
                {useMock
                  ? 'Mode demo — data berjalan di atas mock repository.'
                  : 'Akun uji yang diizinkan server — kata sandinya tidak pernah dikirim ke browser.'}
              </p>
            </>
          )}
        </div>

        {/* Pemilih sumber data juga ada di sini, bukan hanya di Pengaturan:
            kalau backend mati saat mode API, halaman login adalah satu-satunya
            layar yang bisa dijangkau — tanpa ini pengguna terkunci. */}
        <div className="mt-6 flex items-center justify-center gap-2 text-xs">
          <span className="text-ink-400">Sumber data</span>
          <div className="inline-flex overflow-hidden rounded-lg border border-ink-200 bg-white">
            {(['mock', 'api'] as const).map((mode) => (
              <button
                key={mode}
                type="button"
                aria-pressed={dataSource === mode}
                onClick={() => dataSource !== mode && setDataSource(mode)}
                className={`px-2.5 py-1 transition ${
                  dataSource === mode
                    ? 'bg-brand-500 font-medium text-white'
                    : 'text-ink-500 hover:bg-ink-50'
                }`}
              >
                {DATA_SOURCE_LABEL[mode]}
              </button>
            ))}
          </div>
        </div>

        <div className="mt-4 flex items-center justify-center gap-1.5 text-xs text-ink-400">
          <ShieldCheck className="h-3.5 w-3.5" />
          Akses internal BNI · Terenkripsi
        </div>
      </div>
    </div>
  )
}

function QuickCard({
  icon: Icon,
  title,
  subtitle,
  active,
  disabled,
  onClick,
}: {
  icon: typeof UserCog
  title: string
  subtitle: string
  active: boolean
  disabled: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={cn(
        'flex flex-col items-center gap-1 rounded-xl border border-ink-200 px-3 py-3 transition-colors',
        'hover:border-brand-300 hover:bg-brand-50/50 disabled:cursor-not-allowed disabled:opacity-60',
        active && 'border-brand-400 bg-brand-50',
      )}
    >
      <Icon className="h-5 w-5 text-brand-500" />
      <span className="max-w-full truncate text-sm font-semibold text-ink-900">
        {active ? 'Masuk…' : title}
      </span>
      <span className="max-w-full truncate text-[11px] text-ink-400">{subtitle}</span>
    </button>
  )
}
