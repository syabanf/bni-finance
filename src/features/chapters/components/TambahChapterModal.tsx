import { useEffect, useState } from 'react'
import { Plus } from 'lucide-react'
import { Button, Field, Input, Modal, useToast } from '@/components/ui'
import { chapterService } from '@/services'

/**
 * Tambah chapter yang belum ada, misalnya sebelum mengimpor laporan BNI Connect
 * miliknya.
 *
 * Laporan BNI Connect menyebut chapternya di baris parameter ("Chapter:
 * Supreme"), dan importer menolak laporan yang tidak cocok dengan chapter
 * tujuan. Nama yang diisi di sini sebaiknya sama dengan nama di BNI Connect;
 * nama tampil diturunkan dengan awalan "BNI", dan id dengan awalan "ch-".
 */
export function TambahChapterModal({
  open,
  onClose,
  onDibuat,
}: {
  open: boolean
  onClose: () => void
  onDibuat: () => void
}) {
  const { toast } = useToast()
  const [nama, setNama] = useState('')
  const [area, setArea] = useState('')
  const [kota, setKota] = useState('')
  const [menyimpan, setMenyimpan] = useState(false)

  useEffect(() => {
    if (!open) return
    setNama('')
    setArea('')
    setKota('')
  }, [open])

  const bersih = nama.trim()
  const id = `ch-${bersih.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')}`
  const tampil = /^bni\s/i.test(bersih) ? bersih : `BNI ${bersih}`

  const simpan = async () => {
    if (!bersih) {
      toast('Nama chapter wajib diisi.', 'error')
      return
    }
    setMenyimpan(true)
    try {
      await chapterService.create({
        id,
        name: bersih.replace(/^bni\s+/i, ''),
        displayName: tampil,
        areaName: area.trim() || undefined,
        cityName: kota.trim() || undefined,
      })
      toast(`${tampil} ditambahkan.`)
      onDibuat()
      onClose()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menambah chapter.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Tambah Chapter"
      description="Pakai nama yang sama dengan di BNI Connect, supaya laporannya bisa diimpor ke chapter ini."
      footer={
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onClose} disabled={menyimpan}>
            Batal
          </Button>
          <Button onClick={simpan} loading={menyimpan}>
            <Plus className="h-4 w-4" />
            Tambah
          </Button>
        </div>
      }
    >
      <div className="space-y-4">
        <Field
          label="Nama chapter"
          required
          hint={bersih ? `Tampil sebagai "${tampil}", id ${id}.` : 'Contoh: Supreme'}
        >
          <Input value={nama} onChange={(e) => setNama(e.target.value)} placeholder="Supreme" />
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Area">
            <Input value={area} onChange={(e) => setArea(e.target.value)} placeholder="Jakarta Utara" />
          </Field>
          <Field label="Kota">
            <Input value={kota} onChange={(e) => setKota(e.target.value)} placeholder="Jakarta" />
          </Field>
        </div>
      </div>
    </Modal>
  )
}
