import { useState } from 'react'
import { FileUp, Upload } from 'lucide-react'
import type { Invoice } from '@/types'
import { Button, Card, CardBody, CardHeader, Field, Input, Modal, MoneyInput, Select, Textarea, useToast } from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { useCan } from '@/features/auth/usePermission'
import { proofService } from '@/services'
import { formatCurrency, formatDate, formatDateTime } from '@/lib/format'
import { MANUAL_METHODS, paymentMethodLabel } from '@/lib/paymentMethod'
import { todayISO } from '@/lib/date'
import { AksiBukti, BadgeBukti } from './AksiBukti'

/**
 * Bukti bayar di halaman detail invoice: langkah 5 alur renewal klien.
 *
 * ST mengirim bukti transfer member ke finance, dan finance menyetujuinya di
 * sini atau di halaman Bukti Bayar. Invoice baru lunas setelah disetujui.
 */
export function BuktiBayarCard({ invoice, onBerubah }: { invoice: Invoice; onBerubah: () => void }) {
  const { toast } = useToast()
  const bolehKirim = useCan('payment:submit')
  const daftar = useAsync(() => proofService.list({ invoiceId: invoice.id }), [invoice.id, invoice.status])
  const [buka, setBuka] = useState(false)
  const [nominal, setNominal] = useState(invoice.amount)
  const [tanggal, setTanggal] = useState(todayISO())
  const [metode, setMetode] = useState('bank_transfer')
  const [catatan, setCatatan] = useState('')
  const [berkas, setBerkas] = useState<File | null>(null)
  const [mengirim, setMengirim] = useState(false)

  const bukti = daftar.data ?? []
  const adaMenunggu = bukti.some((b) => b.status === 'pending')
  const bisaDibayar = invoice.status === 'sent' || invoice.status === 'overdue'
  const tampilTombol = bolehKirim && bisaDibayar && !adaMenunggu

  if (bukti.length === 0 && !tampilTombol) return null

  const berubah = () => {
    daftar.reload()
    onBerubah()
  }

  const kirim = async () => {
    if (!berkas) return
    setMengirim(true)
    try {
      await proofService.submit({
        invoiceId: invoice.id,
        amount: nominal,
        paidAt: tanggal,
        paymentMethod: metode,
        note: catatan,
        file: berkas,
      })
      toast('Bukti bayar terkirim ke finance.')
      setBuka(false)
      setBerkas(null)
      setCatatan('')
      berubah()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal mengirim bukti bayar.', 'error')
    } finally {
      setMengirim(false)
    }
  }

  return (
    <Card>
      <CardHeader title="Bukti Bayar" subtitle="ST mengirim bukti, finance memverifikasi sebelum invoice lunas." />
      <CardBody className="space-y-3">
        {bukti.map((b) => (
          <div key={b.id} className="space-y-2 rounded-xl border border-ink-100 px-4 py-3">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div className="min-w-0">
                <div className="text-sm font-semibold text-ink-900">{formatCurrency(b.amount)}</div>
                <div className="text-xs text-ink-500">
                  {paymentMethodLabel(b.paymentMethod)} · dibayar {formatDate(b.paidAt)}
                </div>
                <div className="text-xs text-ink-400">
                  Dikirim {b.submittedBy ?? 'ST'} · {formatDateTime(b.submittedAt)}
                </div>
              </div>
              <BadgeBukti status={b.status} />
            </div>
            {b.note && <p className="text-xs text-ink-600">{b.note}</p>}
            {b.reviewNote && (
              <p className="rounded-lg bg-ink-50 px-3 py-2 text-xs text-ink-700">
                {b.reviewedBy ?? 'Finance'}: {b.reviewNote}
              </p>
            )}
            <div className="flex flex-wrap items-center justify-between gap-2">
              <a
                href={b.proofUrl}
                target="_blank"
                rel="noreferrer"
                className="text-xs font-medium text-brand-500 hover:underline"
              >
                Lihat bukti
              </a>
              <AksiBukti bukti={b} onBerubah={berubah} />
            </div>
          </div>
        ))}

        {tampilTombol && (
          <Button variant="outline" onClick={() => setBuka(true)}>
            <Upload className="h-4 w-4" />
            Kirim bukti bayar ke finance
          </Button>
        )}
      </CardBody>

      <Modal
        open={buka}
        onClose={() => setBuka(false)}
        title={`Kirim bukti bayar ${invoice.number}`}
        description="Finance memeriksa bukti ini lalu menandai invoice lunas."
        footer={
          <div className="flex w-full justify-end gap-2">
            <Button variant="outline" onClick={() => setBuka(false)}>
              Batal
            </Button>
            <Button onClick={kirim} loading={mengirim} disabled={!berkas || nominal <= 0}>
              Kirim ke finance
            </Button>
          </div>
        }
      >
        <div className="space-y-4">
          <Field label="Nominal dibayar">
            <MoneyInput value={nominal} onChange={setNominal} />
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Tanggal bayar">
              <Input type="date" value={tanggal} max={todayISO()} onChange={(e) => setTanggal(e.target.value)} />
            </Field>
            <Field label="Metode">
              <Select value={metode} onChange={(e) => setMetode(e.target.value)}>
                {MANUAL_METHODS.map((m) => (
                  <option key={m.value} value={m.value}>
                    {m.label}
                  </option>
                ))}
              </Select>
            </Field>
          </div>
          <Field label="Berkas bukti" hint="Foto atau PDF bukti transfer.">
            <label className="flex cursor-pointer items-center gap-3 rounded-xl border border-dashed border-ink-300 px-4 py-3 text-sm transition-colors hover:bg-ink-50">
              <FileUp className="h-4 w-4 shrink-0 text-brand-500" />
              <span className="min-w-0 flex-1 truncate text-ink-700">{berkas ? berkas.name : 'Pilih berkas…'}</span>
              <input
                type="file"
                accept="application/pdf,image/jpeg,image/png,image/webp,image/heic"
                className="hidden"
                onChange={(e) => setBerkas(e.target.files?.[0] ?? null)}
              />
            </label>
          </Field>
          <Field label="Catatan (opsional)">
            <Textarea value={catatan} onChange={(e) => setCatatan(e.target.value)} rows={2} />
          </Field>
        </div>
      </Modal>
    </Card>
  )
}
