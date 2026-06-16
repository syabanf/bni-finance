import { useParams } from 'react-router-dom'
import { CheckCircle2, CreditCard, ExternalLink, FileText } from 'lucide-react'
import type { InvoiceWithRelations } from '@/types'
import { Button, Card, CardBody, CardHeader, LoadingState } from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { invoiceService } from '@/services'
import { isSelfPaymentMode } from '@/services/supabase/paymentGateway'
import { formatCurrency } from '@/lib/format'
import { InvoicePreview } from '@/features/invoices/components/InvoicePreview'
import { PaymentPanel } from '@/features/invoices/components/PaymentPanel'

export function PublicPaymentPage() {
  const { id = '' } = useParams()
  const { data: invoice, loading, reload } = useAsync<InvoiceWithRelations | null>(
    () => invoiceService.getById(id),
    [id],
  )
  const { data: selfPayment } = useAsync<boolean>(() => isSelfPaymentMode(), [id])

  if (loading) return <Centered><LoadingState label="Memuat invoice…" /></Centered>

  if (!invoice) {
    return (
      <Centered>
        <Card>
          <CardBody className="py-12 text-center">
            <FileText className="mx-auto mb-3 h-10 w-10 text-ink-300" />
            <p className="font-semibold text-ink-800">Invoice tidak ditemukan</p>
            <p className="mt-1 text-sm text-ink-400">Tautan mungkin salah atau sudah tidak berlaku.</p>
          </CardBody>
        </Card>
      </Centered>
    )
  }

  const { status } = invoice
  const isPaid = status === 'paid'
  const isPayable = status === 'sent' || status === 'overdue'

  return (
    <div className="min-h-screen bg-ink-50">
      {/* Brand header */}
      <header className="bg-gradient-to-r from-brand-600 to-brand-700 px-4 py-5 text-white">
        <div className="mx-auto flex max-w-3xl items-center gap-2">
          <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-white/15 font-bold">B</span>
          <div className="leading-tight">
            <div className="text-sm font-bold">BNI Indonesia</div>
            <div className="text-[11px] opacity-80">Payment Platform</div>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-3xl space-y-5 px-4 py-6">
        {/* Status banners */}
        {isPaid && (
          <div className="flex items-center gap-3 rounded-2xl border border-emerald-200 bg-emerald-50 px-5 py-4 text-emerald-700">
            <CheckCircle2 className="h-6 w-6 flex-shrink-0" />
            <div>
              <div className="font-semibold">Pembayaran Lunas</div>
              <div className="text-sm">Terima kasih, tagihan {invoice.number} telah dibayar.</div>
            </div>
          </div>
        )}
        {status === 'draft' && (
          <div className="rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4 text-sm text-amber-700">
            Invoice ini belum diterbitkan. Silakan hubungi admin BNI.
          </div>
        )}
        {status === 'cancelled' && (
          <div className="rounded-2xl border border-ink-200 bg-ink-50 px-5 py-4 text-sm text-ink-500">
            Invoice ini telah dibatalkan.
          </div>
        )}

        {/* Payment options */}
        {isPayable && (
          selfPayment ? (
            <PaymentPanel invoice={invoice} onUpdated={reload} publicMode />
          ) : (
            <Card>
              <CardHeader title="Pembayaran" subtitle="Selesaikan pembayaran melalui Paper.id." />
              <CardBody className="space-y-3">
                <div className="text-sm text-ink-500">
                  Total tagihan: <span className="font-semibold text-ink-900">{formatCurrency(invoice.amount)}</span>
                </div>
                {invoice.paperIdPaymentUrl ? (
                  <Button onClick={() => window.open(invoice.paperIdPaymentUrl, '_blank')}>
                    <CreditCard className="h-4 w-4" />
                    Bayar via Paper.id
                    <ExternalLink className="h-4 w-4" />
                  </Button>
                ) : (
                  <p className="text-sm text-ink-400">Link pembayaran belum tersedia. Silakan hubungi admin.</p>
                )}
              </CardBody>
            </Card>
          )
        )}

        {/* Invoice document */}
        <div className="overflow-hidden rounded-2xl">
          <InvoicePreview invoice={invoice} />
        </div>
      </main>

      <footer className="px-4 pb-10 text-center text-xs text-ink-400">
        BNI Indonesia — Payment Platform
      </footer>
    </div>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-ink-50 p-4">
      <div className="w-full max-w-md">{children}</div>
    </div>
  )
}
