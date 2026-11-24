import { useState } from 'react'
import { Globe2, Undo2 } from 'lucide-react'
import type { Invoice } from '@/types'
import { Button, Card, CardBody, CardHeader, Field, Input, useToast } from '@/components/ui'
import { useCan } from '@/features/auth/usePermission'
import { invoiceService } from '@/services'
import { formatDate } from '@/lib/format'
import { todayISO } from '@/lib/date'

/**
 * Langkah 6 alur renewal klien di halaman detail invoice: setelah invoice
 * renewal lunas, finance mengajukan renewal member ke BNI Global lalu
 * menandainya di sini.
 */
export function PengajuanCard({ invoice, onBerubah }: { invoice: Invoice; onBerubah: () => void }) {
  const { toast } = useToast()
  const bolehTandai = useCan('invoice:manage')
  const [tanggal, setTanggal] = useState(todayISO())
  const [sibuk, setSibuk] = useState(false)

  if (invoice.type !== 'renewal' || invoice.status !== 'paid') return null

  const jalankan = async (fn: () => Promise<unknown>, pesan: string) => {
    setSibuk(true)
    try {
      await fn()
      toast(pesan)
      onBerubah()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan pengajuan.', 'error')
    } finally {
      setSibuk(false)
    }
  }

  return (
    <Card>
      <CardHeader title="Pengajuan ke BNI Global" subtitle="Finance mengajukan renewal setelah invoice lunas." />
      <CardBody className="space-y-3">
        {invoice.renewalSubmittedAt ? (
          <div className="flex items-center gap-3 rounded-xl border border-emerald-100 bg-emerald-50/50 px-4 py-3">
            <Globe2 className="h-5 w-5 shrink-0 text-emerald-600" />
            <div className="min-w-0 text-sm">
              <div className="font-semibold text-emerald-700">Sudah diajukan</div>
              <div className="text-xs text-ink-500">
                {formatDate(invoice.renewalSubmittedAt)}
                {invoice.renewalSubmittedBy ? ` · ${invoice.renewalSubmittedBy}` : ''}
              </div>
            </div>
          </div>
        ) : (
          <p className="text-sm text-ink-500">Renewal member ini belum diajukan ke BNI Global.</p>
        )}

        {bolehTandai &&
          (invoice.renewalSubmittedAt ? (
            <Button
              variant="outline"
              size="sm"
              loading={sibuk}
              onClick={() => {
                if (window.confirm('Batalkan tanda pengajuan ke BNI Global?')) {
                  jalankan(() => invoiceService.unmarkRenewalSubmitted(invoice.id), 'Tanda pengajuan dibatalkan.')
                }
              }}
            >
              <Undo2 className="h-4 w-4" />
              Batalkan tanda
            </Button>
          ) : (
            <div className="flex flex-wrap items-end gap-2">
              <Field label="Tanggal diajukan" className="min-w-[10rem] flex-1">
                <Input type="date" value={tanggal} max={todayISO()} onChange={(e) => setTanggal(e.target.value)} />
              </Field>
              <Button
                loading={sibuk}
                onClick={() =>
                  jalankan(
                    () => invoiceService.markRenewalSubmitted([invoice.id], tanggal),
                    'Renewal ditandai sudah diajukan ke BNI Global.',
                  )
                }
              >
                <Globe2 className="h-4 w-4" />
                Tandai sudah diajukan
              </Button>
            </div>
          ))}
      </CardBody>
    </Card>
  )
}
