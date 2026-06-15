import type { ReactNode } from 'react'

interface PageHeaderProps {
  title: string
  description?: ReactNode
  action?: ReactNode
  breadcrumb?: ReactNode
}

export function PageHeader({ title, description, action, breadcrumb }: PageHeaderProps) {
  return (
    <div className="mb-6">
      {breadcrumb && <div className="mb-2">{breadcrumb}</div>}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink-900">{title}</h1>
          {description && <p className="mt-1 text-sm text-ink-500">{description}</p>}
        </div>
        {action && <div className="flex flex-shrink-0 items-center gap-2">{action}</div>}
      </div>
    </div>
  )
}
