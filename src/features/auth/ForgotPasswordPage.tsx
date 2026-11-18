import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, Mail, Send } from 'lucide-react'
import { Button, Field, Input } from '@/components/ui'
import { authService } from '@/services'
import { AuthLayout } from './AuthLayout'

/**
 * Meminta tautan reset kata sandi.
 *
 * Layarnya berubah jadi konfirmasi setelah dikirim, dan konfirmasinya TIDAK
 * mengatakan apakah emailnya terdaftar. Itu bukan kesopanan — halaman yang
 * membalas "email tidak ditemukan" adalah alat pemeriksa keanggotaan gratis:
 * siapa pun bisa mencoba daftar alamat dan tahu mana yang punya akun di sini.
 */
export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [mengirim, setMengirim] = useState(false)
  const [terkirim, setTerkirim] = useState(false)
  const [galat, setGalat] = useState('')

  const kirim = async (e: React.FormEvent) => {
    e.preventDefault()
    setGalat('')
    setMengirim(true)
    try {
      await authService.mintaResetSandi(email.trim())
      setTerkirim(true)
    } catch (err) {
      // Hanya galat NYATA yang tampil di sini: pengiriman email mati, jaringan
      // putus, terlalu sering mencoba. "Email tidak terdaftar" tidak pernah
      // sampai sejauh ini; server menjawabnya sebagai keberhasilan.
      setGalat(err instanceof Error ? err.message : 'Gagal mengirim permintaan.')
    } finally {
      setMengirim(false)
    }
  }

  return (
    <AuthLayout
      judul="Lupa Kata Sandi"
      keterangan="Masukkan email akun Anda. Kami kirimkan tautan untuk membuat kata sandi baru."
    >
      {terkirim ? (
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
            <Mail className="h-6 w-6" />
          </div>
          <p className="text-sm leading-relaxed text-ink-700">
            Kalau <span className="font-medium">{email.trim()}</span> terdaftar, tautan reset
            sudah dikirim ke sana.
          </p>
          <p className="mt-3 text-xs leading-relaxed text-ink-500">
            Tautannya berlaku 30 menit dan hanya bisa dipakai sekali. Belum masuk juga? Periksa
            folder spam.
          </p>
        </div>
      ) : (
        <form onSubmit={kirim} className="space-y-4">
          <Field label="Email">
            <Input
              type="email"
              required
              autoFocus
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="nama@bni-finance.com"
            />
          </Field>

          {galat && (
            <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{galat}</p>
          )}

          <Button type="submit" className="w-full" loading={mengirim} disabled={!email.trim()}>
            {!mengirim && <Send className="h-4 w-4" />}
            Kirim Tautan Reset
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
