import { useEffect, useState } from 'react'
import { Button, Field, Input, Modal, useToast } from '@/components/ui'
import { chapterService } from '@/services'
import type { Chapter } from '@/types'

/**
 * Ubah nama tampil, area, dan kota chapter.
 *
 * Nama inti dan id tidak ikut: impor laporan BNI Connect dan sinkron BNI VM
 * mencocokkan chapter lewat keduanya.
 */
export function UbahChapterModal({
  chapter,
  onClose,
  onDisimpan,
}: {
  chapter: Chapter | null
  onClose: () => void
  onDisimpan: () => void
}) {
  const { toast } = useToast()
  const [tampil, setTampil] = useState('')
  const [area, setArea] = useState('')
  const [kota, setKota] = useState('')
  const [menyimpan, setMenyimpan] = useState(false)

  useEffect(() => {
    if (!chapter) return
    setTampil(chapter.displayName)
    setArea(chapter.areaName ?? '')
    setKota(chapter.cityName ?? '')
  }, [chapter])

  const simpan = async () => {
    if (!chapter) return
    if (!tampil.trim()) {
      toast('Nama tampil wajib diisi.', 'error')
      return
    }
    setMenyimpan(true)
    try {
      await chapterService.update(chapter.id, {
        displayName: tampil.trim(),
        areaName: area.trim(),
        cityName: kota.trim(),
      })
      toast(`${tampil.trim()} disimpan.`)
      onDisimpan()
      onClose()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan chapter.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  return (
    <Modal
      open={!!chapter}
      onClose={onClose}
      title={`Ubah ${chapter?.displayName ?? 'chapter'}`}
      footer={
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onClose} disabled={menyimpan}>
            Batal
          </Button>
          <Button onClick={simpan} loading={menyimpan}>
            Simpan
          </Button>
        </div>
      }
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Field label="Nama tampil" required>
            <Input value={tampil} onChange={(e) => setTampil(e.target.value)} />
          </Field>
        </div>
        <Field label="Area">
          <Input value={area} onChange={(e) => setArea(e.target.value)} placeholder="Jakarta Utara" />
        </Field>
        <Field label="Kota">
          <Input value={kota} onChange={(e) => setKota(e.target.value)} placeholder="Jakarta" />
        </Field>
      </div>
    </Modal>
  )
}
