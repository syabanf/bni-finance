import { useEffect, useState } from 'react'
import { BellRing, Save } from 'lucide-react'
import { Button, Card, CardBody, CardHeader, Field, Input, Select, useToast } from '@/components/ui'
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
  'login_otp_enabled',
  'notifications_enabled',
  'reminder_worker_enabled',
  'reminder_offsets',
  'denda_aktif',
  'denda_jenis',
  'denda_satuan',
  'denda_per_hari',
  'denda_persen',
  'denda_maks_hari',
] as const

type Kunci = (typeof KUNCI)[number]

export function ReminderCard() {
  const { toast } = useToast()
  const [nilai, setNilai] = useState<Record<Kunci, string>>({
    login_otp_enabled: 'false',
    notifications_enabled: 'true',
    reminder_worker_enabled: 'false',
    reminder_offsets: '7,3,1',
    denda_aktif: 'false',
    denda_jenis: 'rupiah',
    denda_satuan: 'hari',
    denda_per_hari: '0',
    denda_persen: '0',
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

  /** Contoh perhitungan, memakai aturan yang sama dengan server. */
  const contohDenda = () => {
    const hariPerSatuan = nilai.denda_satuan === 'minggu' ? 7 : nilai.denda_satuan === 'bulan' ? 30 : 1
    const maks = Number(nilai.denda_maks_hari) || 0
    const hari = maks > 0 ? Math.min(30, maks) : 30
    const satuan = Math.floor(hari / hariPerSatuan)
    if (satuan <= 0) return 'Rp 0 (belum genap satu satuan)'
    const n =
      nilai.denda_jenis === 'persen'
        ? Math.floor((12_700_000 * (Number(nilai.denda_persen) || 0)) / 100) * satuan
        : (Number(nilai.denda_per_hari) || 0) * satuan
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n)
  }

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
            label="Kode masuk lewat email (OTP)"
            nyala={nilai.login_otp_enabled === 'true'}
            onChange={(v) => ubah('login_otp_enabled', String(v))}
            deskripsi="Setelah kata sandi benar, sistem mengirim kode 6 digit ke email dan meminta kode itu sebelum bisa masuk."
            peringatan={
              nilai.login_otp_enabled === 'true'
                ? 'Menyala: kalau pengiriman email bermasalah, tidak ada yang bisa masuk — termasuk Anda. Sistem otomatis melewati OTP bila email belum dikonfigurasi.'
                : undefined
            }
          />
          <Sakelar
            label="Notifikasi"
            nyala={notifNyala}
            onChange={(v) => ubah('notifications_enabled', String(v))}
            deskripsi="Kalau dimatikan, tidak ada pesan apa pun yang sampai ke member — termasuk yang Anda kirim sendiri."
          />
          <Sakelar
            label="Pengingat otomatis"
            nyala={workerNyala}
            onChange={(v) => ubah('reminder_worker_enabled', String(v))}
            deskripsi="Sistem mengirim pengingat sendiri sesuai jadwal. Kalau dimatikan, Anda tetap bisa mengirim satu per satu."
            peringatan={
              workerNyala
                ? 'Sedang menyala — pengingat dikirim otomatis ke member. Tiap pengiriman memakai satu nomor invoice Paper.id yang tidak bisa dipakai lagi.'
                : undefined
            }
          />
        </div>

        <Field
          label="Jadwal pengingat"
          hint='Berapa hari sebelum jatuh tempo, dipisah koma. Contoh: 7,3,1 mengirim tiga pengingat — tujuh hari, tiga hari, dan sehari sebelum jatuh tempo. Tiap member hanya menerima satu pengingat untuk tiap jadwal, walau sistem sempat dimatikan lalu dinyalakan lagi.'
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
            deskripsi="Hanya ditampilkan sebagai informasi. Denda tidak ditagihkan otomatis dan tidak mengubah nominal invoice."
          />

          {dendaNyala && (
            <div className="mt-4 space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="Jenis denda">
                  <Select
                    value={nilai.denda_jenis}
                    onChange={(e) => ubah('denda_jenis', e.target.value)}
                  >
                    <option value="rupiah">Nominal tetap (Rp)</option>
                    <option value="persen">Persentase dari tagihan</option>
                  </Select>
                </Field>
                <Field
                  label="Dihitung per"
                  hint="Denda bertambah setiap satu satuan keterlambatan yang GENAP."
                >
                  <Select
                    value={nilai.denda_satuan}
                    onChange={(e) => ubah('denda_satuan', e.target.value)}
                  >
                    <option value="hari">Hari</option>
                    <option value="minggu">Minggu (7 hari)</option>
                    <option value="bulan">Bulan (30 hari)</option>
                  </Select>
                </Field>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                {nilai.denda_jenis === 'persen' ? (
                  <Field
                    label={`Persentase per ${nilai.denda_satuan}`}
                    hint="Contoh: 2 berarti 2% dari nominal invoice."
                  >
                    <Input
                      type="number"
                      min={0}
                      step="0.1"
                      value={nilai.denda_persen}
                      onChange={(e) => ubah('denda_persen', e.target.value)}
                    />
                  </Field>
                ) : (
                  <Field label={`Denda per ${nilai.denda_satuan} (Rp)`}>
                    <Input
                      type="number"
                      min={0}
                      value={nilai.denda_per_hari}
                      onChange={(e) => ubah('denda_per_hari', e.target.value)}
                    />
                  </Field>
                )}
                <Field label="Maksimal hari dihitung" hint="0 berarti tanpa batas.">
                  <Input
                    type="number"
                    min={0}
                    value={nilai.denda_maks_hari}
                    onChange={(e) => ubah('denda_maks_hari', e.target.value)}
                  />
                </Field>
              </div>

              {/* Contoh dihitung dari angka yang sedang diisi, bukan angka
                  contoh yang tetap. Orang perlu melihat akibat dari yang baru
                  saja mereka ketik — sebelum menyimpannya, bukan sesudah
                  invoice pertama terlanjur menampilkannya. */}
              <p className="rounded-lg bg-ink-50 px-3 py-2.5 text-xs leading-relaxed text-ink-600">
                Contoh: tagihan Rp 12.700.000 yang telat 30 hari kena denda{' '}
                <span className="font-semibold text-ink-900">{contohDenda()}</span>.
              </p>
            </div>
          )}

          <p className="mt-3 text-xs leading-relaxed text-ink-500">
            Angka dendanya dihitung ulang tiap kali invoice dibuka, jadi selalu mengikuti
            keterlambatan hari ini. Nominal invoice yang sudah terkirim ke Paper.id tidak ikut
            berubah — tagihan yang diterima member harus tetap sama dengan yang tercatat di sini.
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
