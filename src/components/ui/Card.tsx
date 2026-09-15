import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/cn'

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('surface', className)} {...props} />
}

interface CardHeaderProps {
  title: ReactNode
  subtitle?: ReactNode
  action?: ReactNode
  className?: string
}

export function CardHeader({ title, subtitle, action, className }: CardHeaderProps) {
  return (
    /*
      Barisnya boleh membungkus, dan slot aksinya boleh mengalah.
      
      Sebelumnya aksi dipasang `flex-shrink-0` di dalam baris `nowrap`, yang
      berarti ia tidak pernah menyempit. Di halaman Konfirmasi Renewal, deretan
      pilihan periode beserta dropdown chapter selebar 360px menembus kartu
      induknya yang hanya 356px dan berhenti di 413px pada layar 390px. Bagian
      yang terpotong tidak bisa dijangkau sama sekali, karena halamannya sendiri
      tidak menggulir mendatar.
      
      Isi aksi umumnya sudah punya `flex-wrap` sendiri. Yang menghalanginya
      membungkus hanyalah lebar tak terbatas yang diberikan `flex-shrink-0`.
      
      Membungkusnya dibatasi ke layar kecil lewat `sm:flex-nowrap`. Tanpa batas
      itu, kartu sempit di layar lebar ikut menurunkan tombolnya ke baris bawah,
      dan itu perubahan tampilan yang tidak diminta siapa pun.
      
      PageHeader pernah kena hal yang sama dan sudah diperbaiki dengan cara ini;
      CardHeader tertinggal.
    */
    <div className={cn('flex flex-wrap items-start justify-between gap-x-4 gap-y-3 px-5 py-4 sm:flex-nowrap', className)}>
      <div className="min-w-0">
        <h3 className="text-[15px] font-semibold text-ink-900">{title}</h3>
        {subtitle && <p className="mt-0.5 text-[13px] text-ink-500">{subtitle}</p>}
      </div>
      {action && <div className="min-w-0 max-w-full">{action}</div>}
    </div>
  )
}

export function CardBody({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('px-5 py-4', className)} {...props} />
}
