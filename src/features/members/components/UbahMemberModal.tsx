import { useEffect, useState } from 'react'
import { Button, Field, Input, Modal, Select, useToast } from '@/components/ui'
import { memberService } from '@/services'
import type { UpdateMemberInput } from '@/services/types'
import type { Member, MemberStatus } from '@/types'

const LABEL_STATUS: Record<MemberStatus, string> = {
  active: 'Active',
  inactive: 'Inactive (member drop)',
  pending: 'Pending',
  visitor: 'Visitor',
  new_member: 'New Member',
}

/**
 * Ubah kontak dan status satu member.
 *
 * Nama tidak ikut: impor laporan BNI Connect mencocokkan member lewat nama, jadi
 * nama yang diubah di sini membuat impor berikutnya menambah member baru.
 *
 * Status hanya bisa berpindah ke Active atau Inactive. Status lain berubah
 * sendiri saat invoicenya lunas.
 */
export function UbahMemberModal({
  open,
  onClose,
  onDisimpan,
  member,
}: {
  open: boolean
  onClose: () => void
  onDisimpan: () => void
  member: Member
}) {
  const { toast } = useToast()
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [company, setCompany] = useState('')
  const [bidang, setBidang] = useState('')
  const [status, setStatus] = useState<MemberStatus>(member.status)
  const [jatuhTempo, setJatuhTempo] = useState('')
  const [bergabung, setBergabung] = useState('')
  const [menyimpan, setMenyimpan] = useState(false)

  useEffect(() => {
    if (!open) return
    setEmail(member.email ?? '')
    setPhone(member.phone ?? '')
    setCompany(member.company ?? '')
    setBidang(member.businessField ?? '')
    setStatus(member.status)
    setJatuhTempo(member.renewalDate?.slice(0, 10) ?? '')
    setBergabung(member.joinedDate?.slice(0, 10) ?? '')
  }, [open, member])

  const simpan = async () => {
    const ubah: UpdateMemberInput = {}
    if (email.trim() !== (member.email ?? '')) ubah.email = email.trim()
    if (phone.trim() !== (member.phone ?? '')) ubah.phone = phone.trim()
    if (company.trim() !== (member.company ?? '')) ubah.company = company.trim()
    if (bidang.trim() !== (member.businessField ?? '')) ubah.businessField = bidang.trim()
    if (status !== member.status) ubah.status = status
    // Tanggal tidak bisa dikosongkan lewat API, jadi hanya tanggal terisi yang dikirim.
    if (jatuhTempo && jatuhTempo !== member.renewalDate?.slice(0, 10)) ubah.renewalDate = jatuhTempo
    if (bergabung && bergabung !== member.joinedDate?.slice(0, 10)) ubah.joinedDate = bergabung
    if (Object.keys(ubah).length === 0) {
      onClose()
      return
    }
    setMenyimpan(true)
    try {
      await memberService.update(member.id, ubah)
      toast(`Data ${member.name} disimpan.`)
      onDisimpan()
      onClose()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan data member.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={`Ubah data ${member.name}`}
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
        <Field label="Jatuh tempo membership" hint="Renewal yang lunas memajukannya otomatis.">
          <Input type="date" value={jatuhTempo} onChange={(e) => setJatuhTempo(e.target.value)} />
        </Field>
        <Field label="Bergabung">
          <Input type="date" value={bergabung} onChange={(e) => setBergabung(e.target.value)} />
        </Field>
        <div className="sm:col-span-2">
          <Field
            label="Status keanggotaan"
            hint="Member drop diubah ke Inactive. Invoice yang sudah terbit tidak ikut berubah; batalkan dari halaman invoice bila perlu."
          >
            <Select value={status} onChange={(e) => setStatus(e.target.value as MemberStatus)}>
              {[...new Set<MemberStatus>([member.status, 'active', 'inactive'])].map((s) => (
                <option key={s} value={s}>
                  {LABEL_STATUS[s]}
                </option>
              ))}
            </Select>
          </Field>
        </div>
      </div>
    </Modal>
  )
}
