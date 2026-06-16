import type { InvoiceWithRelations } from '@/types'
import { renderInvoiceBody } from '../lib/invoiceDocument'

interface Props {
  invoice: InvoiceWithRelations
}

/** On-screen invoice preview — shares the exact markup used for the PDF document. */
export function InvoicePreview({ invoice }: Props) {
  return (
    <div
      className="invoice-preview"
      dangerouslySetInnerHTML={{ __html: renderInvoiceBody(invoice) }}
    />
  )
}
