import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  Ban,
  CheckCircle2,
  Copy,
  CreditCard,
  ExternalLink,
  FilePlus2,
  MessageCircle,
  Pencil,
  Send,
  TriangleAlert,
  XCircle,
} from 'lucide-react'
import type { AuditAction, AuditLogEntry, InvoiceWithRelations, PaymentWithInvoice } from '@/types'
import {
  Avatar,
  Badge,
  Button,
  Card,
  CardBody,
  CardHeader,
  Field,
  InvoiceStatusBadge,
  InvoiceTypeBadge,
  LoadingState,
  Modal,
  PageHeader,
  Textarea,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { invoiceService, paymentService } from '@/services'
import { formatCurrency, formatDate, formatDateTime } from '@/lib/format'
import { cn } from '@/lib/cn'

const AUDIT_META: Record<AuditAction, { icon: typeof FilePlus2; tone: string }> = {
  created: { icon: FilePlus2, tone: 'bg-ink-100 text-ink-500' },
  sent: { icon: Send, tone: 'bg-amber-50 text-amber-500' },
  paid: { icon: CheckCircle2, tone: 'bg-emerald-50 text-emerald-500' },
  overdue: { icon: TriangleAlert, tone: 'bg-red-50 text-red-500' },
  cancelled: { icon: XCircle, tone: 'bg-ink-100 text-ink-500' },
  updated: { icon: Pencil, tone: 'bg-blue-50 text-blue-500' },
}

const AUDIT_LABEL: Record<AuditAction, string> = {
  created: 'Invoice dibuat',
  sent: 'Dikirim ke Paper.id',
  paid: 'Pembayaran diterima',
  overdue: 'Jatuh tempo terlewati',
  cancelled: 'Invoice dibatalkan',
  updated: 'Invoice diperbarui',
}

type DialogKind = 'send' | 'paid' | 'cancel' | null

export function InvoiceDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const { toast } = useToast()

  const { data: invoice, loading, reload } = useAsync<InvoiceWithRelations | null>(
    () => invoiceService.getById(id),
    [id],
  )
  const { data: audit, reload: reloadAudit } = useAsync<AuditLogEntry[]>(
    () => invoiceService.getAuditLog(id),
    [id],
  )
  const { data: payments, reload: reloadPayments } = useAsync<PaymentWithInvoice[]>(
    () => paymentService.listByInvoice(id),
    [id],
  )

  const [dialog, setDialog] = useState<DialogKind>(null)
  const [cancelReason, setCancelReason] = useState('')
  const [busy, setBusy] = useState(false)

  if (loading) return <LoadingState label="Memuat invoice…" />
  if (!invoice)
    return (
      <Card>
        <CardBody>
          <p className="py-10 text-center text-ink-500">Invoice tidak ditemukan.</p>
        </CardBody>
      </Card>
    )

  const refresh = () => {
    reload()
    reloadAudit()
    reloadPayments()
  }

  const runAction = async (fn: () => Promise<unknown>, message: string) => {
    setBusy(true)
    try {
      await fn()
      toast(message)
      setDialog(null)
      setCancelReason('')
      refresh()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Aksi gagal.', 'error')
    } finally {
      setBusy(false)
    }
  }

  const copyLink = async () => {
    if (!invoice.paperIdPaymentUrl) return
    await navigator.clipboard.writeText(invoice.paperIdPaymentUrl)
    toast('Link pembayaran disalin.')
  }

  const shareViaWhatsApp = () => {
    if (!invoice.paperIdPaymentUrl) return
    const memberName = invoice.member?.name ?? 'Bapak/Ibu'
    const type = invoice.type === 'registration' ? 'Pendaftaran Member' : 'Renewal Keanggotaan'
    const msg = [
      `Halo ${memberName},`,
      ``,
      `Berikut tagihan *${type}* BNI Grow dari kami:`,
      `📋 No. Invoice: *${invoice.number}*`,
      `💰 Nominal: *${formatCurrency(invoice.amount)}*`,
      `📅 Jatuh Tempo: *${formatDate(invoice.dueDate)}*`,
      ``,
      `Silakan lakukan pembayaran melalui link berikut:`,
      invoice.paperIdPaymentUrl,
      ``,
      `Terima kasih. 🙏`,
    ].join('\n')
    window.open(`https://wa.me/?text=${encodeURIComponent(msg)}`, '_blank')
  }

  const { status } = invoice
  const canSend = status === 'draft'
  const canPay = status === 'sent' || status === 'overdue'
  const canCancel = status !== 'paid' && status !== 'cancelled'

  return (
    <div>
      <PageHeader
        title={invoice.number}
        breadcrumb={
          <button
            onClick={() => navigate('/invoices')}
            className="inline-flex items-center gap-1.5 text-sm text-ink-500 hover:text-ink-800"
          >
            <ArrowLeft className="h-4 w-4" />
            Kembali ke Invoice
          </button>
        }
        description={
          <span className="flex items-center gap-2">
            <InvoiceStatusBadge status={status} />
            <InvoiceTypeBadge type={invoice.type} />
          </span>
        }
        action={
          <>
            {canSend && (
              <Button onClick={() => setDialog('send')}>
                <Send className="h-4 w-4" />
                Kirim ke Paper.id
              </Button>
            )}
            {canPay && (
              <Button onClick={() => setDialog('paid')}>
                <CheckCircle2 className="h-4 w-4" />
                Tandai Lunas
              </Button>
            )}
            {canCancel && (
              <Button variant="danger" onClick={() => setDialog('cancel')}>
                <Ban className="h-4 w-4" />
                Batalkan
              </Button>
            )}
          </>
        }
      />

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
        {/* Left: details */}
        <div className="space-y-5 lg:col-span-2">
          {/* Amount summary */}
          <Card>
            <CardBody className="flex flex-col gap-6 p-6 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <div className="text-sm text-ink-500">Total Tagihan</div>
                <div className="mt-1 text-3xl font-bold tracking-tight text-ink-900">
                  {formatCurrency(invoice.amount)}
                </div>
                {invoice.paidAt && (
                  <div className="mt-1 text-sm text-emerald-600">
                    Dibayar {formatDateTime(invoice.paidAt)}
                  </div>
                )}
              </div>
              <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-brand-50 text-brand-500">
                <CreditCard className="h-7 w-7" />
              </div>
            </CardBody>
          </Card>

          {/* Details */}
          <Card>
            <CardHeader title="Detail Invoice" />
            <div className="grid grid-cols-1 gap-px overflow-hidden rounded-b-2xl bg-ink-100 sm:grid-cols-2">
              <DetailItem label="Nomor Invoice" value={invoice.number} mono />
              <DetailItem label="Tipe" value={invoice.type === 'registration' ? 'Pendaftaran' : 'Renewal'} />
              <DetailItem label="Tanggal Terbit" value={formatDate(invoice.dueDate)} />
              <DetailItem label="Mata Uang" value={invoice.currency} />
              <DetailItem label="Awal Periode" value={formatDate(invoice.periodStart)} />
              <DetailItem label="Akhir Periode" value={formatDate(invoice.periodEnd)} />
              <DetailItem label="Tanggal Dibuat" value={formatDateTime(invoice.createdAt)} />
              <DetailItem label="Terakhir Diperbarui" value={formatDateTime(invoice.updatedAt)} />
            </div>
            {invoice.notes && (
              <div className="border-t border-ink-100 px-5 py-4">
                <div className="text-xs font-medium uppercase tracking-wide text-ink-400">Catatan</div>
                <p className="mt-1 text-sm text-ink-600">{invoice.notes}</p>
              </div>
            )}
            {invoice.status === 'cancelled' && invoice.cancelReason && (
              <div className="border-t border-ink-100 bg-red-50/50 px-5 py-4">
                <div className="text-xs font-medium uppercase tracking-wide text-red-400">Alasan Pembatalan</div>
                <p className="mt-1 text-sm text-red-600">{invoice.cancelReason}</p>
              </div>
            )}
          </Card>

          {/* Paper.id */}
          <Card>
            <CardHeader title="Integrasi Paper.id" />
            <CardBody>
              {invoice.paperIdPaymentUrl ? (
                <div className="space-y-3">
                  <DetailRow label="Paper.id Invoice ID" value={invoice.paperIdInvoiceId ?? '—'} mono />
                  <div>
                    <div className="mb-1.5 text-[13px] font-medium text-ink-700">Link Pembayaran</div>
                    <div className="flex items-center gap-2 rounded-xl border border-ink-200 bg-ink-50 px-3 py-2">
                      <span className="flex-1 truncate font-mono text-[13px] text-ink-600">
                        {invoice.paperIdPaymentUrl}
                      </span>
                      <button
                        onClick={copyLink}
                        className="rounded-lg p-1.5 text-ink-500 hover:bg-white hover:text-brand-500"
                        aria-label="Salin link"
                      >
                        <Copy className="h-4 w-4" />
                      </button>
                      <a
                        href={invoice.paperIdPaymentUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="rounded-lg p-1.5 text-ink-500 hover:bg-white hover:text-brand-500"
                        aria-label="Buka link"
                      >
                        <ExternalLink className="h-4 w-4" />
                      </a>
                    </div>
                    <button
                      onClick={shareViaWhatsApp}
                      className="mt-2 flex w-full items-center justify-center gap-2 rounded-xl bg-[#25D366] py-2.5 text-sm font-semibold text-white transition-opacity hover:opacity-90"
                    >
                      <MessageCircle className="h-4 w-4" />
                      Kirim via WhatsApp
                    </button>
                    <p className="mt-1.5 text-xs text-ink-400">
                      Pesan akan disiapkan otomatis, pilih kontak member di WhatsApp.
                    </p>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col items-start gap-3 rounded-xl border border-dashed border-ink-200 p-5 text-sm text-ink-500">
                  <p>Invoice ini belum dikirim ke Paper.id. Kirim untuk menghasilkan link pembayaran.</p>
                  {canSend && (
                    <Button size="sm" onClick={() => setDialog('send')}>
                      <Send className="h-4 w-4" />
                      Kirim sekarang
                    </Button>
                  )}
                </div>
              )}
            </CardBody>
          </Card>
        </div>

        {/* Right: member + audit */}
        <div className="space-y-5">
          <Card>
            <CardHeader title="Member" />
            <CardBody>
              {invoice.member ? (
                <Link
                  to={`/members/${invoice.member.id}`}
                  className="flex items-center gap-3 rounded-xl p-1 transition-colors hover:bg-ink-50"
                >
                  <Avatar name={invoice.member.name} />
                  <div className="min-w-0">
                    <div className="truncate font-semibold text-ink-900">{invoice.member.name}</div>
                    <div className="truncate text-xs text-ink-400">{invoice.member.email}</div>
                  </div>
                </Link>
              ) : (
                <p className="text-sm text-ink-400">—</p>
              )}
              <div className="mt-4 space-y-2.5 border-t border-ink-100 pt-4 text-sm">
                <DetailRow label="Chapter" value={invoice.chapter?.displayName ?? '—'} />
                <DetailRow label="Telepon" value={invoice.member?.phone ?? '—'} />
              </div>
            </CardBody>
          </Card>

          {payments && payments.length > 0 && (
            <Card>
              <CardHeader title="Pembayaran Diterima" />
              <CardBody>
                <div className="space-y-3">
                  {payments.map((p) => (
                    <div key={p.id} className="flex items-center justify-between gap-3 rounded-xl border border-emerald-100 bg-emerald-50/50 px-4 py-3">
                      <div>
                        <div className="text-sm font-semibold text-emerald-700">
                          {formatCurrency(p.amount)}
                        </div>
                        <div className="mt-0.5 text-xs text-ink-500">
                          {p.paymentMethod?.replace(/_/g, ' ') ?? '—'} · {formatDateTime(p.paidAt)}
                        </div>
                      </div>
                      <Badge tone="green">Lunas</Badge>
                    </div>
                  ))}
                </div>
              </CardBody>
            </Card>
          )}

          <Card>
            <CardHeader title="Riwayat" subtitle="Audit trail perubahan status" />
            <CardBody>
              <ol className="relative space-y-5">
                {audit?.map((entry, i) => {
                  const meta = AUDIT_META[entry.action]
                  const Icon = meta.icon
                  return (
                    <li key={entry.id} className="relative flex gap-3">
                      {i < (audit?.length ?? 0) - 1 && (
                        <span className="absolute left-[15px] top-9 h-[calc(100%-4px)] w-px bg-ink-100" />
                      )}
                      <span className={cn('flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full', meta.tone)}>
                        <Icon className="h-4 w-4" />
                      </span>
                      <div className="min-w-0 pt-0.5">
                        <div className="text-sm font-medium text-ink-800">{AUDIT_LABEL[entry.action]}</div>
                        {entry.notes && <div className="text-[13px] text-ink-500">{entry.notes}</div>}
                        <div className="mt-0.5 text-xs text-ink-400">
                          {entry.actorName} · {formatDateTime(entry.createdAt)}
                        </div>
                      </div>
                    </li>
                  )
                })}
              </ol>
            </CardBody>
          </Card>
        </div>
      </div>

      {/* Dialogs */}
      <Modal
        open={dialog === 'send'}
        onClose={() => setDialog(null)}
        title="Kirim ke Paper.id?"
        description="Invoice akan dikirim ke Paper.id dan link pembayaran akan dibuat. Status menjadi 'Terkirim'."
        footer={
          <>
            <Button variant="outline" onClick={() => setDialog(null)} disabled={busy}>
              Batal
            </Button>
            <Button
              loading={busy}
              onClick={() => runAction(() => invoiceService.send(invoice.id), 'Invoice dikirim ke Paper.id.')}
            >
              <Send className="h-4 w-4" />
              Kirim
            </Button>
          </>
        }
      />

      <Modal
        open={dialog === 'paid'}
        onClose={() => setDialog(null)}
        title="Tandai sebagai lunas?"
        description="Mensimulasikan webhook 'payment.success' dari Paper.id — pembayaran akan dicatat dan status menjadi 'Lunas'."
        footer={
          <>
            <Button variant="outline" onClick={() => setDialog(null)} disabled={busy}>
              Batal
            </Button>
            <Button
              loading={busy}
              onClick={() => runAction(() => invoiceService.markPaid(invoice.id), 'Pembayaran dicatat — invoice lunas.')}
            >
              <CheckCircle2 className="h-4 w-4" />
              Tandai Lunas
            </Button>
          </>
        }
      />

      <Modal
        open={dialog === 'cancel'}
        onClose={() => setDialog(null)}
        title="Batalkan invoice?"
        description="Invoice yang dibatalkan tidak dapat dikembalikan. Berikan alasan pembatalan."
        footer={
          <>
            <Button variant="outline" onClick={() => setDialog(null)} disabled={busy}>
              Batal
            </Button>
            <Button
              variant="danger"
              loading={busy}
              disabled={!cancelReason.trim()}
              onClick={() => runAction(() => invoiceService.cancel(invoice.id, cancelReason.trim()), 'Invoice dibatalkan.')}
            >
              <Ban className="h-4 w-4" />
              Batalkan Invoice
            </Button>
          </>
        }
      >
        <Field label="Alasan pembatalan" required>
          <Textarea
            value={cancelReason}
            onChange={(e) => setCancelReason(e.target.value)}
            placeholder="Mis. member mengundurkan diri sebelum pembayaran."
            autoFocus
          />
        </Field>
      </Modal>
    </div>
  )
}

function DetailItem({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="bg-white px-5 py-4">
      <div className="text-xs font-medium uppercase tracking-wide text-ink-400">{label}</div>
      <div className={cn('mt-1 text-sm font-medium text-ink-800', mono && 'font-mono')}>{value}</div>
    </div>
  )
}

function DetailRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-ink-500">{label}</span>
      <span className={cn('text-right font-medium text-ink-800', mono && 'font-mono text-[13px]')}>{value}</span>
    </div>
  )
}
