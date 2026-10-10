import { useEffect, useState } from 'react'
import { UserPlus } from 'lucide-react'
import type { Chapter } from '@/types'
import { Button, Field, Input, Modal, Select, useToast } from '@/components/ui'
import { memberService } from '@/services'

/**
 * Tambah satu member baru dengan status New Member.
 *
 * Statusnya tidak bisa dipilih: orang yang ditambahkan dari sini belum pernah
 * membayar biaya bergabung, jadi ia ditagih pendaftaran, lalu menjadi Active
 * saat renewal pertamanya lunas. Memberi pilihan status di formulir ini
 * membuka jalan menagih renewal orang yang belum pernah mendaftar.
 */
export function TambahMemberModal({
  open,
  onClose,
  onDibuat,
  chapters,
  chapterAwal,
}: {
  open: boolean
  onClose: () => void
  onDibuat: () => void
  chapters: Chapter[]
  chapterAwal?: string
}) {
  const { toast } = useToast()
  const [nama, setNama] = useState('')
  const [chapterId, setChapterId] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [company, setCompany] = useState('')
  const [bidang, setBidang] = useState('')
  const [menyimpan, setMenyimpan] = useState(false)

  // Formulir dikosongkan setiap kali dibuka, dengan chapter yang sedang
  // disaring di daftar sebagai bawaan.
  useEffect(() => {
    if (!open) return
    setNama('')
    setChapterId(chapterAwal ?? '')
    setEmail('')
    setPhone('')
    setCompany('')
    setBidang('')
  }, [open, chapterAwal])

  const simpan = async () => {
    if (!nama.trim() || !chapterId) {
      toast('Nama dan chapter wajib diisi.', 'error')
      return
    }
    setMenyimpan(true)
    try {
      await memberService.create({
        chapterId,
        name: nama.trim(),
        email: email.trim() || undefined,
        phone: phone.trim() || undefined,
        company: company.trim() || undefined,
        businessField: bidang.trim() || undefined,
        status: 'new_member',
      })
      toast(`${nama.trim()} ditambahkan sebagai New Member.`)
      onDibuat()
      onClose()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menambah member.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Tambah Member"
      description="Member baru masuk dengan status New Member dan ditagih pendaftaran."
      footer={
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onClose} disabled={menyimpan}>
            Batal
          </Button>
          <Button onClick={simpan} loading={menyimpan}>
            <UserPlus className="h-4 w-4" />
            Tambah
          </Button>
        </div>
      }
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Nama" required>
          <Input value={nama} onChange={(e) => setNama(e.target.value)} placeholder="Nama lengkap" />
        </Field>
        <Field label="Chapter" required>
          <Select value={chapterId} onChange={(e) => setChapterId(e.target.value)}>
            <option value="">Pilih chapter…</option>
            {chapters.map((c) => (
              <option key={c.id} value={c.id}>
                {c.displayName}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Email">
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </Field>
        <Field label="No. HP">
          <Input inputMode="tel" value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="08…" />
        </Field>
        <Field label="Perusahaan">
          <Input value={company} onChange={(e) => setCompany(e.target.value)} />
        </Field>
        <Field label="Bidang usaha">
          <Input value={bidang} onChange={(e) => setBidang(e.target.value)} />
        </Field>
      </div>
    </Modal>
  )
}
