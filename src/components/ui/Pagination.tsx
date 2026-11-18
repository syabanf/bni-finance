import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/cn'

interface PaginationProps {
  page: number
  pageSize: number
  total: number
  onChange: (page: number) => void
  /** Kata benda barisnya, untuk kalimat "1–25 dari 40 member". */
  satuan: string
  className?: string
}

const tombolPanah =
  'inline-flex h-8 w-8 items-center justify-center rounded-lg text-ink-500 transition-colors hover:bg-ink-100 disabled:cursor-not-allowed disabled:opacity-30'

/** Nomor halaman yang ditampilkan: pertama, terakhir, dan tetangga halaman aktif. */
function nomorTerlihat(page: number, totalPages: number): (number | 'jeda')[] {
  const hasil: (number | 'jeda')[] = []
  for (let p = 1; p <= totalPages; p++) {
    if (p !== 1 && p !== totalPages && Math.abs(p - page) > 1) continue
    const sebelum = hasil[hasil.length - 1]
    if (typeof sebelum === 'number' && p - sebelum > 1) hasil.push('jeda')
    hasil.push(p)
  }
  return hasil
}

/**
 * Kaki daftar: rentang baris yang tampil dan tombol pindah halaman.
 *
 * Tidak merender apa pun untuk daftar kosong; halaman pemanggilnya sudah
 * menampilkan keadaan kosongnya sendiri.
 */
export function Pagination({ page, pageSize, total, onChange, satuan, className }: PaginationProps) {
  if (total === 0) return null
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const awal = (page - 1) * pageSize + 1
  const akhir = Math.min(page * pageSize, total)

  return (
    <div
      className={cn(
        'flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-t border-ink-100 px-5 py-3',
        className,
      )}
    >
      <span className="text-xs text-ink-400">
        {awal}–{akhir} dari {total.toLocaleString('id-ID')} {satuan}
      </span>
      {totalPages > 1 && (
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => onChange(page - 1)}
            disabled={page === 1}
            className={tombolPanah}
            aria-label="Halaman sebelumnya"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          {nomorTerlihat(page, totalPages).map((item, idx) =>
            item === 'jeda' ? (
              <span key={`j${idx}`} className="px-1 text-xs text-ink-400">
                …
              </span>
            ) : (
              <button
                key={item}
                type="button"
                onClick={() => onChange(item)}
                aria-current={page === item ? 'page' : undefined}
                className={cn(
                  'inline-flex h-8 min-w-[2rem] items-center justify-center rounded-lg px-2 text-xs font-medium transition-colors',
                  page === item ? 'bg-brand-500 text-white' : 'text-ink-600 hover:bg-ink-100',
                )}
              >
                {item}
              </button>
            ),
          )}
          <button
            type="button"
            onClick={() => onChange(page + 1)}
            disabled={page === totalPages}
            className={tombolPanah}
            aria-label="Halaman berikutnya"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      )}
    </div>
  )
}
