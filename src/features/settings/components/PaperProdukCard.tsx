import { useEffect, useState } from 'react'
import { Package, Save } from 'lucide-react'
import { Button, Card, CardBody, CardHeader, Field, Input, Textarea, useToast } from '@/components/ui'
import { getAppSetting, setAppSetting } from '@/services/appSettings'

/**
 * Nama dan deskripsi produk yang dikirim ke Paper.id.
 *
 * TEKS INI YANG DIBACA MEMBER, bukan istilah internal kita. Ia muncul di baris
 * "Produk" dan "Deskripsi" pada invoice yang mereka terima lewat email, dan
 * mengubahnya seharusnya tidak menuntut deploy.
 *
 * Dikosongkan berarti memakai teks bawaan, bukan mengirim baris kosong: invoice
 * tanpa nama produk tetap diterima Paper.id apa adanya, dan baris kosong di
 * invoice orang lebih buruk daripada teks bawaan yang masih masuk akal.
 */

const KUNCI = [
  'paper_produk_pendaftaran',
  'paper_deskripsi_pendaftaran',
  'paper_produk_renewal',
  'paper_deskripsi_renewal',
] as const

type Kunci = (typeof KUNCI)[number]

const BAWAAN: Record<Kunci, string> = {
  paper_produk_pendaftaran: 'Biaya Pendaftaran Member BNI',
  paper_deskripsi_pendaftaran: 'Pendaftaran anggota baru (berlaku 1 tahun)',
  paper_produk_renewal: 'Perpanjangan Keanggotaan BNI',
  paper_deskripsi_renewal: 'Perpanjangan keanggotaan tahunan',
}

export function PaperProdukCard() {
  const { toast } = useToast()
  const [nilai, setNilai] = useState<Record<Kunci, string>>({
    paper_produk_pendaftaran: '',
    paper_deskripsi_pendaftaran: '',
    paper_produk_renewal: '',
    paper_deskripsi_renewal: '',
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
            if (hasil[i]) baru[k] = hasil[i] as string
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
      await Promise.all(KUNCI.map((k) => setAppSetting(k, nilai[k].trim())))
      toast('Produk Paper.id disimpan. Berlaku untuk invoice yang dikirim setelah ini.')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  return (
    <Card className="lg:col-span-2">
      <CardHeader
        title={
          <span className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-50 text-blue-500">
              <Package className="h-5 w-5" />
            </span>
            Produk di Paper.id
          </span>
        }
        subtitle="Teks yang muncul di baris Produk dan Deskripsi pada invoice yang diterima member."
        action={
          <Button onClick={simpan} loading={menyimpan} disabled={memuat}>
            <Save className="h-4 w-4" />
            Simpan
          </Button>
        }
      />
      <CardBody className="space-y-5">
        {(['pendaftaran', 'renewal'] as const).map((tipe) => {
          const kNama = `paper_produk_${tipe}` as Kunci
          const kDesc = `paper_deskripsi_${tipe}` as Kunci
          return (
            <div key={tipe} className="space-y-3 border-t border-ink-100 pt-4 first:border-0 first:pt-0">
              <h3 className="text-sm font-semibold text-ink-900">
                {tipe === 'pendaftaran' ? 'Pendaftaran' : 'Renewal'}
              </h3>
              <Field label="Nama produk" hint={`Kosong = "${BAWAAN[kNama]}"`}>
                <Input
                  value={nilai[kNama]}
                  onChange={(e) => ubah(kNama, e.target.value)}
                  placeholder={BAWAAN[kNama]}
                  disabled={memuat}
                />
              </Field>
              <Field label="Deskripsi" hint={`Kosong = "${BAWAAN[kDesc]}"`}>
                <Textarea
                  value={nilai[kDesc]}
                  onChange={(e) => ubah(kDesc, e.target.value)}
                  placeholder={BAWAAN[kDesc]}
                  disabled={memuat}
                />
              </Field>
            </div>
          )
        })}

        <p className="text-xs leading-relaxed text-ink-500">
          Perubahan hanya berlaku untuk invoice yang dikirim setelah disimpan. Invoice yang sudah
          terlanjur dikirim ke Paper.id tidak bisa diubah teksnya — nomornya sudah terbit di sana.
        </p>
      </CardBody>
    </Card>
  )
}
