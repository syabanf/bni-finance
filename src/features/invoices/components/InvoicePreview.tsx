import { forwardRef } from 'react'
import type { InvoiceWithRelations } from '@/types'
import { formatCurrency, formatDate } from '@/lib/format'

interface Props {
  invoice: InvoiceWithRelations
}

export const InvoicePreview = forwardRef<HTMLDivElement, Props>(({ invoice }, ref) => {
  const member = invoice.member
  const chapter = invoice.chapter

  return (
    <div
      ref={ref}
      className="invoice-preview bg-white p-10 text-sm text-gray-800 font-sans"
      style={{ minWidth: 600, maxWidth: 800, margin: '0 auto' }}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-8">
        <div>
          <div className="text-2xl font-bold text-red-600 tracking-tight">BNI Finance Hub</div>
          <div className="text-gray-500 text-xs mt-0.5">Payment Platform</div>
        </div>
        <div className="text-right">
          <div className="text-2xl font-bold text-gray-900">INVOICE</div>
          <div className="font-mono text-gray-600 mt-0.5">{invoice.number}</div>
        </div>
      </div>

      {/* Info baris */}
      <div className="grid grid-cols-2 gap-6 mb-8">
        <div>
          <div className="text-xs font-semibold uppercase tracking-wide text-gray-400 mb-2">Kepada</div>
          <div className="font-semibold text-gray-900">{member?.name ?? '—'}</div>
          {member?.company && <div className="text-gray-600">{member.company}</div>}
          {member?.email && <div className="text-gray-500 text-xs mt-0.5">{member.email}</div>}
          {member?.phone && <div className="text-gray-500 text-xs">{member.phone}</div>}
          {chapter && (
            <div className="text-gray-500 text-xs mt-1">{chapter.displayName}</div>
          )}
        </div>
        <div className="text-right space-y-1.5">
          <InfoRow label="Tipe" value={invoice.type === 'registration' ? 'Pendaftaran Member' : 'Renewal Keanggotaan'} />
          <InfoRow label="Tanggal Terbit" value={formatDate(invoice.createdAt)} />
          {invoice.dueDate && <InfoRow label="Jatuh Tempo" value={formatDate(invoice.dueDate)} />}
          <InfoRow label="Periode" value={`${formatDate(invoice.periodStart)} – ${formatDate(invoice.periodEnd)}`} />
        </div>
      </div>

      {/* Tabel item */}
      <table className="w-full mb-8 border-collapse">
        <thead>
          <tr className="bg-gray-100">
            <th className="text-left px-4 py-2.5 text-xs font-semibold uppercase tracking-wide text-gray-500">Deskripsi</th>
            <th className="text-right px-4 py-2.5 text-xs font-semibold uppercase tracking-wide text-gray-500">Jumlah</th>
          </tr>
        </thead>
        <tbody>
          <tr className="border-b border-gray-100">
            <td className="px-4 py-3">
              <div className="font-medium text-gray-900">
                {invoice.type === 'registration' ? 'Biaya Pendaftaran Member BNI' : 'Biaya Renewal Keanggotaan BNI'}
              </div>
              <div className="text-xs text-gray-500 mt-0.5">
                Periode {formatDate(invoice.periodStart)} – {formatDate(invoice.periodEnd)}
              </div>
              {invoice.notes && (
                <div className="text-xs text-gray-400 mt-0.5 italic">{invoice.notes}</div>
              )}
            </td>
            <td className="px-4 py-3 text-right font-semibold text-gray-900">
              {formatCurrency(invoice.amount)}
            </td>
          </tr>
        </tbody>
        <tfoot>
          <tr>
            <td className="px-4 py-3 text-right font-semibold text-gray-700">Total</td>
            <td className="px-4 py-3 text-right text-lg font-bold text-red-600">
              {formatCurrency(invoice.amount)}
            </td>
          </tr>
        </tfoot>
      </table>

      {/* Notes */}
      {invoice.dueDate && (
        <div className="rounded-lg bg-amber-50 border border-amber-100 px-4 py-3 text-xs text-amber-700 mb-8">
          Mohon melakukan pembayaran sebelum <strong>{formatDate(invoice.dueDate)}</strong> untuk menghindari keterlambatan.
        </div>
      )}

      {/* Footer */}
      <div className="border-t border-gray-200 pt-6 text-xs text-gray-400 text-center">
        <div>BNI Finance Hub — Dokumen ini diterbitkan secara otomatis oleh sistem</div>
        <div className="mt-0.5">Terima kasih atas kepercayaan Anda</div>
      </div>
    </div>
  )
})

InvoicePreview.displayName = 'InvoicePreview'

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-xs">
      <span className="text-gray-400">{label}: </span>
      <span className="font-medium text-gray-700">{value}</span>
    </div>
  )
}
