import { useEffect, useState } from 'react'
import { Package, Save } from 'lucide-react'
import { Button, Card, CardBody, CardHeader, Field, Input, Textarea, useToast } from '@/components/ui'
import { getAppSetting, setAppSetting } from '@/services/appSettings'
import {
  KUNCI_TEKS,
  TEKS_BAWAAN,
  lupakanTeksInvoice,
  type TeksInvoice,
} from '@/features/invoices/lib/teksInvoice'

/**
 * Teks invoice yang dikirim ke Paper.id: nama dan deskripsi produk, catatan,
 * serta syarat dan ketentuan.
 *
 * TEKS INI YANG DIBACA MEMBER, bukan istilah internal kita. Ia muncul di
 * invoice yang mereka terima lewat email, dan mengubahnya seharusnya tidak
 * menuntut deploy.
 *
 * Nama dan deskripsi yang dikosongkan memakai teks bawaan, bukan mengirim
 * baris kosong: baris kosong di invoice orang lebih buruk daripada teks bawaan
 * yang masih masuk akal. Catatan dan syarat yang dikosongkan tidak dikirim
 * sama sekali, dan bagiannya hilang dari invoice.
 */

const NAMA = Object.keys(KUNCI_TEKS) as (keyof TeksInvoice)[]

const KOSONG = Object.fromEntries(NAMA.map((n) => [n, ''])) as TeksInvoice

const PRODUK = [
  { judul: 'Pendaftaran', nama: 'produkPendaftaran', deskripsi: 'deskripsiPendaftaran' },
  { judul: 'Renewal', nama: 'produkRenewal', deskripsi: 'deskripsiRenewal' },
] as const

export function PaperProdukCard() {
  const { toast } = useToast()
  const [nilai, setNilai] = useState<TeksInvoice>(KOSONG)
  const [memuat, setMemuat] = useState(true)
  const [menyimpan, setMenyimpan] = useState(false)

  useEffect(() => {
    let aktif = true
    // Dibaca langsung, bukan lewat muatTeksInvoice: formulir harus menampilkan
    // kolom kosong apa adanya, sedangkan fungsi itu mengisinya dengan bawaan.
    Promise.all(NAMA.map((n) => getAppSetting(KUNCI_TEKS[n]).catch(() => null)))
      .then((hasil) => {
        if (!aktif) return
        setNilai((lama) => {
          const baru = { ...lama }
          NAMA.forEach((n, i) => {
            if (hasil[i]) baru[n] = hasil[i] as string
          })
          return baru
        })
      })
      .finally(() => {
        if (aktif) setMemuat(false)
      })
    return () => {
      aktif = false
    }
  }, [])

  const ubah = (n: keyof TeksInvoice, v: string) => setNilai((s) => ({ ...s, [n]: v }))

  const simpan = async () => {
    setMenyimpan(true)
    try {
      await Promise.all(NAMA.map((n) => setAppSetting(KUNCI_TEKS[n], nilai[n].trim())))
      // Pratinjau dan unduhan invoice membaca teks yang sama.
      lupakanTeksInvoice()
      toast('Teks invoice disimpan. Berlaku untuk invoice yang dikirim setelah ini.')
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
            Teks Invoice di Paper.id
          </span>
        }
        subtitle="Produk, deskripsi, catatan, dan syarat yang tercetak di invoice yang diterima member."
        action={
          <Button onClick={simpan} loading={menyimpan} disabled={memuat}>
            <Save className="h-4 w-4" />
            Simpan
          </Button>
        }
      />
      <CardBody className="space-y-5">
        {PRODUK.map(({ judul, nama, deskripsi }) => (
          <div key={judul} className="space-y-3 border-t border-ink-100 pt-4 first:border-0 first:pt-0">
            <h3 className="text-sm font-semibold text-ink-900">{judul}</h3>
            <Field label="Nama produk" hint={`Kosong = "${TEKS_BAWAAN[nama]}"`}>
              <Input
                value={nilai[nama]}
                onChange={(e) => ubah(nama, e.target.value)}
                placeholder={TEKS_BAWAAN[nama]}
                disabled={memuat}
              />
            </Field>
            <Field label="Deskripsi" hint={`Kosong = "${TEKS_BAWAAN[deskripsi]}"`}>
              <Textarea
                value={nilai[deskripsi]}
                onChange={(e) => ubah(deskripsi, e.target.value)}
                placeholder={TEKS_BAWAAN[deskripsi]}
                disabled={memuat}
              />
            </Field>
          </div>
        ))}

        <div className="space-y-3 border-t border-ink-100 pt-4">
          <h3 className="text-sm font-semibold text-ink-900">Catatan dan syarat</h3>
          <Field
            label="Catatan invoice"
            hint="Untuk detail rekening bila member ingin transfer langsung. Kosong = bagian ini tidak tampil."
          >
            <Textarea
              value={nilai.catatan}
              onChange={(e) => ubah('catatan', e.target.value)}
              placeholder={'Transfer ke Bank … nomor rekening … atas nama …'}
              disabled={memuat}
            />
          </Field>
          <Field label="Syarat dan ketentuan" hint="Kosong = bagian ini tidak tampil di invoice Paper.id.">
            <Textarea
              value={nilai.syarat}
              onChange={(e) => ubah('syarat', e.target.value)}
              placeholder="Pembayaran paling lambat pada tanggal jatuh tempo. …"
              disabled={memuat}
            />
          </Field>
          <p className="rounded-lg bg-amber-50 px-3 py-2 text-xs leading-relaxed text-amber-800">
            Pembayaran lewat transfer langsung tidak lewat Paper.id, jadi invoicenya tidak lunas
            otomatis. Catat pembayarannya dari halaman detail invoice.
          </p>
        </div>

        <p className="text-xs leading-relaxed text-ink-500">
          Perubahan hanya berlaku untuk invoice yang dikirim setelah disimpan. Invoice yang sudah
          terlanjur dikirim ke Paper.id tidak bisa diubah teksnya — nomornya sudah terbit di sana.
        </p>
      </CardBody>
    </Card>
  )
}
