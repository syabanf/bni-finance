import { useEffect, useState } from 'react'
import { BellRing, Save } from 'lucide-react'
import { Button, Card, CardBody, CardHeader, Field, Input, useToast } from '@/components/ui'
import { getAppSetting, setAppSetting } from '@/services/appSettings'

/**
 * Pengaturan pengingat dan denda.
 *
 * DUA SAKELAR YANG SENGAJA TERPISAH, dan bedanya nyata:
 *
 *   notifications_enabled    mematikan SELURUH notifikasi, termasuk pengiriman
 *                            manual — dipakai saat memindahkan lingkungan atau
 *                            menguji, ketika pesan yang telanjur keluar tidak
 *                            bisa ditarik kembali
 *   reminder_worker_enabled  hanya menghentikan yang otomatis; orang tetap bisa
 *                            mengirim sendiri
 *
 * Worker bawaannya MATI. Ia mengirim pesan sungguhan ke member dan membakar
 * nomor invoice Paper.id secara permanen, jadi menyalakannya harus keputusan
 * sadar — bukan efek samping sebuah deploy.
 */

const KUNCI = [
  'notifications_enabled',
  'reminder_worker_enabled',
  'reminder_offsets',
  'denda_aktif',
  'denda_per_hari',
  'denda_maks_hari',
] as const

type Kunci = (typeof KUNCI)[number]

export function ReminderCard() {
  const { toast } = useToast()
  const [nilai, setNilai] = useState<Record<Kunci, string>>({
    notifications_enabled: 'true',
    reminder_worker_enabled: 'false',
    reminder_offsets: '7,3,1',
    denda_aktif: 'false',
    denda_per_hari: '0',
    denda_maks_hari: '90',
  })
  const [memuat, setMemuat] = useState(true)
  const [menyimpan, setMenyimpan] = useState(false)

  useEffect(() => {
    let aktif = true
    Promise.all(KUNCI.map((k) => getAppSetting(k).catch(() => null)))
      .then((hasil) => {
        if (!aktif) return
        setNilai((lama) => {
          const baru = { ...lama }
          KUNCI.forEach((k, i) => {
            if (hasil[i] !== null) baru[k] = hasil[i] as string
          })
          return baru
        })
      })
      .finally(() => aktif && setMemuat(false))
    return () => {
      aktif = false
    }
  }, [])

  const ubah = (k: Kunci, v: string) => setNilai((s) => ({ ...s, [k]: v }))

  const simpan = async () => {
    setMenyimpan(true)
    try {
      await Promise.all(KUNCI.map((k) => setAppSetting(k, nilai[k])))
      toast('Pengaturan pengingat disimpan.')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan pengaturan.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  const workerNyala = nilai.reminder_worker_enabled === 'true'
  const notifNyala = nilai.notifications_enabled === 'true'
  const dendaNyala = nilai.denda_aktif === 'true'

  return (
    <Card className="lg:col-span-2">
      <CardHeader
        title={
          <span className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-amber-50 text-amber-500">
              <BellRing className="h-5 w-5" />
            </span>
            Pengingat & Denda
          </span>
        }
        subtitle="Kapan pengingat dikirim, dan bagaimana denda keterlambatan ditampilkan."
        action={
          <Button onClick={simpan} loading={menyimpan} disabled={memuat}>
            <Save className="h-4 w-4" />
            Simpan
          </Button>
        }
      />
      <CardBody className="space-y-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <Sakelar
            label="Notifikasi"
            nyala={notifNyala}
            onChange={(v) => ubah('notifications_enabled', String(v))}
            deskripsi="Sakelar induk. Dimatikan, TIDAK ADA pesan yang keluar — termasuk pengiriman manual."
          />
          <Sakelar
            label="Worker pengingat otomatis"
            nyala={workerNyala}
            onChange={(v) => ubah('reminder_worker_enabled', String(v))}
            deskripsi="Mengirim pengingat sendiri sesuai jadwal. Mematikannya tidak menghalangi pengiriman manual."
            peringatan={
              workerNyala
                ? 'Menyala: pengingat dikirim otomatis ke member, dan tiap pengiriman membakar nomor Paper.id secara permanen.'
                : undefined
            }
          />
        </div>

        <Field
          label="Jadwal pengingat"
          hint='Hari SEBELUM jatuh tempo, dipisah koma. "7,3,1" berarti tiga pengingat: H-7, H-3, dan H-1. Tiap pasangan invoice dan jadwal hanya dikirim SEKALI, bahkan setelah worker direstart.'
        >
          <Input
            value={nilai.reminder_offsets}
            onChange={(e) => ubah('reminder_offsets', e.target.value)}
            placeholder="7,3,1"
            disabled={memuat}
          />
        </Field>

        <div className="border-t border-ink-100 pt-5">
          <Sakelar
            label="Denda keterlambatan"
            nyala={dendaNyala}
            onChange={(v) => ubah('denda_aktif', String(v))}
            deskripsi="HANYA DITAMPILKAN — tidak pernah ditagihkan otomatis dan tidak mengubah nominal invoice."
          />

          {dendaNyala && (
            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <Field label="Denda per hari (Rp)">
                <Input
                  type="number"
                  min={0}
                  value={nilai.denda_per_hari}
                  onChange={(e) => ubah('denda_per_hari', e.target.value)}
                />
              </Field>
              <Field label="Maksimal hari dihitung" hint="0 berarti tanpa batas.">
                <Input
                  type="number"
                  min={0}
                  value={nilai.denda_maks_hari}
                  onChange={(e) => ubah('denda_maks_hari', e.target.value)}
                />
              </Field>
            </div>
          )}

          <p className="mt-3 text-xs leading-relaxed text-ink-500">
            Denda dihitung saat invoice dibaca, tidak pernah disimpan. Denda yang menempel dan
            tumbuh di invoice akan memaksa nominalnya berubah seiring waktu — dan invoice yang
            sudah terkirim ke Paper.id tidak bisa lagi disamakan dengan yang diterima member.
          </p>
        </div>
      </CardBody>
    </Card>
  )
}

function Sakelar({
  label,
  deskripsi,
  peringatan,
  nyala,
  onChange,
}: {
  label: string
  deskripsi: string
  peringatan?: string
  nyala: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="rounded-xl border border-ink-200 p-4">
      <label className="flex cursor-pointer items-start gap-3">
        <input
          type="checkbox"
          checked={nyala}
          onChange={(e) => onChange(e.target.checked)}
          className="mt-0.5 h-4 w-4 shrink-0 accent-bni-red"
        />
        <span className="min-w-0">
          <span className="block text-sm font-semibold text-ink-900">{label}</span>
          <span className="mt-0.5 block text-xs leading-relaxed text-ink-500">{deskripsi}</span>
          {peringatan && (
            <span className="mt-2 block rounded-lg bg-amber-50 p-2 text-xs leading-relaxed text-amber-800">
              {peringatan}
            </span>
          )}
        </span>
      </label>
    </div>
  )
}
