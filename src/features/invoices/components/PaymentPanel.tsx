import { useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import { Check, Copy, Landmark, QrCode, RefreshCw } from 'lucide-react'
import type { InvoiceWithRelations } from '@/types'
import { Badge, Button, CardBody, CardHeader, Card, useToast, WhatsAppIcon } from '@/components/ui'
import { formatCurrency, formatDateTime } from '@/lib/format'
import { createXenditPayment, VA_BANKS, type VaBank } from '@/services/supabase/paymentGateway'

interface Props {
  invoice: InvoiceWithRelations
  onUpdated: () => void
  /** Halaman member publik: sembunyikan aksi khusus admin (kirim WA). */
  publicMode?: boolean
}

const BANK_LABEL: Record<string, string> = {
  BCA: 'BCA', BNI: 'BNI', MANDIRI: 'Mandiri', BRI: 'BRI',
}

export function PaymentPanel({ invoice, onUpdated, publicMode = false }: Props) {
  const { toast } = useToast()
  const [busy, setBusy] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const hasPayment =
    invoice.xenditPaymentMethod &&
    (invoice.xenditVaNumber || invoice.xenditQrisString) &&
    invoice.xenditPaymentStatus !== 'EXPIRED'

  // QRIS Xendit dibatasi maks Rp 10.000.000 per transaksi
  const QRIS_MAX = 10_000_000
  const qrisDisabled = invoice.amount > QRIS_MAX

  const create = async (method: 'va' | 'qris', bank?: VaBank) => {
    setBusy(bank ?? method)
    try {
      await createXenditPayment(invoice.id, method, bank)
      toast('Detail pembayaran dibuat.')
      onUpdated()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal membuat pembayaran.', 'error')
    } finally {
      setBusy(null)
    }
  }

  const copyVa = () => {
    if (!invoice.xenditVaNumber) return
    navigator.clipboard.writeText(invoice.xenditVaNumber)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const sendWa = () => {
    const name = invoice.member?.name ?? 'Bapak/Ibu'
    const lines = [
      `Halo ${name},`,
      ``,
      `Berikut detail pembayaran tagihan *${invoice.number}*:`,
      `💰 Nominal: *${formatCurrency(invoice.amount)}*`,
    ]
    if (invoice.xenditPaymentMethod === 'va') {
      lines.push(
        `🏦 Bank: *${BANK_LABEL[invoice.xenditVaBank ?? ''] ?? invoice.xenditVaBank}*`,
        `🔢 No. Virtual Account: *${invoice.xenditVaNumber}*`,
      )
    } else {
      lines.push(`📱 Silakan scan QRIS yang kami kirimkan untuk membayar.`)
    }
    if (invoice.xenditExpiresAt) lines.push(`⏰ Berlaku s/d ${formatDateTime(invoice.xenditExpiresAt)}`)
    lines.push(``, `Terima kasih. 🙏`)
    window.open(`https://wa.me/?text=${encodeURIComponent(lines.join('\n'))}`, '_blank')
  }

  return (
    <Card>
      <CardHeader
        title="Pembayaran Mandiri (Xendit)"
        subtitle="Buat Virtual Account atau QRIS, member dapat membayar sendiri."
      />
      <CardBody className="space-y-4">
        {hasPayment ? (
          <>
            <div className="flex items-center justify-between">
              <Badge tone={invoice.xenditPaymentStatus === 'PAID' ? 'green' : 'amber'}>
                {invoice.xenditPaymentStatus === 'PAID' ? 'Sudah dibayar' : 'Menunggu pembayaran'}
              </Badge>
              <button
                onClick={onUpdated}
                className="inline-flex items-center gap-1 text-xs font-medium text-ink-500 hover:text-ink-800"
              >
                <RefreshCw className="h-3.5 w-3.5" /> Cek status
              </button>
            </div>

            {invoice.xenditPaymentMethod === 'va' ? (
              <div className="rounded-xl border border-ink-200 p-4">
                <div className="text-xs text-ink-400">Virtual Account {BANK_LABEL[invoice.xenditVaBank ?? ''] ?? invoice.xenditVaBank}</div>
                <div className="mt-1 flex items-center justify-between gap-3">
                  <span className="font-mono text-lg font-bold tracking-wide text-ink-900">{invoice.xenditVaNumber}</span>
                  <Button variant="outline" size="sm" onClick={copyVa}>
                    {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                    {copied ? 'Tersalin' : 'Salin'}
                  </Button>
                </div>
                <div className="mt-2 text-sm text-ink-500">
                  Nominal: <span className="font-semibold text-ink-800">{formatCurrency(invoice.amount)}</span>
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center rounded-xl border border-ink-200 p-4">
                <div className="rounded-lg bg-white p-3">
                  <QRCodeSVG value={invoice.xenditQrisString ?? ''} size={180} />
                </div>
                <div className="mt-3 text-sm text-ink-500">
                  Scan untuk membayar <span className="font-semibold text-ink-800">{formatCurrency(invoice.amount)}</span>
                </div>
              </div>
            )}

            {invoice.xenditExpiresAt && (
              <div className="text-xs text-ink-400">Berlaku sampai {formatDateTime(invoice.xenditExpiresAt)}</div>
            )}

            <div className="flex flex-wrap gap-2 border-t border-ink-100 pt-3">
              {!publicMode && (
                <Button variant="outline" size="sm" onClick={sendWa}>
                  <WhatsAppIcon className="h-4 w-4" /> Kirim via WhatsApp
                </Button>
              )}
              {!(invoice.xenditPaymentMethod === 'va' && qrisDisabled) && (
                <Button variant="ghost" size="sm" onClick={() => create(invoice.xenditPaymentMethod === 'va' ? 'qris' : 'va', invoice.xenditPaymentMethod === 'va' ? undefined : 'BCA')}>
                  Ganti metode
                </Button>
              )}
            </div>
          </>
        ) : (
          <>
            <div>
              <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-ink-400">
                <Landmark className="h-4 w-4" /> Virtual Account
              </div>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                {VA_BANKS.map((bank) => (
                  <Button
                    key={bank}
                    variant="outline"
                    size="sm"
                    loading={busy === bank}
                    onClick={() => create('va', bank)}
                  >
                    {BANK_LABEL[bank]}
                  </Button>
                ))}
              </div>
            </div>
            <div className="border-t border-ink-100 pt-3">
              <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-ink-400">
                <QrCode className="h-4 w-4" /> QRIS
              </div>
              <Button
                variant="outline"
                size="sm"
                loading={busy === 'qris'}
                disabled={qrisDisabled}
                onClick={() => create('qris')}
              >
                <QrCode className="h-4 w-4" /> Buat QRIS
              </Button>
              {qrisDisabled && (
                <p className="mt-2 text-xs text-ink-400">
                  Nominal melebihi batas QRIS (maks Rp 10.000.000). Gunakan Virtual Account.
                </p>
              )}
            </div>
          </>
        )}
      </CardBody>
    </Card>
  )
}
