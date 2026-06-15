import type { ReactNode, ThHTMLAttributes } from 'react'
import { ChevronsUpDown, ChevronUp, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/cn'

export function Table({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className="w-full overflow-x-auto">
      <table className={cn('w-full border-collapse text-sm', className)}>{children}</table>
    </div>
  )
}

export function THead({ children }: { children: ReactNode }) {
  return <thead>{children}</thead>
}

export function TBody({ children }: { children: ReactNode }) {
  return <tbody className="divide-y divide-ink-100">{children}</tbody>
}

interface ThProps extends ThHTMLAttributes<HTMLTableCellElement> {
  sortable?: boolean
  sortDir?: 'asc' | 'desc' | null
  onSort?: () => void
}

export function Th({ children, className, sortable, sortDir, onSort, ...props }: ThProps) {
  return (
    <th
      className={cn(
        'border-b border-ink-100 px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-ink-400',
        className,
      )}
      {...props}
    >
      {sortable ? (
        <button
          onClick={onSort}
          className="inline-flex items-center gap-1 transition-colors hover:text-ink-600"
        >
          {children}
          {sortDir === 'asc' ? (
            <ChevronUp className="h-3.5 w-3.5" />
          ) : sortDir === 'desc' ? (
            <ChevronDown className="h-3.5 w-3.5" />
          ) : (
            <ChevronsUpDown className="h-3.5 w-3.5 opacity-60" />
          )}
        </button>
      ) : (
        children
      )}
    </th>
  )
}

export function Tr({
  children,
  className,
  onClick,
}: {
  children: ReactNode
  className?: string
  onClick?: () => void
}) {
  return (
    <tr
      onClick={onClick}
      className={cn('transition-colors', onClick && 'cursor-pointer hover:bg-ink-50/70', className)}
    >
      {children}
    </tr>
  )
}

export function Td({ children, className }: { children: ReactNode; className?: string }) {
  return <td className={cn('px-5 py-3.5 align-middle text-ink-700', className)}>{children}</td>
}
