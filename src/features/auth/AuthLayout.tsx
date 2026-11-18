import type { ReactNode } from 'react'
import { CalendarCheck, FileText, Wallet } from 'lucide-react'
import { BniLogo } from '@/components/ui'
import { IdentitasOrganisasi } from './IdentitasOrganisasi'

const SOROTAN = [
  { icon: FileText, teks: 'Invoice dan pengingat otomatis' },
  { icon: Wallet, teks: 'Pembayaran lewat Paper.id' },
  { icon: CalendarCheck, teks: 'Konfirmasi renewal chapter' },
]

interface AuthLayoutProps {
  judul: string
  keterangan: ReactNode
  children: ReactNode
}

/**
 * Bingkai halaman yang bisa dibuka tanpa login: formulir di kiri, panel merek
 * di kanan. Di layar sempit panel merek naik menjadi pita di atas formulir.
 *
 * IdentitasOrganisasi dipasang di sini, bukan di tiap halaman, supaya halaman
 * publik baru tidak bisa lupa memuatnya. Lihat berkas itu untuk sebabnya.
 */
export function AuthLayout({ judul, keterangan, children }: AuthLayoutProps) {
  return (
    <div className="kanvas-kaca relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-8 sm:py-10">
      {/* Bercak warna di belakang kartu: tanpa ini kaca tidak punya apa pun
          untuk diburamkan dan tampak seperti kartu putih biasa. */}
      <div className="pointer-events-none absolute inset-0" aria-hidden>
        <div className="absolute -left-24 top-10 h-80 w-80 rounded-full bg-brand-500/25 blur-3xl" />
        <div className="absolute -right-20 bottom-0 h-96 w-96 rounded-full bg-rose-400/20 blur-3xl" />
        <div className="absolute left-1/3 top-2/3 h-72 w-72 rounded-full bg-sky-400/15 blur-3xl" />
      </div>

      <div className="kaca relative grid w-full max-w-md overflow-hidden rounded-[28px] md:max-w-4xl md:grid-cols-2">
        <div className="flex flex-col justify-center px-6 py-8 sm:px-10 md:py-12">
          <h1 className="text-2xl font-bold tracking-tight text-ink-900">{judul}</h1>
          <p className="mt-1.5 text-sm leading-relaxed text-ink-500">{keterangan}</p>

          <div className="mt-6">{children}</div>

          <IdentitasOrganisasi />
        </div>

        <aside className="relative order-first flex flex-col items-center justify-center overflow-hidden bg-gradient-to-br from-brand-500 via-brand-600 to-brand-900 px-6 py-7 text-center text-white md:order-last md:px-10 md:py-12">
          <div className="pointer-events-none absolute inset-0" aria-hidden>
            <div className="absolute -right-16 -top-16 h-64 w-64 rounded-full bg-white/15 blur-3xl" />
            <div className="absolute -bottom-24 -left-10 h-72 w-72 rounded-full bg-brand-900/60 blur-3xl" />
            <div className="absolute left-[-20%] top-[28%] h-px w-[140%] -rotate-[16deg] bg-gradient-to-r from-transparent via-white/40 to-transparent" />
          </div>

          <div className="relative flex flex-col items-center">
            <BniLogo white className="h-10 w-auto md:h-16" />
            <p className="mt-3 text-[11px] font-semibold uppercase tracking-[0.28em] md:text-sm">
              Business Network International
            </p>
            <p className="mt-1 text-[10px] uppercase tracking-[0.5em] text-white/80 md:text-xs">
              Indonesia
            </p>

            <span className="my-5 hidden h-px w-24 bg-white/50 md:block" />
            <p className="hidden text-2xl font-extrabold italic tracking-tight md:block">Finance Hub</p>

            <ul className="mt-7 hidden w-full max-w-xs space-y-2.5 md:block">
              {SOROTAN.map(({ icon: Icon, teks }) => (
                <li
                  key={teks}
                  className="flex items-center gap-3 rounded-2xl border border-white/20 bg-white/10 px-4 py-3 text-left text-sm font-medium shadow-lg shadow-brand-900/20 backdrop-blur-md"
                >
                  <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-white/15">
                    <Icon className="h-4 w-4" />
                  </span>
                  {teks}
                </li>
              ))}
            </ul>
          </div>
        </aside>
      </div>
    </div>
  )
}
