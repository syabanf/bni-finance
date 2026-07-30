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
        <div className="min-w-0">
          <h1 className="text-2xl font-bold tracking-tight text-ink-900">{title}</h1>
          {description && <p className="mt-1 text-sm text-ink-500">{description}</p>}
        </div>
        {/*
          `flex-shrink-0` tanpa `flex-wrap` berarti deretan aksi tidak pernah
          boleh mengalah: empat tombol di halaman detail invoice memaksa lebar
          melebihi layar tablet dan menggeser seluruh halaman. Sekarang ia boleh
          membungkus ke baris berikutnya — dipakai setiap halaman, jadi satu
          perbaikan di sini berlaku untuk semuanya.
        */}
        {action && (
          <div className="flex min-w-0 flex-wrap items-center gap-2 sm:justify-end">{action}</div>
        )}
      </div>
    </div>
  )
}
