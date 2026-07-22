import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowRight, Eye, EyeOff, Lock, Mail, ShieldCheck, UserCog, UserRound } from 'lucide-react'
import type { UserRole } from '@/types'
import { BniLogo, Button, Field, Input } from '@/components/ui'
import { useAuth } from './AuthContext'
import { cn } from '@/lib/cn'

const useMock = import.meta.env.VITE_USE_MOCK !== 'false'

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

/**
 * Di mode Supabase, quick sign-in butuh akun nyata — isi lewat env
 * (VITE_DEMO_EMAIL / VITE_DEMO_PASSWORD), mis. di Vercel preview.
 * ⚠️ Nilai VITE_* ikut ter-bundle ke klien, jadi arahkan hanya ke akun demo.
 */
const ENV_DEMO_EMAIL = import.meta.env.VITE_DEMO_EMAIL || ''
const ENV_DEMO_PASSWORD = import.meta.env.VITE_DEMO_PASSWORD || ''
const ENV_QUICK_LOGIN = Boolean(ENV_DEMO_EMAIL && ENV_DEMO_PASSWORD)

export function LoginPage() {
  const { login, user, loading: authLoading } = useAuth()
  const navigate = useNavigate()

  const [email, setEmail] = useState(useMock ? DEMO_ROLES[0].email : '')
  const [password, setPassword] = useState(useMock ? DEMO_ROLES[0].password : '')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [quickRole, setQuickRole] = useState<string | null>(null)

  if (!authLoading && user) {
    navigate('/dashboard', { replace: true })
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

          {/* Quick sign-in */}
          {(useMock || ENV_QUICK_LOGIN) && (
            <>
              <div className="my-5 flex items-center gap-3 text-xs text-ink-400">
                <span className="h-px flex-1 bg-ink-100" />
                masuk cepat
                <span className="h-px flex-1 bg-ink-100" />
              </div>

              {useMock ? (
                <div className="grid grid-cols-2 gap-3">
                  {DEMO_ROLES.map((r) => {
                    const Icon = r.icon
                    return (
                      <button
                        key={r.role}
                        type="button"
                        disabled={busy}
                        onClick={() => quickLogin(r.role, r.email, r.password)}
                        className={cn(
                          'flex flex-col items-center gap-1 rounded-xl border border-ink-200 px-3 py-3 transition-colors',
                          'hover:border-brand-300 hover:bg-brand-50/50 disabled:cursor-not-allowed disabled:opacity-60',
                          quickRole === r.role && 'border-brand-400 bg-brand-50',
                        )}
                      >
                        <Icon className="h-5 w-5 text-brand-500" />
                        <span className="text-sm font-semibold text-ink-900">
                          {quickRole === r.role ? 'Masuk…' : r.label}
                        </span>
                        <span className="text-[11px] text-ink-400">{r.desc}</span>
                      </button>
                    )
                  })}
                </div>
              ) : (
                <Button
                  type="button"
                  variant="outline"
                  size="lg"
                  loading={quickRole === 'env'}
                  disabled={busy && quickRole !== 'env'}
                  onClick={() => quickLogin('env', ENV_DEMO_EMAIL, ENV_DEMO_PASSWORD)}
                  className="w-full"
                >
                  {quickRole !== 'env' && <UserCog className="h-4 w-4" />}
                  Masuk sebagai akun demo
                </Button>
              )}

              <p className="mt-4 text-center text-xs text-ink-400">
                {useMock
                  ? 'Mode demo — data berjalan di atas mock repository.'
                  : `Akun demo: ${ENV_DEMO_EMAIL}`}
              </p>
            </>
          )}
        </div>

        <div className="mt-6 flex items-center justify-center gap-1.5 text-xs text-ink-400">
          <ShieldCheck className="h-3.5 w-3.5" />
          Akses internal BNI · Terenkripsi
        </div>
      </div>
    </div>
  )
}
