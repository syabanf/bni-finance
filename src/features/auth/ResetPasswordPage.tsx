import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { ArrowLeft, CheckCircle2, Eye, EyeOff, KeyRound } from 'lucide-react'
import { Button, Field, Input } from '@/components/ui'
import { authService } from '@/services'
import { AuthLayout } from './AuthLayout'

const MIN_SANDI = 6

/**
 * Menukar tautan dari email dengan kata sandi baru.
 *
 * Token dibaca dari query string dan TIDAK pernah ditampilkan. Ia setara kata
 * sandi sementara; menampilkannya di layar berarti ia ikut terbaca oleh siapa
 * pun yang kebetulan melihat, ikut ter-screenshot, dan ikut tersalin saat orang
 * mengirim tangkapan layar untuk minta bantuan.
 */
export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const token = params.get('token') ?? ''

  const [sandi, setSandi] = useState('')
  const [ulang, setUlang] = useState('')
  const [terlihat, setTerlihat] = useState(false)
  const [menyimpan, setMenyimpan] = useState(false)
  const [galat, setGalat] = useState('')
  const [selesai, setSelesai] = useState(false)

  const terlaluPendek = sandi.length > 0 && sandi.length < MIN_SANDI
  const tidakSama = ulang.length > 0 && sandi !== ulang
  const bolehKirim = sandi.length >= MIN_SANDI && sandi === ulang && !menyimpan

  const simpan = async (e: React.FormEvent) => {
    e.preventDefault()
    setGalat('')
    setMenyimpan(true)
    try {
      await authService.resetSandi(token, sandi)
      setSelesai(true)
    } catch (err) {
      setGalat(err instanceof Error ? err.message : 'Gagal mengubah kata sandi.')
    } finally {
      setMenyimpan(false)
    }
  }

  return (
    <AuthLayout
      judul="Kata Sandi Baru"
      keterangan="Buat kata sandi baru untuk akun Anda."
    >
      {selesai ? (
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
            <CheckCircle2 className="h-6 w-6" />
          </div>
          <p className="text-sm text-ink-700">
            Kata sandi berhasil diubah. Silakan masuk dengan yang baru.
          </p>
          <Button className="mt-5 w-full" onClick={() => navigate('/login', { replace: true })}>
            Ke Halaman Masuk
          </Button>
        </div>
      ) : !token ? (
        // Tautan tanpa token biasanya berarti alamatnya terpotong saat
        // disalin dari email — disebutkan supaya orang tahu apa yang salah,
        // bukan hanya bahwa ada yang salah.
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-amber-50 text-amber-600">
            <KeyRound className="h-6 w-6" />
          </div>
          <p className="text-sm leading-relaxed text-ink-700">
            Tautannya tidak lengkap. Coba buka lagi langsung dari email — alamatnya kadang
            terpotong saat disalin.
          </p>
          <Link to="/forgot-password" className="mt-4 inline-block text-sm text-brand-600 hover:text-brand-700">
            Minta tautan baru
          </Link>
        </div>
      ) : (
        <form onSubmit={simpan} className="space-y-4">
          <Field
            label="Kata sandi baru"
            hint={`Minimal ${MIN_SANDI} karakter.`}
            error={terlaluPendek ? `Minimal ${MIN_SANDI} karakter.` : undefined}
          >
            <div className="relative">
              <Input
                type={terlihat ? 'text' : 'password'}
                required
                autoFocus
                value={sandi}
                onChange={(e) => setSandi(e.target.value)}
                className="pr-10"
              />
              <button
                type="button"
                onClick={() => setTerlihat((v) => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-ink-400 hover:text-ink-600"
                aria-label={terlihat ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
              >
                {terlihat ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </Field>

          <Field
            label="Ulangi kata sandi"
            error={tidakSama ? 'Belum sama dengan yang di atas.' : undefined}
          >
            <Input
              type={terlihat ? 'text' : 'password'}
              required
              value={ulang}
              onChange={(e) => setUlang(e.target.value)}
            />
          </Field>

          {galat && (
            <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{galat}</p>
          )}

          <Button type="submit" className="w-full" loading={menyimpan} disabled={!bolehKirim}>
            Simpan Kata Sandi
          </Button>
        </form>
      )}
    <Link
      to="/login"
      className="mt-5 flex items-center justify-center gap-1.5 text-sm text-ink-500 hover:text-ink-700"
    >
      <ArrowLeft className="h-4 w-4" />
      Kembali ke halaman masuk
    </Link>
    </AuthLayout>
  )
}
