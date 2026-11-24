import { useState } from 'react'
import { CheckCircle2, Trash2, XCircle } from 'lucide-react'
import type { PaymentProof, ProofStatus } from '@/types'
import { Badge, Button, Field, Modal, Textarea, useToast } from '@/components/ui'
import { useCan } from '@/features/auth/usePermission'
import { proofService } from '@/services'

const STATUS: Record<ProofStatus, { label: string; tone: 'amber' | 'green' | 'red' }> = {
  pending: { label: 'Menunggu verifikasi', tone: 'amber' },
  approved: { label: 'Disetujui', tone: 'green' },
  rejected: { label: 'Ditolak', tone: 'red' },
}

export function BadgeBukti({ status }: { status: ProofStatus }) {
  return <Badge tone={STATUS[status].tone}>{STATUS[status].label}</Badge>
}

/**
 * Tombol untuk satu bukti yang masih menunggu. Finance (admin) menyetujui atau
 * menolak; ST menarik buktinya sendiri. Bukti yang sudah diperiksa tidak punya
 * tombol, karena server menolak semua perubahan atasnya.
 */
export function AksiBukti({ bukti, onBerubah }: { bukti: PaymentProof; onBerubah: () => void }) {
  const { toast } = useToast()
  const bolehPeriksa = useCan('payment:record')
  const bolehTarik = useCan('payment:submit')
  const [sibuk, setSibuk] = useState(false)
  const [tolak, setTolak] = useState(false)
  const [alasan, setAlasan] = useState('')

  if (bukti.status !== 'pending') return null

  const jalankan = async (fn: () => Promise<unknown>, pesan: string) => {
    setSibuk(true)
    try {
      await fn()
      toast(pesan)
      setTolak(false)
      onBerubah()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal memproses bukti bayar.', 'error')
    } finally {
      setSibuk(false)
    }
  }

  return (
    <div className="inline-flex flex-wrap justify-end gap-1.5">
      {bolehPeriksa && (
        <>
          <Button
            size="sm"
            loading={sibuk}
            onClick={() => jalankan(() => proofService.approve(bukti.id), `${bukti.invoiceNumber} lunas.`)}
          >
            <CheckCircle2 className="h-4 w-4" />
            Setujui
          </Button>
          <Button size="sm" variant="outline" disabled={sibuk} onClick={() => setTolak(true)}>
            <XCircle className="h-4 w-4" />
            Tolak
          </Button>
        </>
      )}
      {bolehTarik && (
        <Button
          size="sm"
          variant="outline"
          loading={sibuk}
          onClick={() => {
            if (window.confirm(`Tarik bukti bayar ${bukti.invoiceNumber}?`)) {
              jalankan(() => proofService.withdraw(bukti.id), 'Bukti bayar ditarik.')
            }
          }}
        >
          <Trash2 className="h-4 w-4" />
          Tarik
        </Button>
      )}

      <Modal
        open={tolak}
        onClose={() => setTolak(false)}
        title={`Tolak bukti bayar ${bukti.invoiceNumber}`}
        description="ST membaca alasan ini dan mengirim bukti baru."
        footer={
          <div className="flex w-full justify-end gap-2">
            <Button variant="outline" onClick={() => setTolak(false)}>
              Batal
            </Button>
            <Button
              variant="danger"
              loading={sibuk}
              disabled={!alasan.trim()}
              onClick={() => jalankan(() => proofService.reject(bukti.id, alasan), 'Bukti bayar ditolak.')}
            >
              Tolak bukti
            </Button>
          </div>
        }
      >
        <Field label="Alasan penolakan">
          <Textarea
            value={alasan}
            onChange={(e) => setAlasan(e.target.value)}
            rows={3}
            placeholder="Contoh: nominal di bukti tidak sama dengan tagihan"
          />
        </Field>
      </Modal>
    </div>
  )
}
