/**
 * Single source of truth for payment-method labels — so the same method reads
 * identically on the payments list, invoice detail, and reports.
 */
const PAYMENT_METHOD_LABEL: Record<string, string> = {
  virtual_account: 'Virtual Account',
  bank_transfer: 'Transfer Bank',
  qris: 'QRIS',
  cash: 'Tunai',
  paper_id: 'Paper.id',
  other: 'Lainnya',
}

/** Metode yang bisa dipilih saat mencatat atau mengubah pembayaran manual. */
export const MANUAL_METHODS: { value: string; label: string }[] = [
  { value: 'bank_transfer', label: 'Transfer Bank' },
  { value: 'cash', label: 'Tunai' },
  { value: 'qris', label: 'QRIS' },
  { value: 'other', label: 'Lainnya' },
]

/** Human-readable label for a payment-method code. Unknown codes are title-cased. */
export function paymentMethodLabel(method?: string | null): string {
  const key = (method ?? '').trim()
  if (!key) return '—'
  return (
    PAYMENT_METHOD_LABEL[key] ??
    key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
  )
}
