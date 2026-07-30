import { X } from 'lucide-react'
import { Input } from './Field'

/**
 * Rentang tanggal "dari – sampai" untuk baris filter.
 *
 * Empat halaman menuliskan markup ini sendiri-sendiri, dan keempatnya punya bug
 * yang sama: dua input berlebar tetap 150px di dalam `flex` tanpa `flex-wrap`
 * tidak muat di layar ponsel, sehingga SELURUH halaman bisa digeser ke samping.
 * Satu komponen berarti satu tempat untuk membuatnya benar.
 *
 * Cara kerjanya di layar sempit: grup mengambil satu baris penuh, labelnya
 * pindah ke baris sendiri, dan kedua input berbagi sisa lebar lewat `flex-1`.
 * `min-w-0` itu yang membuatnya bisa menyusut — tanpa itu input tetap memakai
 * lebar intrinsik kontrol tanggal bawaan browser dan tetap meluber.
 */
export function DateRangeFilter({
  label,
  from,
  to,
  onFrom,
  onTo,
  onReset,
}: {
  label: string
  from: string
  to: string
  onFrom: (value: string) => void
  onTo: (value: string) => void
  /** Bila diisi, tombol reset muncul saat salah satu tanggal terisi. */
  onReset?: () => void
}) {
  const field = 'min-w-0 flex-1 sm:w-[150px] sm:flex-none'

  return (
    <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto">
      {label && <span className="w-full text-[13px] text-ink-500 sm:w-auto">{label}</span>}
      <Input
        type="date"
        value={from}
        max={to || undefined}
        onChange={(e) => onFrom(e.target.value)}
        className={field}
        aria-label={`${label || 'Tanggal'} dari`}
      />
      <span className="text-ink-400">–</span>
      <Input
        type="date"
        value={to}
        min={from || undefined}
        onChange={(e) => onTo(e.target.value)}
        className={field}
        aria-label={`${label || 'Tanggal'} sampai`}
      />
      {onReset && (from || to) && (
        <button
          type="button"
          onClick={onReset}
          className="rounded-lg p-1.5 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-700"
          aria-label={`Reset filter ${label.toLowerCase() || 'tanggal'}`}
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  )
}
