import { cn } from '@/lib/cn'

/**
 * Wordmark logo tiap bank (stylized pakai warna brand resmi).
 * Untuk logo PNG resmi, taruh file di /public/banks/<kode>.png lalu ganti komponen ini.
 */
export function BankLogo({ bank, className }: { bank: string; className?: string }) {
  const base = cn('inline-flex items-center font-extrabold tracking-tight leading-none', className)
  switch (bank.toUpperCase()) {
    case 'BCA':
      return <span className={base} style={{ color: '#0066B3' }}>BCA</span>
    case 'BNI':
      return (
        <span className={base}>
          <span style={{ color: '#EE7203' }}>BNI</span>
          <span style={{ color: '#00857B' }} className="ml-0.5">46</span>
        </span>
      )
    case 'MANDIRI':
      return <span className={cn(base, 'lowercase')} style={{ color: '#063A6E' }}>mandiri</span>
    case 'BRI':
      return <span className={base} style={{ color: '#00529C' }}>BRI</span>
    default:
      return <span className={base}>{bank}</span>
  }
}
