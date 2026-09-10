import { useEffect, useState } from 'react'
import { Eye, EyeOff, Key, Save } from 'lucide-react'
import { Button, Card, CardBody, CardHeader, Field, Input, useToast } from '@/components/ui'
import { getAppSetting, setAppSetting } from '@/services/appSettings'

/**
 * Alamat dan token BNI Visitor Management.
 *
 * Sebelumnya menumpang di halaman "Sinkronisasi Data". Saat tombol tariknya
 * dipindah ke halaman Member dan Chapter, konfigurasi ini TIDAK ikut pindah ke
 * sana — ia bukan aksi harian, melainkan pengaturan yang disentuh sekali lalu
 * dilupakan. Menaruhnya di samping tombol berarti setiap orang yang sekadar
 * ingin menyegarkan daftar juga melihat kolom token.
 */
export function BniVmCard() {
  const { toast } = useToast()
  const [url, setUrl] = useState('')
  const [token, setToken] = useState('')
  const [terlihat, setTerlihat] = useState(false)
  const [menyimpan, setMenyimpan] = useState(false)
  const [memuat, setMemuat] = useState(true)

  useEffect(() => {
    let aktif = true
    Promise.all([
      getAppSetting('bni_vm_url').catch(() => null),
      getAppSetting('bni_vm_token').catch(() => null),
    ])
      .then(([u, t]) => {
        if (!aktif) return
        setUrl(u ?? '')
        setToken(t ?? '')
      })
      .finally(() => aktif && setMemuat(false))
    return () => {
      aktif = false
    }
  }, [])

  const simpan = async () => {
    setMenyimpan(true)
    try {
      await setAppSetting('bni_vm_url', url.trim())
      await setAppSetting('bni_vm_token', token.trim())
      toast('Konfigurasi BNI VM disimpan.')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan konfigurasi.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  return (
    <Card>
      <CardHeader
        title={
          <span className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-amber-50 text-amber-500">
              <Key className="h-5 w-5" />
            </span>
            BNI Visitor Management
          </span>
        }
        subtitle="Sumber data member dan chapter. Tombol tariknya ada di halaman Member dan Chapter."
      />
      <CardBody className="space-y-4">
        <Field label="Alamat API">
          <Input
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://www.bni-vh.com/api/external/v1"
            className="font-mono text-sm"
            disabled={memuat}
          />
        </Field>

        <Field
          label="Token"
          hint="Disimpan di server dan tidak pernah dikirim ke browser."
        >
          <div className="relative">
            <Input
              type={terlihat ? 'text' : 'password'}
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="bnifin_…"
              className="pr-10 font-mono text-sm"
              disabled={memuat}
            />
            <button
              type="button"
              onClick={() => setTerlihat((v) => !v)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-ink-400 hover:text-ink-600"
              aria-label={terlihat ? 'Sembunyikan token' : 'Tampilkan token'}
            >
              {terlihat ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
        </Field>

        <Button onClick={simpan} loading={menyimpan} disabled={memuat}>
          {!menyimpan && <Save className="h-4 w-4" />}
          Simpan
        </Button>
      </CardBody>
    </Card>
  )
}
