import { useEffect } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { CheckCircle2, CreditCard, Download, ExternalLink, FileText, XCircle } from 'lucide-react'
import type { InvoiceWithRelations } from '@/types'
import { BniLogo, Button, Card, CardBody, CardHeader, LoadingState } from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { invoiceService } from '@/services'
import { isSelfPaymentMode } from '@/services/supabase/paymentGateway'
import { formatCurrency, formatDate } from '@/lib/format'
import { InvoicePreview } from '@/features/invoices/components/InvoicePreview'
import { PaymentPanel } from '@/features/invoices/components/PaymentPanel'
import { downloadInvoice } from '@/features/invoices/lib/invoiceDocument'

export function PublicPaymentPage() {
  const { id = '' } = useParams()
  const [params] = useSearchParams()
  const forced = params.get('status') // 'success' | 'failed' (redirect dari gateway)

  const { data: invoice, loading, reload } = useAsync<InvoiceWithRelations | null>(
    () => invoiceService.getById(id),
    [id],
  )
  const { data: selfPayment } = useAsync<boolean>(() => isSelfPaymentMode(), [id])

  const isPaid = invoice?.status === 'paid'
  const isPayable = invoice?.status === 'sent' || invoice?.status === 'overdue'

  // Auto-poll status saat menunggu pembayaran — saat webhook menandai Lunas,
  // halaman member otomatis berpindah ke layar sukses tanpa refresh manual.
  useEffect(() => {
    if (!isPayable || isPaid) return
    const t = setInterval(() => reload(), 7000)
    return () => clearInterval(t)
  }, [isPayable, isPaid, reload])

  if (loading) return <Shell><Centered><LoadingState label="Memuat invoice…" /></Centered></Shell>

  if (!invoice) {
    return (
      <Shell>
        <Centered>
          <Result
            tone="neutral"
            icon={<FileText className="h-12 w-12" />}
            title="Invoice tidak ditemukan"
            desc="Tautan mungkin salah atau sudah tidak berlaku."
          />
        </Centered>
      </Shell>
    )
  }

  // Layar sukses
  if (isPaid || forced === 'success') {
    return (
      <Shell>
        <Centered>
          <Result
            tone="success"
            icon={<CheckCircle2 className="h-12 w-12" />}
            title="Pembayaran Berhasil"
            desc={`Terima kasih, tagihan ${invoice.number} sebesar ${formatCurrency(invoice.amount)} telah kami terima.`}
          >
            <div className="mt-4 rounded-xl bg-emerald-50 px-4 py-3 text-sm text-emerald-700">
              Keanggotaan diperpanjang hingga <strong>{formatDate(invoice.periodEnd)}</strong>.
            </div>
          </Result>
        </Centered>
      </Shell>
    )
  }

  // Layar gagal / dibatalkan
  if (forced === 'failed' || invoice.status === 'cancelled') {
    return (
      <Shell>
        <Centered>
          <Result
            tone="error"
            icon={<XCircle className="h-12 w-12" />}
            title={invoice.status === 'cancelled' ? 'Invoice Dibatalkan' : 'Pembayaran Gagal'}
            desc="Pembayaran tidak dapat diproses. Silakan coba lagi atau hubungi admin BNI."
          >
            {invoice.status !== 'cancelled' && (
              <Button className="mt-4" onClick={() => { window.location.href = `/pay/${invoice.id}` }}>
                Coba Lagi
              </Button>
            )}
          </Result>
        </Centered>
      </Shell>
    )
  }

  return (
    <Shell>
      <main className="mx-auto max-w-3xl space-y-5 px-4 py-6">
        {invoice.status === 'draft' && (
          <div className="rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4 text-sm text-amber-700">
            Invoice ini belum diterbitkan. Silakan hubungi admin BNI.
          </div>
        )}

        {/* 1. Informasi Invoice */}
        <div className="overflow-hidden rounded-2xl">
          <InvoicePreview invoice={invoice} />
        </div>

        {/* 2. Download PDF */}
        <div className="flex justify-center">
          <Button variant="outline" onClick={() => downloadInvoice(invoice)}>
            <Download className="h-4 w-4" />
            Download Invoice (PDF)
          </Button>
        </div>

        {/* 3. Metode Pembayaran */}
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
      </main>
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-ink-50">
      <header className="bg-gradient-to-r from-brand-600 to-brand-700 px-4 py-5 text-white">
        <div className="mx-auto flex max-w-3xl items-center gap-3">
          <span className="flex items-center rounded-lg bg-white px-3 py-1.5">
            <BniLogo className="h-6 w-auto" />
          </span>
          <div className="leading-tight">
            <div className="text-sm font-bold">BNI Indonesia</div>
            <div className="text-[11px] opacity-80">Payment Platform</div>
          </div>
        </div>
      </header>
      {children}
      <footer className="px-4 pb-10 pt-6 text-center text-xs text-ink-400">
        BNI Indonesia — Payment Platform
      </footer>
    </div>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-[60vh] items-center justify-center p-4">
      <div className="w-full max-w-md">{children}</div>
    </div>
  )
}

function Result({
  tone,
  icon,
  title,
  desc,
  children,
}: {
  tone: 'success' | 'error' | 'neutral'
  icon: React.ReactNode
  title: string
  desc: string
  children?: React.ReactNode
}) {
  const color =
    tone === 'success' ? 'text-emerald-500' : tone === 'error' ? 'text-red-500' : 'text-ink-300'
  return (
    <Card>
      <CardBody className="flex flex-col items-center py-10 text-center">
        <span className={color}>{icon}</span>
        <h1 className="mt-4 text-xl font-bold text-ink-900">{title}</h1>
        <p className="mt-1.5 max-w-xs text-sm text-ink-500">{desc}</p>
        {children}
      </CardBody>
    </Card>
  )
}
