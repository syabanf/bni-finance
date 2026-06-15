import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowRight, Lock, Mail, ShieldCheck } from 'lucide-react'
import { Button, Field, Input } from '@/components/ui'
import { useAuth } from './AuthContext'

const useMock = import.meta.env.VITE_USE_MOCK !== 'false'

export function LoginPage() {
  const { login, user } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState(useMock ? 'admin@bni-finance.com' : '')
  const [password, setPassword] = useState(useMock ? 'admin123' : '')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  if (user) {
    navigate('/dashboard', { replace: true })
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await login(email, password)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal masuk.')
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen bg-ink-50">
      {/* Brand panel */}
      <div className="relative hidden w-1/2 flex-col justify-between overflow-hidden bg-gradient-to-br from-brand-600 via-brand-500 to-brand-700 p-12 text-white lg:flex">
        <div className="absolute -right-24 -top-24 h-96 w-96 rounded-full bg-white/10 blur-2xl" />
        <div className="absolute -bottom-32 -left-16 h-96 w-96 rounded-full bg-black/10 blur-2xl" />

        <div className="relative flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-white/15 text-xl font-extrabold backdrop-blur">
            B
          </div>
          <div className="leading-tight">
            <div className="text-[11px] font-bold uppercase tracking-wider text-white/80">
              BNI Indonesia
            </div>
            <div className="text-lg font-bold">Finance Hub</div>
          </div>
        </div>

        <div className="relative max-w-md">
          <h1 className="text-3xl font-bold leading-tight">
            Kelola invoice & pembayaran keanggotaan dalam satu tempat.
          </h1>
          <p className="mt-4 text-white/80">
            Sistem finance terpadu untuk BNI Grow Chapter Management — pendaftaran,
            renewal, dan rekonsiliasi pembayaran via Paper.id.
          </p>
        </div>

        <div className="relative flex items-center gap-2 text-sm text-white/70">
          <ShieldCheck className="h-4 w-4" />
          Akses khusus National Admin · Terenkripsi
        </div>
      </div>

      {/* Form panel */}
      <div className="flex w-full flex-col items-center justify-center px-6 lg:w-1/2">
        <div className="w-full max-w-sm">
          <div className="mb-8 lg:hidden">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-brand-500 to-brand-600 text-xl font-extrabold text-white">
              B
            </div>
          </div>

          <h2 className="text-2xl font-bold text-ink-900">Selamat datang kembali</h2>
          <p className="mt-1.5 text-sm text-ink-500">Masuk untuk melanjutkan ke Finance Hub.</p>

          <form onSubmit={handleSubmit} className="mt-8 space-y-4">
            <Field label="Email">
              <div className="relative">
                <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@bni-finance.com"
                  className="pl-10"
                  autoComplete="email"
                />
              </div>
            </Field>

            <Field label="Password">
              <div className="relative">
                <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="pl-10"
                  autoComplete="current-password"
                />
              </div>
            </Field>

            {error && (
              <div className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{error}</div>
            )}

            <Button type="submit" size="lg" loading={loading} className="w-full">
              Masuk
              {!loading && <ArrowRight className="h-4 w-4" />}
            </Button>
          </form>

          {useMock && (
            <div className="mt-6 rounded-xl border border-dashed border-ink-200 bg-white px-4 py-3 text-xs text-ink-500">
              <span className="font-semibold text-ink-700">Demo:</span> gunakan kredensial apa pun —
              data berjalan di atas mock repository.
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
