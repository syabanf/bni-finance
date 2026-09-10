import { useParams, useNavigate, Link } from 'react-router-dom'
import { ArrowLeft, ArrowUpRight, FileText, Printer, Wallet } from 'lucide-react'
import type { PaymentWithInvoice } from '@/types'
import {
  Avatar,
  Button,
  Card,
  CardBody,
  CardHeader,
  EmptyState,
  ErrorState,
  InvoiceTypeBadge,
  LoadingState,
  PageHeader,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { paymentService } from '@/services'
import { formatCurrency, formatDateTime } from '@/lib/format'
import { paymentMethodLabel } from '@/lib/paymentMethod'
import { printReceipt } from '@/lib/receipt'

/**
 * Rincian satu pembayaran, beserta notanya.
 *
 * Sebelumnya baris pembayaran langsung melompat ke invoicenya. Itu menjawab
 * pertanyaan yang salah: yang dicari orang saat menekan sebuah pembayaran
 * adalah pembayaran ITU — kapan masuk, lewat apa, sebesar apa — bukan tagihan
 * yang melatarbelakanginya. Tagihannya tetap sejauh satu tombol dari sini.
 */
export function PaymentDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const { toast } = useToast()

  const { data, loading, error, reload } = useAsync<PaymentWithInvoice | null>(
    () => paymentService.getById(id),
    [id],
  )

  if (loading) return <LoadingState />
  if (error) return <ErrorState message={error} onRetry={reload} />
  if (!data) {
    return (
      <EmptyState
        icon={Wallet}
        title="Pembayaran tidak ditemukan"
        description="Pembayaran ini mungkin sudah dihapus."
        action={
          <Button variant="outline" onClick={() => navigate('/payments')}>
            <ArrowLeft className="h-4 w-4" />
            Kembali ke Pembayaran
          </Button>
        }
      />
    )
  }

  const cetak = () => {
    const ok = printReceipt({
      nomorNota: data.id,
      dibayarPada: data.paidAt,
      member: data.member?.name ?? '—',
      nomorInvoice: data.invoice?.number ?? '—',
      tipe:
        data.invoice?.type === 'renewal'
          ? 'Renewal'
          : data.invoice?.type === 'registration'
            ? 'Pendaftaran'
            : '—',
      metode: paymentMethodLabel(data.paymentMethod),
      nominal: data.amount,
      catatan: data.note ?? undefined,
    })
    if (!ok) toast('Izinkan popup di browser untuk mencetak nota.', 'error')
  }

  return (
    <div>
      <Link
        to="/payments"
        className="mb-3 inline-flex items-center gap-1.5 text-sm text-ink-500 hover:text-ink-700"
      >
        <ArrowLeft className="h-4 w-4" />
        Kembali ke Pembayaran
      </Link>

      <PageHeader
        title="Detail Pembayaran"
        description={`Diterima ${formatDateTime(data.paidAt)}`}
        action={
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" onClick={cetak}>
              <Printer className="h-4 w-4" />
              Cetak Nota
            </Button>
            {data.invoice && (
              <Button onClick={() => navigate(`/invoices/${data.invoice!.id}`)}>
                <FileText className="h-4 w-4" />
                Lihat Invoice
              </Button>
            )}
          </div>
        }
      />

      <div className="grid gap-5 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader title="Rincian" />
          <CardBody>
            <dl className="divide-y divide-ink-100">
              <Baris label="Nominal">
                <span className="text-lg font-bold text-emerald-600">
                  {formatCurrency(data.amount)}
                </span>
              </Baris>
              <Baris label="Metode">{paymentMethodLabel(data.paymentMethod)}</Baris>
              <Baris label="Waktu bayar">{formatDateTime(data.paidAt)}</Baris>
              <Baris label="Tipe">
                {data.invoice ? (
                  <InvoiceTypeBadge type={data.invoice.type} />
                ) : (
                  <span className="text-ink-400">—</span>
                )}
              </Baris>
              <Baris label="Invoice">
                {data.invoice ? (
                  <Link
                    to={`/invoices/${data.invoice.id}`}
                    className="inline-flex items-center gap-1 font-mono text-[13px] text-brand-600 hover:text-brand-700"
                  >
                    {data.invoice.number}
                    <ArrowUpRight className="h-3.5 w-3.5" />
                  </Link>
                ) : (
                  // Bukan sel kosong: pembayaran tanpa invoice berarti tautannya
                  // putus, dan itu perlu terbaca sebagai keadaan yang salah —
                  // bukan sebagai kolom yang kebetulan tidak diisi.
                  <span className="text-amber-600">Invoicenya tidak ditemukan</span>
                )}
              </Baris>
              {data.note && <Baris label="Catatan">{data.note}</Baris>}
            </dl>
          </CardBody>
        </Card>

        <Card>
          <CardHeader title="Member" />
          <CardBody>
            {data.member ? (
              <div className="flex items-center gap-3">
                <Avatar name={data.member.name} />
                <div className="min-w-0">
                  <div className="truncate font-semibold text-ink-900">{data.member.name}</div>
                  {data.member.email && (
                    <div className="truncate text-sm text-ink-500">{data.member.email}</div>
                  )}
                  {data.member.phone && (
                    <div className="text-sm text-ink-500">{data.member.phone}</div>
                  )}
                </div>
              </div>
            ) : (
              <p className="text-sm text-ink-500">Data member tidak ditemukan.</p>
            )}
          </CardBody>
        </Card>
      </div>
    </div>
  )
}

function Baris({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 py-3">
      <dt className="text-sm text-ink-500">{label}</dt>
      <dd className="text-sm font-medium text-ink-900">{children}</dd>
    </div>
  )
}
